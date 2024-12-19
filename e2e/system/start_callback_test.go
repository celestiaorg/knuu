package system

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/util/intstr"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/celestiaorg/knuu/e2e"
)

func (s *Suite) TestStartWithCallback() {
	const (
		namePrefix           = "callback"
		sleepTimeBeforeReady = "1" // second
	)

	// Setup
	ctx := context.Background()

	target := s.CreateNginxInstance(ctx, namePrefix+"-target")

	// This probe is used to make sure the instance will not be ready for a second so the
	// second execute command must fail and the first one with callback must succeed
	err := target.Monitoring().SetReadinessProbe(
		corev1.Probe().
			WithHTTPGet(corev1.HTTPGetAction().
				WithPath("/").
				WithPort(intstr.FromInt(e2e.NginxPort)),
			),
	)
	s.Require().NoError(err)

	err = target.Build().SetStartCommand([]string{"sleep", sleepTimeBeforeReady, "&&", e2e.NginxCommand}...)
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
