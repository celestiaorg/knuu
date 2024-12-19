package instance

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/celestiaorg/knuu/pkg/k8s"
)

type monitoring struct {
	instance       *Instance
	livenessProbe  *corev1.ProbeApplyConfiguration
	readinessProbe *corev1.ProbeApplyConfiguration
	startupProbe   *corev1.ProbeApplyConfiguration
}

func (i *Instance) Monitoring() *monitoring {
	return i.monitoring
}

func (m *monitoring) Logs(ctx context.Context) (io.ReadCloser, error) {
	if m.instance.sidecars.IsSidecar() {
		return m.instance.K8sClient.GetLogStream(ctx, m.instance.parentInstance.Name(), m.instance.Name())
	}
	return m.instance.K8sClient.GetLogStream(ctx, m.instance.Name(), m.instance.Name())
}

// SetLivenessProbe sets the liveness probe of the instance
// A live probe is a probe that is used to determine if the instance is still alive, and should be restarted if not
// See usage documentation: https://pkg.go.dev/i.K8sCli.io/api/core/v1@v0.27.3#Probe
// This function can only be called in the states 'Preparing' and 'Committed'
func (m *monitoring) SetLivenessProbe(livenessProbe *corev1.ProbeApplyConfiguration) error {
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
func (m *monitoring) SetReadinessProbe(readinessProbe *corev1.ProbeApplyConfiguration) error {
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
func (m *monitoring) SetStartupProbe(startupProbe *corev1.ProbeApplyConfiguration) error {
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
	if !m.instance.IsInState(StatePreparing, StateCommitted, StateStopped) {
		return ErrSettingProbeNotAllowed.WithParams(m.instance.state.String())
	}
	return nil
}

func (m *monitoring) clone() *monitoring {
	if m == nil {
		return nil
	}

	var livenessProbeCopy *corev1.ProbeApplyConfiguration
	if m.livenessProbe != nil {
		livenessProbeCopy = k8s.DeepCopyProbe(m.livenessProbe)
	}

	var readinessProbeCopy *corev1.ProbeApplyConfiguration
	if m.readinessProbe != nil {
		readinessProbeCopy = k8s.DeepCopyProbe(m.readinessProbe)
	}

	var startupProbeCopy *corev1.ProbeApplyConfiguration
	if m.startupProbe != nil {
		startupProbeCopy = k8s.DeepCopyProbe(m.startupProbe)
	}

	return &monitoring{
		instance:       nil,
		livenessProbe:  livenessProbeCopy,
		readinessProbe: readinessProbeCopy,
		startupProbe:   startupProbeCopy,
	}
}
