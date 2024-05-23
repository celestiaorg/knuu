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

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/instance"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func (i *Instance) SetImage(image string) error {
	if tmpKnuu == nil {
		return errors.New("tmpKnuu is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), tmpKnuu.timeout)
	defer cancel()
	return i.Instance.SetImage(ctx, image)
}

func (i *Instance) SetGitRepo(ctx context.Context, gitContext builder.GitContext) error {
	return i.Instance.SetGitRepo(ctx, gitContext)
}

func (i *Instance) SetImageInstant(image string) error {
	return i.Instance.SetImageInstant(context.Background(), image)
}

func (i *Instance) SetCommand(command ...string) error {
	return i.Instance.SetCommand(command...)
}

func (i *Instance) SetArgs(args ...string) error {
	return i.Instance.SetArgs(args...)
}

func (i *Instance) AddPortTCP(port int) error {
	return i.Instance.AddPortTCP(port)
}

func (i *Instance) PortForwardTCP(port int) (int, error) {
	return i.Instance.PortForwardTCP(context.Background(), port)
}

// AddPortUDP adds a UDP port to the instance
func (i *Instance) AddPortUDP(port int) error {
	return i.Instance.AddPortUDP(port)
}

// ExecuteCommand executes a command in the instance
func (i *Instance) ExecuteCommand(command ...string) (string, error) {
	return i.Instance.ExecuteCommand(context.Background(), command...)
}

func (i *Instance) ExecuteCommandWithContext(ctx context.Context, command ...string) (string, error) {
	return i.Instance.ExecuteCommand(ctx, command...)
}

func (i *Instance) AddFile(srcPath, dstPath string, chown string) error {
	return i.Instance.AddFile(srcPath, dstPath, chown)
}

func (i *Instance) AddFolder(srcPath, dstPath string, chown string) error {
	return i.Instance.AddFolder(srcPath, dstPath, chown)
}

func (i *Instance) AddFileBytes(bytes []byte, dest string, chown string) error {
	return i.Instance.AddFileBytes(bytes, dest, chown)
}

func (i *Instance) SetUser(user string) error {
	return i.Instance.SetUser(user)
}

func (i *Instance) Commit() error {
	return i.Instance.Commit()
}

func (i *Instance) AddVolume(path, size string) error {
	return i.Instance.AddVolume(path, size)
}

func (i *Instance) AddVolumeWithOwner(path, size string, owner int64) error {
	return i.Instance.AddVolumeWithOwner(path, size, owner)
}

func (i *Instance) SetMemory(request, limit string) error {
	return i.Instance.SetMemory(request, limit)
}

func (i *Instance) SetCPU(request string) error {
	return i.Instance.SetCPU(request)
}

func (i *Instance) SetEnvironmentVariable(key, value string) error {
	return i.Instance.SetEnvironmentVariable(key, value)
}

func (i *Instance) GetIP() (string, error) {
	return i.Instance.GetIP(context.Background())
}

func (i *Instance) GetFileBytes(file string) ([]byte, error) {
	return i.Instance.GetFileBytes(context.Background(), file)
}

func (i *Instance) ReadFileFromRunningInstance(ctx context.Context, filePath string) (io.ReadCloser, error) {
	return i.Instance.ReadFileFromRunningInstance(ctx, filePath)
}

func (i *Instance) AddPolicyRule(rule rbacv1.PolicyRule) error {
	return i.Instance.AddPolicyRule(rule)
}

func (i *Instance) SetLivenessProbe(livenessProbe *v1.Probe) error {
	return i.Instance.SetLivenessProbe(livenessProbe)
}

func (i *Instance) SetReadinessProbe(readinessProbe *v1.Probe) error {
	return i.Instance.SetReadinessProbe(readinessProbe)
}

func (i *Instance) SetStartupProbe(startupProbe *v1.Probe) error {
	return i.Instance.SetStartupProbe(startupProbe)
}

func (i *Instance) AddSidecar(sidecar *Instance) error {
	return i.Instance.AddSidecar(&sidecar.Instance)
}

func (i *Instance) SetOtelCollectorVersion(version string) error {
	return i.Instance.SetOtelCollectorVersion(version)
}

func (i *Instance) SetOtelEndpoint(port int) error {
	return i.Instance.SetOtelEndpoint(port)
}

func (i *Instance) SetPrometheusEndpoint(port int, jobName, scapeInterval string) error {
	return i.Instance.SetPrometheusEndpoint(port, jobName, scapeInterval)
}

func (i *Instance) SetJaegerEndpoint(grpcPort, thriftCompactPort, thriftHttpPort int) error {
	return i.Instance.SetJaegerEndpoint(grpcPort, thriftCompactPort, thriftHttpPort)
}

func (i *Instance) SetOtlpExporter(endpoint, username, password string) error {
	return i.Instance.SetOtlpExporter(endpoint, username, password)
}
func (i *Instance) SetJaegerExporter(endpoint string) error {
	return i.Instance.SetJaegerExporter(endpoint)
}
func (i *Instance) SetPrometheusExporter(endpoint string) error {
	return i.Instance.SetPrometheusExporter(endpoint)
}

func (i *Instance) SetPrometheusRemoteWriteExporter(endpoint string) error {
	return i.Instance.SetPrometheusRemoteWriteExporter(endpoint)
}
func (i *Instance) SetPrivileged(privileged bool) error {
	return i.Instance.SetPrivileged(privileged)
}
func (i *Instance) AddCapability(capability string) error {
	return i.Instance.AddCapability(capability)
}
func (i *Instance) AddCapabilities(capabilities []string) error {
	return i.Instance.AddCapabilities(capabilities)
}
func (i *Instance) StartAsync() error {
	return i.Instance.StartAsync(context.Background())
}
func (i *Instance) StartWithoutWait() error {
	return i.Instance.StartWithoutWait(context.Background())
}
func (i *Instance) Start() error {
	return i.Instance.Start(context.Background())
}
func (i *Instance) IsRunning() (bool, error) {
	return i.Instance.IsRunning(context.Background())
}
func (i *Instance) WaitInstanceIsRunning() error {
	return i.Instance.WaitInstanceIsRunning(context.Background())
}
func (i *Instance) DisableNetwork() error {
	return i.Instance.DisableNetwork(context.Background())
}
func (i *Instance) SetBandwidthLimit(limit int64) error {
	return i.Instance.SetBandwidthLimit(limit)
}
func (i *Instance) SetLatencyAndJitter(latency, jitter int64) error {
	return i.Instance.SetLatencyAndJitter(latency, jitter)
}
func (i *Instance) SetPacketLoss(packetLoss int32) error {
	return i.Instance.SetPacketLoss(packetLoss)
}
func (i *Instance) EnableNetwork() error {
	return i.Instance.EnableNetwork(context.Background())
}
func (i *Instance) NetworkIsDisabled() (bool, error) {
	return i.Instance.NetworkIsDisabled(context.Background())
}
func (i *Instance) WaitInstanceIsStopped() error {
	return i.Instance.WaitInstanceIsStopped(context.Background())
}
func (i *Instance) Stop() error {
	return i.Instance.Stop(context.Background())
}
func (i *Instance) Clone() (*Instance, error) {
	newInst, err := i.Instance.Clone()
	if err != nil {
		return nil, err
	}
	return &Instance{Instance: *newInst}, nil
}
func (i *Instance) CloneWithName(name string) (*Instance, error) {
	newInst, err := i.Instance.CloneWithName(name)
	if err != nil {
		return nil, err
	}
	return &Instance{*newInst}, nil
}
func (i *Instance) CreateCustomResource(gvr *schema.GroupVersionResource, obj *map[string]interface{}) error {
	return i.Instance.CreateCustomResource(context.Background(), gvr, obj)
}
func (i *Instance) CustomResourceDefinitionExists(gvr *schema.GroupVersionResource) (bool, error) {
	return i.Instance.CustomResourceDefinitionExists(context.Background(), gvr)
}

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

func (e *Executor) Destroy() error {
	return e.Instance.Destroy()
}

func (i *Instance) Destroy() error {
	return i.Instance.Destroy(context.Background())
}

func BatchDestroy(instances ...*Instance) error {
	ins := make([]*instance.Instance, len(instances))
	for i, instance := range instances {
		ins[i] = &instance.Instance
	}
	return instance.BatchDestroy(context.Background(), ins...)
}

func (i *Instance) CreatePool(amount int) (*InstancePool, error) {
	pool, err := i.Instance.NewPool(amount)
	if err != nil {
		return nil, err
	}
	return &InstancePool{*pool}, nil
}

func (i *InstancePool) StartWithoutWait() error {
	return i.InstancePool.StartWithoutWait(context.Background())
}

func (i *InstancePool) Start() error {
	return i.InstancePool.Start(context.Background())
}

func (i *InstancePool) Destroy() error {
	return i.InstancePool.Destroy(context.Background())
}

func (i *InstancePool) WaitInstancePoolIsRunning() error {
	return i.InstancePool.WaitInstancePoolIsRunning(context.Background())
}

func (i *InstancePool) Instances() []*Instance {
	instances := i.InstancePool.Instances()
	newInstances := make([]*Instance, len(instances))
	for i, instance := range instances {
		newInstances[i] = &Instance{*instance}
	}
	return newInstances
}

func (i *Instance) Labels() map[string]string {
	return i.Instance.Labels()
}

func (i *Instance) IsInState(states ...InstanceState) bool {
	statesNew := make([]instance.InstanceState, len(states))
	for i, state := range states {
		statesNew[i] = instance.InstanceState(state)
	}
	return i.Instance.IsInState(statesNew...)
}
