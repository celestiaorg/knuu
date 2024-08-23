package basic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/sidecars/netshaper"
)

// TestReverseProxy is a test function that verifies the functionality of a reverse proxy setup.
// It mainly tests the ability to reach to a service running in a sidecar like netshaper (BitTwister).
// It calls an endpoint of the service and checks if the response is as expected.
func (s *Suite) TestReverseProxyTMP() {
	const namePrefix = "reverse-proxy"
	ctx := context.Background()

	main, err := s.Knuu.NewInstance(namePrefix + "-main")
	s.Require().NoError(err)

	s.Require().NoError(main.Build().SetImage(ctx, alpineImage))
	s.Require().NoError(main.Build().SetStartCommand("sleep", "infinite"))
	s.Require().NoError(main.Build().Commit(ctx))

	btSidecar := netshaper.New()
	s.Require().NoError(main.Sidecars().Add(ctx, btSidecar))

	s.Require().NoError(main.Execution().Start(ctx))

	s.Require().NoError(btSidecar.WaitForStart(ctx))

	// // assert that the BitTwister running in a sidecar is accessible
	// s.Assert().NoError(btSidecar.SetBandwidthLimit(1000))

	// Check if the BitTwister service is set
	out, err := btSidecar.AllServicesStatus()
	s.Require().NoError(err)

	fmt.Printf("\n\t\tout: %+v\n", out)

	s.Assert().GreaterOrEqual(len(out), 1)
	s.Assert().NotEmpty(out[0].Name)

	// assert that the BitTwister running in a sidecar is accessible
	s.Assert().NoError(btSidecar.SetBandwidthLimit(4_000_000))
}

// func (s *Suite) TestReverseProxy() {
// 	const namePrefix = "reverse-proxy"
// 	ctx := context.Background()

// 	target := s.createNginxInstance(ctx, namePrefix+"-target")

// 	// Make sure it is ready when we want to test the proxy
// 	livenessProbe := v1.Probe{
// 		ProbeHandler: v1.ProbeHandler{
// 			HTTPGet: &v1.HTTPGetAction{
// 				Path: "/",
// 				Port: intstr.IntOrString{Type: intstr.Int, IntVal: nginxPort},
// 			},
// 		},
// 		InitialDelaySeconds: 10,
// 	}
// 	s.Require().NoError(target.Monitoring().SetLivenessProbe(&livenessProbe))

// 	s.Require().NoError(target.Build().Commit(ctx))
// 	s.Require().NoError(target.Execution().Start(ctx))

// 	host, err := target.Network().AddHost(ctx, nginxPort)
// 	s.Require().NoError(err)
// 	s.Assert().NotEmpty(host)

// 	// Just to be on the safe side so the proxy setting is ready to be used
// 	// The best way is ot use the AddHostWithReadyCheck, but that is out of the scope of this test
// 	time.Sleep(10 * time.Second)

// 	resp, err := http.Get(host)
// 	s.Require().NoError(err)

// 	defer resp.Body.Close()
// 	s.Assert().Equal(http.StatusOK, resp.StatusCode)

// 	bodyBytes, err := io.ReadAll(resp.Body)
// 	s.Require().NoError(err)
// 	s.Assert().Contains(string(bodyBytes), "Welcome to nginx!")

// 	// Perform a POST request
// 	postBody := `{"key": "value"}`
// 	resp, err = http.Post(host, "application/json", bytes.NewBuffer([]byte(postBody)))
// 	s.Require().NoError(err)

// 	defer resp.Body.Close()
// 	s.Assert().Equal(http.StatusOK, resp.StatusCode)

// 	bodyBytes, err = io.ReadAll(resp.Body)
// 	s.Require().NoError(err)
// 	s.Assert().Contains(string(bodyBytes), "expected response content")
// }

func (s *Suite) TestAddHostWithReadyCheck() {
	const namePrefix = "add-host-with-ready-check"
	ctx := context.Background()

	target := s.createNginxInstance(ctx, namePrefix+"-target")
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// checkFunc verifies the proxy is serving the nginx page
	checkFunc := func(host string) (bool, error) {
		resp, err := http.Get(host)
		fmt.Printf("\n\nresp: %v\n\terr: %v\n", resp, err)
		if resp != nil {
			fmt.Printf("\tresp.StatusCode: %v\n", resp.StatusCode)
		}
		if err != nil {
			return false, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, nil
		}
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		return strings.Contains(string(bodyBytes), "Welcome to nginx!"), nil
	}

	host, err := target.Network().AddHostWithReadyCheck(ctx, nginxPort, checkFunc)
	s.Require().NoError(err, "error adding host with ready check")
	s.Assert().NotEmpty(host, "host should not be empty")

	// Additional verification that the host is accessible
	ok, err := checkFunc(host)
	s.Require().NoError(err, "error checking host")
	s.Assert().True(ok, "Host should be ready and serving content: expected true, got false")
}
