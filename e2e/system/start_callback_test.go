package system

import (
	"context"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (s *Suite) TestStartWithCallback() {
	const (
		namePrefix           = "callback"
		sleepTimeBeforeReady = "1" // second
	)

	// Setup
	ctx := context.Background()

	target := s.createNginxInstance(ctx, namePrefix+"-target")

	// This probe is used to make sure the instance will not be ready for a second so the
	// second execute command must fail and the first one with callback must succeed
	err := target.Monitoring().SetReadinessProbe(&corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/",
				Port: intstr.FromInt(nginxPort),
			},
		},
	})
	s.Require().NoError(err)

	err = target.Build().SetStartCommand([]string{"sleep", sleepTimeBeforeReady, "&&", nginxCommand}...)
	s.Require().NoError(err)

	s.Require().NoError(target.Build().Commit(ctx))

	wg := sync.WaitGroup{}
	err = target.Execution().StartWithCallback(ctx, func() {
		wg.Add(1)
		defer wg.Done()
		// This should Not fail as the instance will be ready when this is called
		out, err := target.Execution().ExecuteCommand(ctx, "curl", "-s", "http://localhost")
		s.Require().NoError(err)
		s.Require().Contains(out, "Welcome to nginx")
	})
	s.Require().NoError(err)

	// This should fail as the instance is not ready yet
	out, err := target.Execution().ExecuteCommand(ctx, "curl", "-s", "http://localhost")
	s.Require().Error(err)
	s.Require().Empty(out)

	// We need to have this to allow the async callback to finish
	wg.Wait()
}
