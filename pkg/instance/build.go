package instance

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/container"
)

const buildDirBase = "/tmp/knuu"

type build struct {
	instance       *Instance
	imageName      string
	builderFactory *container.BuilderFactory
	command        []string
	args           []string
	env            map[string]string
	imageCache     *sync.Map
}

func (i *Instance) Build() *build {
	return i.build
}

func (b *build) ImageName() string {
	return b.imageName
}

// SetImage sets the image of the instance.
// It is only allowed in the 'None' and 'Preparing' states.
func (b *build) SetImage(ctx context.Context, image string) error {
	if !b.instance.IsInState(StateNone, StatePreparing) {
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

// SetCommand sets the command to run in the instance
// This function can only be called when the instance is in state 'Preparing' or 'Committed'
func (b *build) SetCommand(command ...string) error {
	if !b.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingCommand.WithParams(b.instance.state.String())
	}
	b.command = command
	return nil
}

// SetArgs sets the arguments passed to the instance
// This function can only be called in the states 'Preparing' or 'Committed'
func (b *build) SetArgs(args ...string) error {
	if !b.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingArgsNotAllowed.WithParams(b.instance.state.String())
	}
	b.args = args
	return nil
}

// ExecuteCommand executes the given command in the instance once it starts
// This function can only be called in the states 'Preparing'
func (b *build) ExecuteCommand(command ...string) error {
	if !b.instance.IsInState(StatePreparing) {
		return ErrExecutingCommandNotAllowed.WithParams(b.instance.state.String())
	}

	_, err := b.builderFactory.ExecuteCmdInBuilder(command)
	if err != nil {
		return ErrExecutingCommandInInstance.WithParams(command, b.instance.name).Wrap(err)
	}
	return nil
}

// SetUser sets the user for the instance
// This function can only be called in the state 'Preparing'
func (b *build) SetUser(user string) error {
	if !b.instance.IsState(StatePreparing) {
		return ErrSettingUserNotAllowed.WithParams(b.instance.state.String())
	}

	if err := b.builderFactory.SetUser(user); err != nil {
		return ErrSettingUser.WithParams(user, b.instance.name).Wrap(err)
	}
	b.instance.Logger.Debugf("Set user '%s' for instance '%s'", user, b.instance.name)
	return nil
}

// Commit commits the instance
// This function can only be called in the state 'Preparing'
func (b *build) Commit() error {
	if !b.instance.IsState(StatePreparing) {
		return ErrCommittingNotAllowed.WithParams(b.instance.state.String())
	}

	if !b.builderFactory.Changed() {
		b.imageName = b.builderFactory.ImageNameFrom()
		b.instance.Logger.Debugf("No need to build and push image for instance '%s'", b.instance.name)

		b.instance.SetState(StateCommitted)
		return nil
	}

	//TODO: To speed up the process, the image name could be dependent on the hash of the image
	imageName, err := b.getImageRegistry()
	if err != nil {
		return ErrGettingImageRegistry.Wrap(err)
	}

	// Generate a hash for the current image
	imageHash, err := b.builderFactory.GenerateImageHash()
	if err != nil {
		return ErrGeneratingImageHash.Wrap(err)
	}

	// Check if the generated image hash already exists in the cache, otherwise, we build it.
	cachedImageName, exists := b.checkImageHashInCache(imageHash)
	if exists {
		b.imageName = cachedImageName
		b.instance.Logger.Debugf("Using cached image for instance '%s'", b.instance.name)
	} else {
		b.instance.Logger.Debugf("Cannot use any cached image for instance '%s'", b.instance.name)
		err = b.builderFactory.PushBuilderImage(imageName)
		if err != nil {
			return ErrPushingImage.WithParams(b.instance.name).Wrap(err)
		}
		b.updateImageCacheWithHash(imageHash, imageName)
		b.imageName = imageName
		b.instance.Logger.Debugf("Pushed new image for instance '%s'", b.instance.name)
	}

	b.instance.SetState(StateCommitted)
	return nil
}

// getImageRegistry returns the name of the temporary image registry
func (b *build) getImageRegistry() (string, error) {
	if b.imageName != "" {
		return b.imageName, nil
	}
	// If not already set, generate a random name using ttl.sh
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("error generating UUID: %w", err)
	}
	imageName := fmt.Sprintf("ttl.sh/%s:24h", uuid.String())
	return imageName, nil
}

// getBuildDir returns the build directory for the instance
func (b *build) getBuildDir() string {
	return filepath.Join(buildDirBase, b.instance.k8sName)
}

// addFileToBuilder adds a file to the builder
func (b *build) addFileToBuilder(src, dest, chown string) error {
	_ = src
	// dest is the same as src here, as we copy the file to the build dir with the subfolder structure of dest
	err := b.builderFactory.AddToBuilder(dest, dest, chown)
	if err != nil {
		return ErrAddingFileToInstance.WithParams(dest, b.instance.name).Wrap(err)
	}
	return nil
}

// SetEnvironmentVariable sets the given environment variable in the instance
// This function can only be called in the states 'Preparing' and 'Committed'
func (b *build) SetEnvironmentVariable(key, value string) error {
	if !b.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingEnvNotAllowed.WithParams(b.instance.state.String())
	}
	b.instance.Logger.Debugf("Setting environment variable '%s' in instance '%s'", key, b.instance.name)
	if b.instance.state == StatePreparing {
		return b.builderFactory.SetEnvVar(key, value)
	}

	b.env[key] = value
	return nil
}

// setImageWithGracePeriod sets the image of the instance with a grace period
func (b *build) setImageWithGracePeriod(ctx context.Context, imageName string, gracePeriod time.Duration) error {
	b.imageName = imageName

	var gracePeriodInSecondsPtr *int64
	if gracePeriod != 0 {
		gpInSeconds := int64(gracePeriod.Seconds())
		gracePeriodInSecondsPtr = &gpInSeconds
	}
	_, err := b.instance.K8sClient.ReplaceReplicaSetWithGracePeriod(ctx, b.instance.execution.prepareReplicaSetConfig(), gracePeriodInSecondsPtr)
	if err != nil {
		return ErrReplacingPod.Wrap(err)
	}

	if err := b.instance.execution.WaitInstanceIsRunning(ctx); err != nil {
		return ErrWaitingInstanceIsRunning.Wrap(err)
	}

	return nil
}

// imageCache maps image hash values to image names

// checkImageHashInCache checks if the given image hash exists in the cache.
func (b *build) checkImageHashInCache(imageHash string) (string, bool) {
	imageName, exists := b.imageCache.Load(imageHash)
	return imageName.(string), exists
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
