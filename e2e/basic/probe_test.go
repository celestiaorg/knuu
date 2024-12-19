package basic

import (
	"context"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"

	"github.com/celestiaorg/knuu/e2e"
)

func (s *Suite) TestProbe() {
	const namePrefix = "probe"
	ctx := context.Background()

	// Ideally this has to be defined in the test suit setup
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	web := s.CreateNginxInstanceWithVolume(ctx, namePrefix+"-web")

	err = web.Storage().AddFile(resourcesHTML+"/index.html", e2e.NginxHTMLPath+"/index.html", "0:0")
	s.Require().NoError(err)

	err = web.Monitoring().SetLivenessProbe(
		corev1.Probe().
			WithHTTPGet(corev1.HTTPGetAction().
				WithPath("/").
				WithPort(intstr.FromInt(e2e.NginxPort)),
			).
			WithInitialDelaySeconds(10),
	)
	s.Require().NoError(err)

	s.Require().NoError(web.Build().Commit(ctx))
	s.Require().NoError(web.Execution().Start(ctx))

	webIP, err := web.Network().GetEphemeralIP(ctx)
	s.Require().NoError(err)

	wgetOutput, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	s.Require().NoError(err)

	wgetOutput = strings.TrimSpace(wgetOutput)
	s.Assert().Contains(wgetOutput, "Hello World!")
}
