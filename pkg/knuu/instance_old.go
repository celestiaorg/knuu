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

type InstancePool struct {
	instance.InstancePool
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
	return i.Instance.SetImage(ctx, image)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetGitRepo(ctx context.Context, gitContext builder.GitContext) error {
	return i.Instance.SetGitRepo(ctx, gitContext)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetCommand(command ...string) error {
	return i.Instance.SetCommand(command...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetArgs(args ...string) error {
	return i.Instance.SetArgs(args...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddPortTCP(port int) error {
	return i.Instance.AddPortTCP(port)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) PortForwardTCP(port int) (int, error) {
	return i.Instance.PortForwardTCP(context.Background(), port)
}

// AddPortUDP adds a UDP port to the instance
// Deprecated: Use the new package knuu instead.
func (i *Instance) AddPortUDP(port int) error {
	return i.Instance.AddPortUDP(port)
}

// Deprecated: Use the new package knuu instead.
// ExecuteCommand executes a command in the instance
func (i *Instance) ExecuteCommand(command ...string) (string, error) {
	return i.Instance.ExecuteCommand(context.Background(), command...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) ExecuteCommandWithContext(ctx context.Context, command ...string) (string, error) {
	return i.Instance.ExecuteCommand(ctx, command...)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddFile(srcPath, dstPath string, chown string) error {
	return i.Instance.AddFile(srcPath, dstPath, chown)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddFolder(srcPath, dstPath string, chown string) error {
	return i.Instance.AddFolder(srcPath, dstPath, chown)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddFileBytes(bytes []byte, dest string, chown string) error {
	return i.Instance.AddFileBytes(bytes, dest, chown)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetUser(user string) error {
	return i.Instance.SetUser(user)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Commit() error {
	return i.Instance.Commit()
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddVolume(path, size string) error {
	return i.Instance.AddVolume(path, resource.MustParse(size))
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddVolumeWithOwner(path, size string, owner int64) error {
	return i.Instance.AddVolumeWithOwner(path, resource.MustParse(size), owner)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetMemory(request, limit string) error {
	return i.Instance.SetMemory(resource.MustParse(request), resource.MustParse(limit))
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetCPU(request string) error {
	return i.Instance.SetCPU(resource.MustParse(request))
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetEnvironmentVariable(key, value string) error {
	return i.Instance.SetEnvironmentVariable(key, value)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) GetIP() (string, error) {
	return i.Instance.GetIP(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) GetFileBytes(file string) ([]byte, error) {
	return i.Instance.GetFileBytes(context.Background(), file)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) ReadFileFromRunningInstance(ctx context.Context, filePath string) (io.ReadCloser, error) {
	return i.Instance.ReadFileFromRunningInstance(ctx, filePath)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddPolicyRule(rule rbacv1.PolicyRule) error {
	return i.Instance.AddPolicyRule(rule)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetLivenessProbe(livenessProbe *v1.Probe) error {
	return i.Instance.SetLivenessProbe(livenessProbe)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetReadinessProbe(readinessProbe *v1.Probe) error {
	return i.Instance.SetReadinessProbe(readinessProbe)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetStartupProbe(startupProbe *v1.Probe) error {
	return i.Instance.SetStartupProbe(startupProbe)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddSidecar(sidecar *Instance) error {
	return i.Instance.AddSidecar(&sidecar.Instance)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetOtelCollectorVersion(version string) error {
	return i.Instance.SetOtelCollectorVersion(version)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetOtelEndpoint(port int) error {
	return i.Instance.SetOtelEndpoint(port)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetPrometheusEndpoint(port int, jobName, scapeInterval string) error {
	return i.Instance.SetPrometheusEndpoint(port, jobName, scapeInterval)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetJaegerEndpoint(grpcPort, thriftCompactPort, thriftHttpPort int) error {
	return i.Instance.SetJaegerEndpoint(grpcPort, thriftCompactPort, thriftHttpPort)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetOtlpExporter(endpoint, username, password string) error {
	return i.Instance.SetOtlpExporter(endpoint, username, password)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetJaegerExporter(endpoint string) error {
	return i.Instance.SetJaegerExporter(endpoint)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetPrometheusExporter(endpoint string) error {
	return i.Instance.SetPrometheusExporter(endpoint)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetPrometheusRemoteWriteExporter(endpoint string) error {
	return i.Instance.SetPrometheusRemoteWriteExporter(endpoint)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetPrivileged(privileged bool) error {
	return i.Instance.SetPrivileged(privileged)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddCapability(capability string) error {
	return i.Instance.AddCapability(capability)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) AddCapabilities(capabilities []string) error {
	return i.Instance.AddCapabilities(capabilities)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) StartAsync() error {
	return i.Instance.StartAsync(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) StartWithoutWait() error {
	return i.Instance.StartAsync(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Start() error {
	return i.Instance.Start(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) IsRunning() (bool, error) {
	return i.Instance.IsRunning(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) WaitInstanceIsRunning() error {
	return i.Instance.WaitInstanceIsRunning(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) DisableNetwork() error {
	return i.Instance.DisableNetwork(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetBandwidthLimit(limit int64) error {
	return i.Instance.SetBandwidthLimit(limit)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetLatencyAndJitter(latency, jitter int64) error {
	return i.Instance.SetLatencyAndJitter(latency, jitter)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) SetPacketLoss(packetLoss int32) error {
	return i.Instance.SetPacketLoss(packetLoss)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) EnableNetwork() error {
	return i.Instance.EnableNetwork(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) NetworkIsDisabled() (bool, error) {
	return i.Instance.NetworkIsDisabled(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) WaitInstanceIsStopped() error {
	return i.Instance.WaitInstanceIsStopped(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Stop() error {
	return i.Instance.Stop(context.Background())
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
	return i.Instance.CreateCustomResource(context.Background(), gvr, obj)
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) CustomResourceDefinitionExists(gvr *schema.GroupVersionResource) (bool, error) {
	return i.Instance.CustomResourceDefinitionExists(context.Background(), gvr)
}

// Deprecated: Use the new package knuu instead.
func NewExecutor() (*Executor, error) {
	if tmpKnuu == nil {
		return nil, errors.New("tmpKnuu is not initialized")
	}
	e, err := tmpKnuu.NewExecutor(context.Background())
	if err != nil {
		return nil, err
	}
	return &Executor{
		Instance: &Instance{
			Instance: *e.Instance,
		},
	}, nil
}

// Deprecated: Use the new package knuu instead.
func (e *Executor) Destroy() error {
	return e.Instance.Destroy()
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Destroy() error {
	return i.Instance.Destroy(context.Background())
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
func (i *Instance) CreatePool(amount int) (*InstancePool, error) {
	pool, err := i.Instance.NewPool(amount)
	if err != nil {
		return nil, err
	}
	return &InstancePool{*pool}, nil
}

// Deprecated: Use the new package knuu instead.
func (i *InstancePool) StartWithoutWait() error {
	return i.InstancePool.StartWithoutWait(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *InstancePool) Start() error {
	return i.InstancePool.Start(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *InstancePool) Destroy() error {
	return i.InstancePool.Destroy(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *InstancePool) WaitInstancePoolIsRunning() error {
	return i.InstancePool.WaitInstancePoolIsRunning(context.Background())
}

// Deprecated: Use the new package knuu instead.
func (i *InstancePool) Instances() []*Instance {
	instances := i.InstancePool.Instances()
	newInstances := make([]*Instance, len(instances))
	for i, instance := range instances {
		newInstances[i] = &Instance{*instance}
	}
	return newInstances
}

// Deprecated: Use the new package knuu instead.
func (i *Instance) Labels() map[string]string {
	return i.Instance.Labels()
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
	host, err = i.Instance.AddHost(context.Background(), port)
	return err, host
}
