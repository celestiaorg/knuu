package instance

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/container"
)

const buildDirBase = "/tmp/knuu"

type build struct {
	instance        *Instance
	imageName       string
	imagePullPolicy v1.PullPolicy
	builderFactory  *container.BuilderFactory
	command         []string
	args            []string
	env             map[string]string
	imageCache      *sync.Map
}

func (i *Instance) Build() *build {
	return i.build
}

func (b *build) ImageName() string {
	return b.imageName
}

func (b *build) ImagePullPolicy() v1.PullPolicy {
	return b.imagePullPolicy
}

func (b *build) SetImagePullPolicy(pullPolicy v1.PullPolicy) {
	b.imagePullPolicy = pullPolicy
}

// SetImage sets the image of the instance.
// It is only allowed in the 'None' and 'Preparing' states.
func (b *build) SetImage(ctx context.Context, image string) error {
	if !b.instance.IsInState(StateNone, StatePreparing, StateStopped) {
		if b.instance.sidecars.IsSidecar() {
			return ErrSettingImageNotAllowedForSidecarsStarted
		}
		return ErrSettingImageNotAllowed.WithParams(b.instance.state.String())
	}

	// Use the builder to build a new image
	factory, err := container.NewBuilderFactory(image, b.getBuildDir(), b.instance.ImageBuilder)
	if err != nil {
		return ErrCreatingBuilder.Wrap(err)
	}
	b.builderFactory = factory

	b.instance.SetState(StatePreparing)
	return nil
}

// SetGitRepo builds the image from the given git repo, pushes it
// to the registry under the given name and sets the image of the instance.
func (b *build) SetGitRepo(ctx context.Context, gitContext builder.GitContext) error {
	if !b.instance.IsState(StateNone) {
		return ErrSettingGitRepo.WithParams(b.instance.state.String())
	}

	bCtx, err := gitContext.BuildContext()
	if err != nil {
		return ErrGettingBuildContext.Wrap(err)
	}
	imageName, err := builder.DefaultImageName(bCtx)
	if err != nil {
		return ErrGettingImageName.Wrap(err)
	}

	factory, err := container.NewBuilderFactory(imageName, b.getBuildDir(), b.instance.ImageBuilder)
	if err != nil {
		return ErrCreatingBuilder.Wrap(err)
	}
	b.builderFactory = factory
	b.imageName = imageName
	b.instance.SetState(StatePreparing)

	return b.builderFactory.BuildImageFromGitRepo(ctx, gitContext, imageName)
}

// SetStartCommand sets the command to run in the instance
// This function can only be called when the instance is in state 'Preparing' or 'Committed'
func (b *build) SetStartCommand(command ...string) error {
	if !b.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrSettingCommand.WithParams(b.instance.state.String())
	}
	b.command = command
	return nil
}

// SetArgs sets the arguments passed to the instance
// This function can only be called in the states 'Preparing' or 'Committed'
func (b *build) SetArgs(args ...string) error {
	if !b.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrSettingArgsNotAllowed.WithParams(b.instance.state.String())
	}
	b.args = args
	return nil
}

// ExecuteCommand executes the given command in the instance once it starts
// This function can only be called in the states 'Preparing'
func (b *build) ExecuteCommand(command ...string) error {
	if b.instance.state != StatePreparing {
		return ErrAddingCommandNotAllowed.WithParams(b.instance.state.String())
	}

	b.builderFactory.AddCmdToBuilder(command)
	return nil
}

// SetUser sets the user for the instance
// This function can only be called in the state 'Preparing'
func (b *build) SetUser(user string) error {
	if !b.instance.IsState(StatePreparing) {
		return ErrSettingUserNotAllowed.WithParams(b.instance.state.String())
	}

	b.builderFactory.SetUser(user)
	b.instance.Logger.WithFields(logrus.Fields{
		"instance": b.instance.name,
		"user":     user,
	}).Debugf("Set user for instance")
	return nil
}

