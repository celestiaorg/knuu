package instance

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/names"
)

type storage struct {
	instance *Instance
	volumes  []*k8s.Volume
	files    []*k8s.File
	fsGroup  int64
}

func (i *Instance) Storage() *storage {
	return i.storage
}

// This function can only be called in the state 'Preparing'
func (s *storage) AddFile(src string, dest string, chown string) error {
	if err := s.checkStateForAddingFile(); err != nil {
		return err
	}

	if err := s.validateFileArgs(src, dest, chown); err != nil {
		return err
	}

	if err := s.checkSrcExists(src); err != nil {
		return err
	}

	dstPath, err := s.copyFileToBuildDir(src, dest)
	if err != nil {
		return err
	}

	switch s.instance.state {
	case StatePreparing:
		s.instance.build.addFileToBuilder(src, dest, chown)
		return nil
	case StateCommitted, StateStopped:
		return s.addFileToInstance(dstPath, dest, chown)
	}

	s.instance.Logger.WithFields(logrus.Fields{
		"file":      dest,
		"instance":  s.instance.name,
		"state":     s.instance.state,
		"build_dir": s.instance.build.getBuildDir(),
	}).Debug("added file")
	return nil
}

// AddFolder adds a folder to the instance
// This function can only be called in the state 'Preparing', 'Committed' or 'Stopped'
func (s *storage) AddFolder(src string, dest string, chown string) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrAddingFolderNotAllowed.WithParams(s.instance.state.String())
	}

	if err := s.validateFileArgs(src, dest, chown); err != nil {
		return err
	}

	// check if src exists (should be a folder)
	srcInfo, err := os.Stat(src)
	if os.IsNotExist(err) || !srcInfo.IsDir() {
		return ErrSrcDoesNotExistOrIsNotDirectory.WithParams(src).Wrap(err)
	}

	// iterate over the files/directories in the src
	err = filepath.Walk(src,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// create the destination path
			relPath, err := filepath.Rel(src, path)
			if err != nil {
				return err
			}
			dstPath := filepath.Join(s.instance.build.getBuildDir(), dest, relPath)

			if info.IsDir() {
				// create directory at destination path
				return os.MkdirAll(dstPath, os.ModePerm)
			}
			// copy file to destination path
			return s.AddFile(path, filepath.Join(dest, relPath), chown)
		})

	if err != nil {
		return ErrCopyingFolderToInstance.WithParams(src, s.instance.name).Wrap(err)
	}

	s.instance.Logger.WithFields(logrus.Fields{
		"folder":    dest,
		"instance":  s.instance.name,
		"state":     s.instance.state,
		"build_dir": s.instance.build.getBuildDir(),
	}).Debug("added folder")
	return nil
}

// AddFileBytes adds a file with the given content to the instance
// This function can only be called in the state 'Preparing'
func (s *storage) AddFileBytes(bytes []byte, dest string, chown string) error {
	if err := s.checkStateForAddingFile(); err != nil {
		return err
	}

	tmpfile, err := os.CreateTemp("", "temp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(bytes); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}

	// use AddFile to copy the temp file to the destination
	return s.AddFile(tmpfile.Name(), dest, chown)
}

// AddVolume adds a volume to the instance
// The owner of the volume is set to 0, if you want to set a custom owner use AddVolumeWithOwner
// This function can only be called in the states 'Preparing', 'Committed' and 'Stopped'
func (s *storage) AddVolume(path string, size resource.Quantity) error {
	// temporary feat, we will remove it once we can add multiple volumes
	if len(s.volumes) > 0 {
		s.instance.Logger.WithFields(logrus.Fields{
			"instance": s.instance.name,
			"volumes":  len(s.volumes),
		}).Debug("maximum volumes exceeded")
		return ErrMaximumVolumesExceeded.WithParams(s.instance.name)
	}
	return s.AddVolumeWithOwner(path, size, 0)
}

// AddVolumeWithOwner adds a volume to the instance with the given owner
// This function can only be called in the states 'Preparing', 'Committed' and 'Stopped'
func (s *storage) AddVolumeWithOwner(path string, size resource.Quantity, owner int64) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrAddingVolumeNotAllowed.WithParams(s.instance.state.String())
	}
	// temporary feat, we will remove it once we can add multiple volumes
	if len(s.volumes) > 0 {
		s.instance.Logger.WithFields(logrus.Fields{
			"instance": s.instance.name,
			"volumes":  len(s.volumes),
		}).Debug("maximum volumes exceeded")
		return ErrMaximumVolumesExceeded.WithParams(s.instance.name)
	}
	volume := s.instance.K8sClient.NewVolume(path, size, owner)
	s.volumes = append(s.volumes, volume)
	s.instance.Logger.WithFields(logrus.Fields{
		"volume":   path,
		"size":     size.String(),
		"owner":    owner,
		"instance": s.instance.name,
	}).Debug("added volume")
	return nil
}

