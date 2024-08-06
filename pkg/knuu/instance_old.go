/*
* This file is deprecated.
* Please use the new package knuu instead.
* This file keeps the old functionality of knuu for backward compatibility.
* A global variable is defined, tmpKnuu, which is used to hold the knuu instance.
 */

package knuu

import (
	"context"
	"errors"
	"io"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/instance"
)

type Instance struct {
	instance.Instance
}

type Executor struct {
	*Instance
}

type InstanceState instance.InstanceState

const (
	None InstanceState = iota
	Preparing
	Committed
	Started
	Stopped
	Destroyed
)

// Deprecated: Use the new package knuu instead.
func NewInstance(name string) (*Instance, error) {
	if tmpKnuu == nil {
		return nil, errors.New("tmpKnuu is not initialized")
	}

	i, err := tmpKnuu.NewInstance(name)
	if err != nil {
		return nil, err
	}
	return &Instance{*i}, nil
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetImage(image string) error {
	if tmpKnuu == nil {
		return errors.New("tmpKnuu is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), tmpKnuu.timeout)
	defer cancel()
	return i.Instance.Build().SetImage(ctx, image)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetGitRepo(ctx context.Context, gitContext builder.GitContext) error {
	return i.Instance.Build().SetGitRepo(ctx, gitContext)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetImageInstant(image string) error {
	return i.Instance.Execution().UpgradeImage(context.Background(), image)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetCommand(command ...string) error {
	return i.Instance.Build().SetCommand(command...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetArgs(args ...string) error {
	return i.Instance.Build().SetArgs(args...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddPortTCP(port int) error {
	return i.Instance.Network().AddPortTCP(port)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) PortForwardTCP(port int) (int, error) {
	return i.Instance.Network().PortForwardTCP(context.Background(), port)
}

// AddPortUDP adds a UDP port to the instance
// Deprecated: Use the new package knuu instead.
func (i *Instance) AddPortUDP(port int) error {
	return i.Instance.Network().AddPortUDP(port)
}

// Deprecated: Use the new package knuu instead.
// ExecuteCommand executes a command in the instance
func (i *Instance) ExecuteCommand(command ...string) (string, error) {
	return i.Instance.Execution().ExecuteCommand(context.Background(), command...)
}

// Deprecated: Use the new package knuu instead.
// This function adds a command to the instance while it is in the building phase
func (i *Instance) AddExecuteCommand(command ...string) error {
	return i.Instance.Build().ExecuteCommand(command...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) ExecuteCommandWithContext(ctx context.Context, command ...string) (string, error) {
	return i.Instance.Execution().ExecuteCommand(ctx, command...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddFile(srcPath, dstPath string, chown string) error {
	return i.Instance.Storage().AddFile(srcPath, dstPath, chown)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddFolder(srcPath, dstPath string, chown string) error {
	return i.Instance.Storage().AddFolder(srcPath, dstPath, chown)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddFileBytes(bytes []byte, dest string, chown string) error {
	return i.Instance.Storage().AddFileBytes(bytes, dest, chown)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetUser(user string) error {
	return i.Instance.Build().SetUser(user)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Commit() error {
	return i.Instance.Build().Commit()
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddVolume(path, size string) error {
	return i.Instance.Storage().AddVolume(path, resource.MustParse(size))
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddVolumeWithOwner(path, size string, owner int64) error {
	return i.Instance.Storage().AddVolumeWithOwner(path, resource.MustParse(size), owner)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetMemory(request, limit string) error {
	return i.Instance.Resources().SetMemory(resource.MustParse(request), resource.MustParse(limit))
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetCPU(request string) error {
	return i.Instance.Resources().SetCPU(resource.MustParse(request))
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetEnvironmentVariable(key, value string) error {
	return i.Instance.Build().SetEnvironmentVariable(key, value)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) GetIP() (string, error) {
	return i.Instance.Network().GetIP(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) GetFileBytes(file string) ([]byte, error) {
	return i.Instance.Storage().GetFileBytes(context.Background(), file)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) ReadFileFromRunningInstance(ctx context.Context, filePath string) (io.ReadCloser, error) {
	return i.Instance.Storage().ReadFileFromRunningInstance(ctx, filePath)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddPolicyRule(rule rbacv1.PolicyRule) error {
	return i.Instance.Security().AddPolicyRule(rule)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetLivenessProbe(livenessProbe *v1.Probe) error {
	return i.Instance.Monitoring().SetLivenessProbe(livenessProbe)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetReadinessProbe(readinessProbe *v1.Probe) error {
	return i.Instance.Monitoring().SetReadinessProbe(readinessProbe)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetStartupProbe(startupProbe *v1.Probe) error {
	return i.Instance.Monitoring().SetStartupProbe(startupProbe)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddSidecar(ctx context.Context, sc instance.SidecarManager) error {
	return i.Instance.Sidecars().Add(ctx, sc)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetPrivileged(privileged bool) error {
	return i.Instance.Security().SetPrivileged(privileged)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddCapability(capability string) error {
	return i.Instance.Security().AddCapability(capability)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddCapabilities(capabilities []string) error {
	return i.Instance.Security().AddCapabilities(capabilities)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) StartAsync() error {
	return i.Instance.Execution().StartAsync(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) StartWithoutWait() error {
	return i.Instance.Execution().StartAsync(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Start() error {
	return i.Instance.Execution().Start(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) IsRunning() (bool, error) {
	return i.Instance.Execution().IsRunning(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) WaitInstanceIsRunning() error {
	return i.Instance.Execution().WaitInstanceIsRunning(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) DisableNetwork() error {
	return i.Instance.Network().Disable(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) EnableNetwork() error {
	return i.Instance.Network().Enable(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) NetworkIsDisabled() (bool, error) {
	return i.Instance.Network().IsDisabled(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) WaitInstanceIsStopped() error {
	return i.Instance.Execution().WaitInstanceIsStopped(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Stop() error {
	return i.Instance.Execution().Stop(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Clone() (*Instance, error) {
	newInst, err := i.Instance.Clone()
	if err != nil {
		return nil, err
	}
	return &Instance{Instance: *newInst}, nil
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) CloneWithName(name string) (*Instance, error) {
	newInst, err := i.Instance.CloneWithName(name)
	if err != nil {
		return nil, err
	}
	return &Instance{*newInst}, nil
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) CreateCustomResource(gvr *schema.GroupVersionResource, obj *map[string]interface{}) error {
	return i.Instance.Resources().CreateCustomResource(context.Background(), gvr, obj)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) CustomResourceDefinitionExists(gvr *schema.GroupVersionResource) (bool, error) {
	return i.Instance.Resources().CustomResourceDefinitionExists(context.Background(), gvr)
}

// Deprecated: Use the new package knuu instead.
func NewExecutor() (*Executor, error) {
	return nil, ErrDeprecated
}

// Deprecated: Use the new package knuu instead.
func (e *Executor) Destroy() error {
	return e.Instance.Destroy()
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Destroy() error {
	return i.Instance.Execution().Destroy(context.Background())
}

// Deprecated: Use the new package knuu instead.
func BatchDestroy(instances ...*Instance) error {
	ins := make([]*instance.Instance, len(instances))
	for i, instance := range instances {
		ins[i] = &instance.Instance
	}
	return instance.BatchDestroy(context.Background(), ins...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Labels() map[string]string {
	return i.Instance.Execution().Labels()
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) IsInState(states ...InstanceState) bool {
	statesNew := make([]instance.InstanceState, len(states))
	for i, state := range states {
		statesNew[i] = instance.InstanceState(state)
	}
	return i.Instance.IsInState(statesNew...)
}

func (i *Instance) AddHost(port int) (err error, host string) {
	host, err = i.Instance.Network().AddHost(context.Background(), port)
	return err, host
}
