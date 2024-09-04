package basic

import (
	"context"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

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

	livenessProbe := v1.Probe{
		ProbeHandler: v1.ProbeHandler{
			HTTPGet: &v1.HTTPGetAction{
				Path: "/",
				Port: intstr.IntOrString{Type: intstr.Int, IntVal: e2e.NginxPort},
			},
		},
		InitialDelaySeconds: 10,
	}
	s.Require().NoError(web.Monitoring().SetLivenessProbe(&livenessProbe))
	s.Require().NoError(web.Build().Commit(ctx))

	// Test logic
	webIP, err := web.Network().GetIP(ctx)
	s.Require().NoError(err)

	s.Require().NoError(web.Execution().Start(ctx))

	wgetOutput, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	s.Require().NoError(err)

	wgetOutput = strings.TrimSpace(wgetOutput)
	s.Assert().Contains(wgetOutput, "Hello World!")
}
