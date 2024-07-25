package observability

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrCreatingOtelAgentInstance   = errors.New("CreatingOtelAgentInstance", "error creating otel agent instance")
	ErrSettingOtelAgentImage       = errors.New("SettingOtelAgentImage", "error setting image for otel agent instance")
	ErrAddingOtelAgentPort         = errors.New("AddingOtelAgentPort", "error adding Otel Agent port")
	ErrCommittingOtelAgentInstance = errors.New("CommittingOtelAgentInstance", "error committing otel agent instance")
	ErrSettingOtelAgentEnv         = errors.New("SettingOtelAgentEnv", "error setting environment variable for otel agent instance")
	ErrSettingOtelAgentCPU         = errors.New("SettingOtelAgentCPU", "error setting CPU for otel agent instance")
	ErrSettingOtelAgentMemory      = errors.New("SettingOtelAgentMemory", "error setting memory for otel agent instance")
	ErrMarshalingYAML              = errors.New("MarshalingYAML", "error marshaling yaml")
	ErrAddingOtelAgentConfigFile   = errors.New("AddingOtelAgentConfigFile", "error adding otel agent config file")
	ErrSettingOtelAgentCommand     = errors.New("SettingOtelAgentCommand", "error setting otel agent command")
	ErrObsyInstanceNotInitialized  = errors.New("ObsyInstanceNotInitialized", "obsy instance not initialized")
	ErrSettingNotAllowed           = errors.New("SettingNotAllowed", "setting %s is only allowed in state 'None'. Current state is '%s'")
	ErrAddingVolume                = errors.New("AddingVolume", "error adding volume")
)