// GetFileBytes returns the content of the given file
// This function can only be called in the states 'Preparing' and 'Committed'
func (s *storage) GetFileBytes(ctx context.Context, file string) ([]byte, error) {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStarted) {
		return nil, ErrGettingFileNotAllowed.WithParams(s.instance.state.String())
	}

	if s.instance.state != StateStarted {
		bytes, err := s.readFileFromImage(ctx, file)
		if err != nil {
			return nil, ErrGettingFile.WithParams(file, s.instance.name).Wrap(err)
		}
		return bytes, nil
	}

	rc, err := s.ReadFileFromRunningInstance(ctx, file)
	if err != nil {
		return nil, ErrReadingFile.WithParams(file, s.instance.name).Wrap(err)
	}

	defer rc.Close()
	return io.ReadAll(rc)
}

func (s *storage) ReadFileFromRunningInstance(ctx context.Context, filePath string) (io.ReadCloser, error) {
	if !s.instance.IsInState(StateStarted) {
		return nil, ErrReadingFileNotAllowed.WithParams(s.instance.state.String())
	}

	// Not the best solution, we need to find a better one.
	// Tested with a 110MB+ file and it worked.
	fileContent, err := s.instance.execution.ExecuteCommand(ctx, "cat", filePath)
	if err != nil {
		return nil, ErrReadingFileFromInstance.WithParams(filePath, s.instance.name).Wrap(err)
	}
	return io.NopCloser(strings.NewReader(fileContent)), nil
}

func (s *storage) checkSrcExists(src string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return ErrSrcDoesNotExist.WithParams(src).Wrap(err)
	}
	return nil
}

// validateFileArgs validates the file arguments
func (s *storage) validateFileArgs(src, dest, chown string) error {
	if src == "" {
		return ErrSrcMustBeSet
	}
	if dest == "" {
		return ErrDestMustBeSet
	}
	if chown == "" {
		return ErrChownMustBeSet
	}

	// validate chown format
	if !strings.Contains(chown, ":") || len(strings.Split(chown, ":")) != 2 {
		return ErrChownMustBeInFormatUserGroup
	}
	return nil
}

func (s *storage) copyFileToBuildDir(src, dest string) (string, error) {
	dstPath := filepath.Join(s.instance.build.getBuildDir(), dest)
	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return "", ErrCreatingDirectory.Wrap(err)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return "", ErrFailedToCreateDestFile.WithParams(dstPath).Wrap(err)
	}
	defer dst.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return "", ErrFailedToOpenSrcFile.WithParams(src).Wrap(err)
	}
	defer srcFile.Close()

	if _, err := io.Copy(dst, srcFile); err != nil {
		return "", ErrFailedToCopyFile.WithParams(src, dstPath).Wrap(err)
	}

	return dstPath, nil
}

func (s *storage) addFileToInstance(dstPath, dest, chown string) error {
	srcInfo, err := os.Stat(dstPath)
	if os.IsNotExist(err) || srcInfo.IsDir() {
		return ErrSrcDoesNotExistOrIsDirectory.WithParams(dstPath).Wrap(err)
	}

	file := s.instance.K8sClient.NewFile(dstPath, dest)
	parts := strings.Split(chown, ":")
	if len(parts) != 2 {
		return ErrInvalidFormat
	}

	group, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return ErrFailedToConvertToInt64.Wrap(err)
	}

	if s.fsGroup != 0 && s.fsGroup != group {
		return ErrAllFilesMustHaveSameGroup
	}
	s.fsGroup = group
	s.files = append(s.files, file)
	return nil
}

// checkStateForAddingFile checks if the current state allows adding a file
func (s *storage) checkStateForAddingFile() error {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrAddingFileNotAllowed.WithParams(s.instance.state.String())
	}
	return nil
}

