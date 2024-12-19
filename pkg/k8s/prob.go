package k8s

import corev1 "k8s.io/client-go/applyconfigurations/core/v1"

func DeepCopyProbe(probe *corev1.ProbeApplyConfiguration) *corev1.ProbeApplyConfiguration {
	if probe == nil {
		return nil
	}

	copy := &corev1.ProbeApplyConfiguration{
		InitialDelaySeconds:           probe.InitialDelaySeconds,
		TimeoutSeconds:                probe.TimeoutSeconds,
		PeriodSeconds:                 probe.PeriodSeconds,
		SuccessThreshold:              probe.SuccessThreshold,
		FailureThreshold:              probe.FailureThreshold,
		TerminationGracePeriodSeconds: probe.TerminationGracePeriodSeconds,
	}

	if probe.Exec != nil {
		copy.Exec = &corev1.ExecActionApplyConfiguration{
			Command: append([]string{}, probe.Exec.Command...),
		}
	}

	if probe.HTTPGet != nil {
		copy.HTTPGet = &corev1.HTTPGetActionApplyConfiguration{
			Path:   probe.HTTPGet.Path,
			Port:   probe.HTTPGet.Port,
			Host:   probe.HTTPGet.Host,
			Scheme: probe.HTTPGet.Scheme,
		}
	}

	if probe.TCPSocket != nil {
		copy.TCPSocket = &corev1.TCPSocketActionApplyConfiguration{
			Port: probe.TCPSocket.Port,
		}
	}

	if probe.GRPC != nil {
		copy.GRPC = &corev1.GRPCActionApplyConfiguration{
			Port:    probe.GRPC.Port,
			Service: probe.GRPC.Service,
		}
	}

	return copy
}
