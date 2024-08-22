package basic

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/sidecars/netshaper"
)

// TestReverseProxy is a test function that verifies the functionality of a reverse proxy setup.
// It mainly tests the ability to reach to a service running in a sidecar like netshaper (BitTwister).
// It calls an endpoint of the service and checks if the response is as expected.
func (s *Suite) TestReverseProxy() {
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

	ctx1min, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	s.Require().NoError(btSidecar.WaitForStart(ctx1min))

	// test if BitTwister running in a sidecar is accessible
	s.Require().NoError(btSidecar.SetBandwidthLimit(1000))

	// Check if the BitTwister service is set
	out, err := btSidecar.AllServicesStatus()
	s.Require().NoError(err)

	s.Assert().GreaterOrEqual(len(out), 1)
	s.Assert().NotEmpty(out[0].Name)
}

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
