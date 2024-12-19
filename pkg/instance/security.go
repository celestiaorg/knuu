package instance

import (
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

// represents the security settings for a container
type security struct {
	instance *Instance

	// Privileged indicates whether the container should be run in privileged mode
	privileged bool

	// CapabilitiesAdd is the list of capabilities to add to the container
	capabilitiesAdd []string

	// PolicyRules is the list of policy rules to add to the instance
	policyRules []rbacv1.PolicyRule
}

func (i *Instance) Security() *security {
	return i.security
}

// AddPolicyRule adds a policy rule to the instance
// This function can only be called in the states 'Preparing', 'Committed' and 'Stopped'
func (s *security) AddPolicyRule(rule rbacv1.PolicyRule) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrAddingPolicyRuleNotAllowed.WithParams(s.instance.state.String())
	}
	s.policyRules = append(s.policyRules, rule)
	return nil
}

// SetPrivileged sets the privileged status for the instance
// This function can only be called in the state 'Preparing', 'Committed' or 'Stopped'
func (s *security) SetPrivileged(privileged bool) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrSettingPrivilegedNotAllowed.WithParams(s.instance.state.String())
	}
	s.privileged = privileged
	s.instance.Logger.WithFields(logrus.Fields{
		"instance":   s.instance.name,
		"privileged": privileged,
	}).Debug("set privileged for instance")
	return nil
}

// AddKubernetesCapability adds a Kubernetes capability to the instance
// This function can only be called in the state 'Preparing', 'Committed' or 'Stopped'
func (s *security) AddKubernetesCapability(capability string) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrAddingCapabilityNotAllowed.WithParams(s.instance.state.String())
	}
	s.capabilitiesAdd = append(s.capabilitiesAdd, capability)
	s.instance.Logger.WithFields(logrus.Fields{
		"instance":   s.instance.name,
		"capability": capability,
	}).Debug("added capability to instance")
	return nil
}

// AddKubernetesCapabilities adds multiple Kubernetes capabilities to the instance
// This function can only be called in the state 'Preparing', 'Committed' or 'Stopped'
func (s *security) AddKubernetesCapabilities(capabilities []string) error {
	if !s.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrAddingCapabilitiesNotAllowed.WithParams(s.instance.state.String())
	}
	s.capabilitiesAdd = append(s.capabilitiesAdd, capabilities...)

	s.instance.Logger.WithFields(logrus.Fields{
		"instance":     s.instance.name,
		"capabilities": strings.Join(capabilities, ", "),
	}).Debug("added capabilities to instance")
	return nil
}

// prepareSecurityContext creates a v1.SecurityContext from the security configs
func (s *security) prepareSecurityContext() *corev1.SecurityContextApplyConfiguration {
	sc := &corev1.SecurityContextApplyConfiguration{}

	if s.privileged {
		sc.Privileged = &s.privileged
	}

	capabilities := make([]v1.Capability, len(s.capabilitiesAdd))
	for i, cap := range s.capabilitiesAdd {
		capabilities[i] = v1.Capability(cap)
	}
	sc.Capabilities = &corev1.CapabilitiesApplyConfiguration{
		Add: capabilities,
	}

	return sc
}

func (s *security) clone() *security {
	capabilitiesAddCopy := make([]string, len(s.capabilitiesAdd))
	copy(capabilitiesAddCopy, s.capabilitiesAdd)

	policyRulesCopy := make([]rbacv1.PolicyRule, len(s.policyRules))
	copy(policyRulesCopy, s.policyRules)

	return &security{
		instance:        nil,
		privileged:      s.privileged,
		capabilitiesAdd: capabilitiesAddCopy,
		policyRules:     policyRulesCopy,
	}
}
