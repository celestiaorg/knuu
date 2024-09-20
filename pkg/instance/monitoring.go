package instance

import (
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type monitoring struct {
	instance       *Instance
	livenessProbe  *v1.Probe
	readinessProbe *v1.Probe
	startupProbe   *v1.Probe
}

func (i *Instance) Monitoring() *monitoring {
	return i.monitoring
}

// SetLivenessProbe sets the liveness probe of the instance
// A live probe is a probe that is used to determine if the instance is still alive, and should be restarted if not
// See usage documentation: https://pkg.go.dev/i.K8sCli.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (m *monitoring) SetLivenessProbe(livenessProbe *v1.Probe) error {
	if err := m.checkStateForProbe(); err != nil {
		return err
	}
	m.livenessProbe = livenessProbe
	m.instance.Logger.WithFields(logrus.Fields{
		"instance":       m.instance.name,
		"liveness_probe": livenessProbe,
	}).Debug("set liveness probe")
	return nil
}

// SetReadinessProbe sets the readiness probe of the instance
// A readiness probe is a probe that is used to determine if the instance is ready to receive traffic
// See usage documentation: https://pkg.go.dev/i.K8sCli.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (m *monitoring) SetReadinessProbe(readinessProbe *v1.Probe) error {
	if err := m.checkStateForProbe(); err != nil {
		return err
	}
	m.readinessProbe = readinessProbe
	m.instance.Logger.WithFields(logrus.Fields{
		"instance":        m.instance.name,
		"readiness_probe": readinessProbe,
	}).Debug("set readiness probe")
	return nil
}

// SetStartupProbe sets the startup probe of the instance
// A startup probe is a probe that is used to determine if the instance is ready to receive traffic after a startup
// See usage documentation: https://pkg.go.dev/i.K8sCli.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (m *monitoring) SetStartupProbe(startupProbe *v1.Probe) error {
	if err := m.checkStateForProbe(); err != nil {
		return err
	}
	m.startupProbe = startupProbe
	m.instance.Logger.WithFields(logrus.Fields{
		"instance":      m.instance.name,
		"startup_probe": startupProbe,
	}).Debug("set startup probe")
	return nil
}

// checkStateForProbe checks if the current state is allowed for setting a probe
func (m *monitoring) checkStateForProbe() error {
	if !m.instance.IsInState(StatePreparing, StateCommitted) {
		return ErrSettingProbeNotAllowed.WithParams(m.instance.state.String())
	}
	return nil
}

func (m *monitoring) clone() *monitoring {
	if m == nil {
		return nil
	}

	var livenessProbeCopy *v1.Probe
	if m.livenessProbe != nil {
		livenessProbeCopy = m.livenessProbe.DeepCopy()
	}

	var readinessProbeCopy *v1.Probe
	if m.readinessProbe != nil {
		readinessProbeCopy = m.readinessProbe.DeepCopy()
	}

	var startupProbeCopy *v1.Probe
	if m.startupProbe != nil {
		startupProbeCopy = m.startupProbe.DeepCopy()
	}

	return &monitoring{
		instance:       nil,
		livenessProbe:  livenessProbeCopy,
		readinessProbe: readinessProbeCopy,
		startupProbe:   startupProbeCopy,
	}
}