// deployVolume deploys the volume for the instance
func (s *storage) deployVolume(ctx context.Context) error {
	totalSize := resource.Quantity{}
	for _, volume := range s.volumes {
		totalSize.Add(volume.Size)
	}
	s.instance.K8sClient.CreatePersistentVolumeClaim(ctx, s.instance.name, s.instance.execution.Labels(), totalSize)
	s.instance.Logger.WithFields(logrus.Fields{
		"total_size": totalSize.String(),
		"instance":   s.instance.name,
	}).Debug("deployed persistent volume")

	return nil
}

// destroyVolume destroys the volume for the instance
func (s *storage) destroyVolume(ctx context.Context) error {
	err := s.instance.K8sClient.DeletePersistentVolumeClaim(ctx, s.instance.name)
	if err != nil {
		return ErrFailedToDeletePersistentVolumeClaim.Wrap(err)
	}
	s.instance.Logger.WithField("instance", s.instance.name).Debug("destroyed persistent volume")
	return nil
}

// deployFiles deploys the files for the instance
func (s *storage) deployFiles(ctx context.Context) error {
	data := map[string]string{}

	for i, file := range s.files {
		// read out file content and assign to variable
		srcFile, err := os.Open(file.Source)
		if err != nil {
			return ErrFailedToOpenFile.Wrap(err)
		}
		defer srcFile.Close()

		fileContentBytes, err := io.ReadAll(srcFile)
		if err != nil {
			return ErrFailedToReadFile.Wrap(err)
		}

		var (
			fileContent = string(fileContentBytes)
			keyName     = fmt.Sprintf("%d", i)
		)

		data[keyName] = fileContent
	}

	// If the configmap already exists, we update it
	// This ensures long-running tests and image upgrade tests function correctly.
	_, err := s.instance.K8sClient.CreateOrUpdateConfigMap(ctx, s.instance.name, s.instance.execution.Labels(), data)
	if err != nil {
		return ErrFailedToCreateConfigMap.Wrap(err)
	}

	s.instance.Logger.WithField("configmap", s.instance.name).Debug("deployed configmap")
	return nil
}

// destroyFiles destroys the files for the instance
func (s *storage) destroyFiles(ctx context.Context) error {
	if err := s.instance.K8sClient.DeleteConfigMap(ctx, s.instance.name); err != nil {
		return ErrFailedToDeleteConfigMap.Wrap(err)
	}

	s.instance.Logger.WithField("configmap", s.instance.name).Debug("destroyed configmap")
	return nil
}

func (s *storage) readFileFromImage(ctx context.Context, filePath string) ([]byte, error) {
	// Another way to implement this is to download all the layers of the image and then
	// extract the file from them, but it seems hacky and will run on the user's machine.
	// Therefore, we will use the tmp instance to get the file from the image

	tmpName, err := names.NewRandomK8("tmp-dl")
	if err != nil {
		return nil, err
	}

	ti, err := New(tmpName, s.instance.SystemDependencies)
	if err != nil {
		return nil, err
	}
	if err := ti.build.SetImage(ctx, s.instance.build.ImageName()); err != nil {
		return nil, err
	}

	if err := ti.build.SetStartCommand("sleep", "infinity"); err != nil {
		return nil, err
	}

	if err := ti.build.Commit(ctx); err != nil {
		return nil, err
	}

	if err := ti.execution.Start(ctx); err != nil {
		return nil, err
	}
	defer func() {
		if err := ti.execution.Destroy(ctx); err != nil {
			ti.Logger.Errorf("failed to destroy tmp instance %s: %v", ti.name, err)
		}
	}()

	output, err := ti.execution.ExecuteCommand(ctx, "cat", filePath)
	if err != nil {
		return nil, err
	}
	return []byte(output), nil
}

func (s *storage) clone() *storage {
	if s == nil {
		return nil
	}

	volumesCopy := make([]*k8s.Volume, len(s.volumes))
	for i, v := range s.volumes {
		if v != nil {
			volumeCopy := *v
			volumesCopy[i] = &volumeCopy
		}
	}

	filesCopy := make([]*k8s.File, len(s.files))
	for i, f := range s.files {
		if f != nil {
			fileCopy := *f
			filesCopy[i] = &fileCopy
		}
	}

	return &storage{
		instance: nil,
		volumes:  volumesCopy,
		files:    filesCopy,
		fsGroup:  s.fsGroup,
	}
}