// Commit commits the instance
// This function can only be called in the state 'Preparing'
func (b *build) Commit(ctx context.Context) error {
	if !b.instance.IsState(StatePreparing) {
		return ErrCommittingNotAllowed.WithParams(b.instance.state.String())
	}

	if !b.builderFactory.Changed() {
		b.imageName = b.builderFactory.ImageNameFrom()
		b.instance.Logger.WithFields(logrus.Fields{
			"instance": b.instance.name,
			"image":    b.imageName,
		}).Debugf("no need to build and push image for instance")

		b.instance.SetState(StateCommitted)
		return nil
	}

	// Generate a hash for the current image
	imageHash, err := b.builderFactory.GenerateImageHash()
	if err != nil {
		return ErrGeneratingImageHash.Wrap(err)
	}

	imageName, err := getImageRegistry(imageHash)
	if err != nil {
		return ErrGettingImageRegistry.Wrap(err)
	}

	// Check if the generated image hash already exists in the cache, otherwise, we build it.
	cachedImageName, exists := b.checkImageHashInCache(imageHash)
	if exists {
		b.imageName = cachedImageName

		b.instance.Logger.WithFields(logrus.Fields{
			"instance": b.instance.name,
			"image":    b.imageName,
		}).Debugf("using cached image for instance")

		b.instance.SetState(StateCommitted)
		return nil
	}

	b.instance.Logger.WithFields(logrus.Fields{
		"instance": b.instance.name,
	}).Debugf("cannot use any cached image for instance")
	err = b.builderFactory.PushBuilderImage(ctx, imageName)
	if err != nil {
		return ErrPushingImage.WithParams(b.instance.name).Wrap(err)
	}
	b.updateImageCacheWithHash(imageHash, imageName)
	b.imageName = imageName

	b.instance.Logger.WithFields(logrus.Fields{
		"instance": b.instance.name,
		"image":    b.imageName,
	}).Debugf("pushed new image for instance")

	b.instance.SetState(StateCommitted)
	return nil
}

// getImageRegistry returns the name of the temporary image registry
func getImageRegistry(imageName string) (string, error) {
	if imageName == "" {
		// If not already set, generate a random name using ttl.sh
		uuid, err := uuid.NewRandom()
		if err != nil {
			return "", fmt.Errorf("error generating UUID: %w", err)
		}
		imageName = uuid.String()
	}
	return fmt.Sprintf("ttl.sh/%s:24h", imageName), nil
}

// getBuildDir returns the build directory for the instance
func (b *build) getBuildDir() string {
	return filepath.Join(buildDirBase, b.instance.name)
}

// addFileToBuilder adds a file to the builder
func (b *build) addFileToBuilder(src, dest, chown string) {
	// dest is the same as src here, as we copy the file to the build dir with the subfolder structure of dest
	_ = src
	b.builderFactory.AddToBuilder(dest, dest, chown)
}

// SetEnvironmentVariable sets the given environment variable in the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (b *build) SetEnvironmentVariable(key, value string) error {
	if !b.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrSettingEnvNotAllowed.WithParams(b.instance.state.String())
	}
	b.instance.Logger.WithFields(logrus.Fields{
		"instance": b.instance.name,
		"key":      key,
		// value is not logged to avoid leaking sensitive information
	}).Debugf("Setting environment variable")

	if b.instance.state == StatePreparing {
		b.builderFactory.SetEnvVar(key, value)
		return nil
	}
	b.env[key] = value
	return nil
}

// checkImageHashInCache checks if the given image hash exists in the cache.
func (b *build) checkImageHashInCache(imageHash string) (string, bool) {
	value, exists := b.imageCache.Load(imageHash)
	imageName, ok := value.(string)
	if !ok {
		return "", false
	}
	return imageName, exists
}

// updateImageCacheWithHash adds or updates the image cache with the given hash and image name.
func (b *build) updateImageCacheWithHash(imageHash, imageName string) {
	b.imageCache.Store(imageHash, imageName)
}

func (b *build) clone() *build {
	if b == nil {
		return nil
	}

	commandCopy := make([]string, len(b.command))
	copy(commandCopy, b.command)

	argsCopy := make([]string, len(b.args))
	copy(argsCopy, b.args)

	envCopy := make(map[string]string, len(b.env))
	for k, v := range b.env {
		envCopy[k] = v
	}

	var imageCacheClone sync.Map
	// Clone the imageCache if it exists
	if b.imageCache != nil {
		b.imageCache.Range(func(key, value interface{}) bool {
			// Copy each key-value pair to the new imageCacheClone
			// This ensures a deep copy of the map structure, but not of the values themselves
			imageCacheClone.Store(key, value)
			return true
		})
	}

	// Return the deep copied build
	return &build{
		instance:  nil,
		imageName: b.imageName,

		//TODO: This does not create a deep copy of the builderFactory. Implement it in another PR
		builderFactory: b.builderFactory,

		command:    commandCopy,
		args:       argsCopy,
		env:        envCopy,
		imageCache: &imageCacheClone,
	}
}
