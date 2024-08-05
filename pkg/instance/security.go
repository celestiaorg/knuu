package instance

import v1 "k8s.io/api/core/v1"

// represents the security settings for a container
type security struct {
	instance *Instance

	// Privileged indicates whether the container should be run in privileged mode
	privileged bool

	// CapabilitiesAdd is the list of capabilities to add to the container
	capabilitiesAdd []string
}

func (i *Instance) Security() *security {
	return i.security
}

// SetPrivileged sets the privileged status for the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (s *security) SetPrivileged(privileged bool) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingPrivilegedNotAllowed.WithParams(s.instance.state.String())
	}
	s.privileged = privileged
	s.instance.Logger.Debugf("Set privileged to '%t' for instance '%s'", privileged, s.instance.name)
	return nil
}

// AddCapability adds a capability to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (s *security) AddCapability(capability string) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingCapabilityNotAllowed.WithParams(s.instance.state.String())
	}
	s.capabilitiesAdd = append(s.capabilitiesAdd, capability)
	s.instance.Logger.Debugf("Added capability '%s' to instance '%s'", capability, s.instance.name)
	return nil
}

// AddCapabilities adds multiple capabilities to the instance
// This function can only be called in the state 'Preparing' or 'Committed'
func (s *security) AddCapabilities(capabilities []string) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrAddingCapabilitiesNotAllowed.WithParams(s.instance.state.String())
	}
	for _, capability := range capabilities {
		s.capabilitiesAdd = append(s.capabilitiesAdd, capability)
		s.instance.Logger.Debugf("Added capability '%s' to instance '%s'", capability, s.instance.name)
	}
	return nil
}

// prepareSecurityContext creates a v1.SecurityContext from the security configs
func (s *security) prepareSecurityContext() *v1.SecurityContext {
	sc := &v1.SecurityContext{}

	if s.privileged {
		sc.Privileged = &s.privileged
	}

	if len(s.capabilitiesAdd) == 0 {
		return sc
	}

	capabilities := make([]v1.Capability, len(s.capabilitiesAdd))
	for i, cap := range s.capabilitiesAdd {
		capabilities[i] = v1.Capability(cap)
	}
	sc.Capabilities = &v1.Capabilities{
		Add: capabilities,
	}

	return sc
}

func (s *security) clone() *security {
	capabilitiesAddCopy := make([]string, len(s.capabilitiesAdd))
	copy(capabilitiesAddCopy, s.capabilitiesAdd)

	return &security{
		instance:        nil,
		privileged:      s.privileged,
		capabilitiesAdd: capabilitiesAddCopy,
	}
}
