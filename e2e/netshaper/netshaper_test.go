package netshaper

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/sidecars/netshaper"
)

const (
	maxRandomTestCases = 2
	iperfImage         = "networkstatic/iperf3:latest"
	gopingImage        = "ghcr.io/celestiaorg/goping:4803195"
)

func (s *Suite) TestNetShaperBandwidth() {
	const (
		namePrefix           = "ntshp-bw"
		iperfTestDuration    = 45 * time.Second
		iperfParallelClients = 4
		iperfPort            = 5201
	)
	ctx := context.Background()

	iperfMother, err := s.Knuu.NewInstance(namePrefix + "iperf")
	s.Require().NoError(err)

	s.Require().NoError(iperfMother.Build().SetImage(ctx, iperfImage))
	s.Require().NoError(iperfMother.Build().SetStartCommand("iperf3", "-s"))
	s.Require().NoError(iperfMother.Network().AddPortTCP(iperfPort))
	s.Require().NoError(iperfMother.Build().Commit(ctx))

	iperfServer, err := iperfMother.CloneWithName(namePrefix + "iperf-server")
	s.Require().NoError(err)

	iperfClient, err := iperfMother.CloneWithName(namePrefix + "iperf-client")
	s.Require().NoError(err)

	btSidecar := netshaper.New()
	s.Require().NoError(iperfServer.Sidecars().Add(ctx, btSidecar))

	s.T().Cleanup(func() {
		s.T().Log("Tearing down TestNetShaperBandwidth test...")
		err := instance.BatchDestroy(ctx, iperfServer, iperfClient)
		if err != nil {
			s.T().Logf("error destroying instances: %v", err)
		}
	})

	// Prepare iperf client & server

	s.Require().NoError(iperfServer.Execution().Start(ctx))
	s.Require().NoError(btSidecar.WaitForStart(ctx))

	s.Require().NoError(iperfClient.Execution().Start(ctx))

	iperfServerIP, err := iperfServer.Network().GetIP(ctx)
	s.Require().NoError(err)

	// Perform the test
	type testCase struct {
		name             string
		targetBandwidth  int64
		tolerancePercent int
	}
	tt := make([]testCase, maxRandomTestCases)
	for i := 0; i < maxRandomTestCases; i++ {
		tt[i] = testCase{
			name: fmt.Sprintf("random test case %d", i+1),

			// 512 Kbps to 4 Mbps
			targetBandwidth: int64(rand.Intn(4_000_000-512_000) + 512_000),

			// 40 to 50 percent tolerance
			// high tolerance is chosen as sometimes specially when we run a large test (like all e2e at the same time),
			// the bandwidth might not be stable and the test might fail due to some transient issues.
			tolerancePercent: rand.Intn(10) + 40,
		}
	}

	for _, tc := range tt {
		tc := tc
		s.Run(tc.name, func() {
			s.T().Logf("Max bandwidth: %v \t tolerance: %v%%", formatBandwidth(float64(tc.targetBandwidth)), tc.tolerancePercent)
			s.Require().NoError(btSidecar.SetBandwidthLimit(tc.targetBandwidth))

			s.T().Log("Starting bandwidth test. It takes a while.")
			startTime := time.Now()
			output, err := iperfClient.Execution().ExecuteCommand(ctx,
				"iperf3", "-c", iperfServerIP,
				"-t", fmt.Sprint(int64(iperfTestDuration.Seconds())),
				"-P", fmt.Sprint(iperfParallelClients), "--json")
			s.Require().NoError(err)

			elapsed := time.Since(startTime)
			s.T().Logf("test took %d seconds", int64(elapsed.Seconds()))

			var iperfOutput struct {
				End struct {
					SumReceived struct {
						BitsPerSecond float64 `json:"bits_per_second"`
					} `json:"sum_received"`
				} `json:"end"`
			}
			s.Require().NoError(json.Unmarshal([]byte(output), &iperfOutput))

			deviationPercent := math.Abs(iperfOutput.End.SumReceived.BitsPerSecond-float64(tc.targetBandwidth)) / float64(tc.targetBandwidth) * 100
			s.Assert().LessOrEqual(deviationPercent, float64(tc.tolerancePercent), "deviation is too high")

			s.T().Logf("bandwidth expected: %v \tgot: %v \tdeviation: %v%% \ttolerance: %v%%",
				formatBandwidth(float64(tc.targetBandwidth)),
				formatBandwidth(iperfOutput.End.SumReceived.BitsPerSecond),
				math.Round(deviationPercent),
				tc.tolerancePercent)
		})
	}
}

func (s *Suite) TestNetShaperPacketloss() {
	const (
		namePrefix       = "ntshp-pl"
		numOfPingPackets = 100
		packetTimeout    = 1 * time.Second
		gopingPort       = 8001
	)
	ctx := context.Background()

	mother, err := s.Knuu.NewInstance(namePrefix + "mother")
	s.Require().NoError(err)

	err = mother.Build().SetImage(ctx, gopingImage)
	s.Require().NoError(err)

	s.Require().NoError(mother.Network().AddPortTCP(gopingPort))
	s.Require().NoError(mother.Build().Commit(ctx))

	err = mother.Build().SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	s.Require().NoError(err)

	target, err := mother.CloneWithName(namePrefix + "target")
	s.Require().NoError(err)

	btSidecar := netshaper.New()
	s.Require().NoError(target.Sidecars().Add(ctx, btSidecar))

	executor, err := mother.CloneWithName(namePrefix + "executor")
	s.Require().NoError(err)

	// Prepare ping executor & target

	s.Require().NoError(target.Execution().Start(ctx))
	s.Require().NoError(btSidecar.WaitForStart(ctx))

	s.Require().NoError(executor.Execution().Start(ctx))

	// Perform the test
	type testCase struct {
		name                 string
		targetPacketlossRate int32
		tolerancePercent     int
	}
	tt := make([]testCase, maxRandomTestCases)
	for i := 0; i < maxRandomTestCases; i++ {
		randPacketloss := rand.Intn(80) + 10 // 10 to 90 %
		tt[i] = testCase{
			name:                 fmt.Sprintf("random test case %d", i+1),
			targetPacketlossRate: int32(randPacketloss),

			// higher tolerance for lower packetloss rates
			tolerancePercent: 50 - randPacketloss/2,
		}
	}

	targetIP, err := target.Network().GetIP(ctx)
	s.Require().NoError(err)

	for _, tc := range tt {
		tc := tc
		s.Run(tc.name, func() {
			s.T().Logf("Target packetloss: %v%% \t tolerance: %v%%", tc.targetPacketlossRate, tc.tolerancePercent)
			s.Require().NoError(btSidecar.SetPacketLoss(tc.targetPacketlossRate))

			s.T().Log("Starting packetloss test. It takes a while.")
			startTime := time.Now()

			targetAddress := fmt.Sprintf("%s:%d", targetIP, gopingPort)
			output, err := executor.Execution().ExecuteCommand(ctx, "goping", "ping", "-q",
				"-c", fmt.Sprint(numOfPingPackets),
				"-t", packetTimeout.String(),
				"-m", "packetloss",
				targetAddress)
			s.Require().NoError(err)

			elapsed := time.Since(startTime)
			s.T().Logf("test took %d seconds", int64(elapsed.Seconds()))

			gotPacketloss, err := strconv.ParseFloat(output, 64)
			s.Require().NoError(err, fmt.Sprintf("error parsing output: `%s`", output))

			deviationPercent := math.Abs(gotPacketloss - float64(tc.targetPacketlossRate))
			s.Assert().LessOrEqual(deviationPercent, float64(tc.tolerancePercent), "deviation is too high")

			s.T().Logf("Packetloss expected: %v%% \tgot: %.2f%% \tdeviation: %.2f%% \ttolerance: %v%%",
				tc.targetPacketlossRate,
				gotPacketloss,
				deviationPercent,
				tc.tolerancePercent)
		})
	}
}

func (s *Suite) TestNetShaperLatency() {
	const (
		namePrefix       = "ntshp-lat"
		numOfPingPackets = 100
		gopingPort       = 8001
		packetTimeout    = 1 * time.Second
	)
	ctx := context.Background()

	mother, err := s.Knuu.NewInstance(namePrefix + "mother")
	s.Require().NoError(err)

	err = mother.Build().SetImage(ctx, gopingImage)
	s.Require().NoError(err)

	s.Require().NoError(mother.Network().AddPortTCP(gopingPort))
	s.Require().NoError(mother.Build().Commit(ctx))

	err = mother.Build().SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	s.Require().NoError(err)

	target, err := mother.CloneWithName(namePrefix + "target")
	s.Require().NoError(err)

	btSidecar := netshaper.New()
	s.Require().NoError(target.Sidecars().Add(ctx, btSidecar))

	executor, err := mother.CloneWithName(namePrefix + "executor")
	s.Require().NoError(err)

	// Prepare ping executor & target

	s.Require().NoError(target.Execution().Start(ctx))
	s.Require().NoError(btSidecar.WaitForStart(ctx))

	s.Require().NoError(executor.Execution().Start(ctx))

	// Perform the test

	type testCase struct {
		name             string
		targetLatency    time.Duration
		tolerancePercent int
	}
	tt := make([]testCase, maxRandomTestCases)
	for i := 0; i < maxRandomTestCases; i++ {
		tt[i] = testCase{
			name: fmt.Sprintf("random test case %d", i+1),

			// 50 to 1000ms
			targetLatency: time.Duration(rand.Intn(1000)+50) * time.Millisecond,

			// 30 to 50 percent tolerance
			// high tolerance is chosen as sometimes specially when we run a large test (like all e2e at the same time),
			// the latency might not be stable and the test might fail due to some transient issues.
			tolerancePercent: rand.Intn(20) + 30,
		}
	}

	targetIP, err := target.Network().GetIP(ctx)
	s.Require().NoError(err)

	for _, tc := range tt {
		tc := tc
		s.Run(tc.name, func() {
			s.T().Logf("Max latency: %v ms \t tolerance: %v%%", tc.targetLatency.Milliseconds(), tc.tolerancePercent)

			err = btSidecar.SetLatencyAndJitter(tc.targetLatency.Milliseconds(), 0)
			s.Require().NoError(err)

			s.T().Log("Starting latency test. It takes a while.")
			startTime := time.Now()

			targetAddress := fmt.Sprintf("%s:%d", targetIP, gopingPort)
			output, err := executor.Execution().ExecuteCommand(ctx,
				"goping", "ping", "-q",
				"-c", fmt.Sprint(numOfPingPackets),
				// we need to make sure the client waits long enough for the server to respond
				"-t", (packetTimeout + tc.targetLatency).String(),
				"-m", "latency",
				targetAddress)
			s.Require().NoError(err)

			elapsed := time.Since(startTime)
			s.T().Logf("Test took %d seconds", int64(elapsed.Seconds()))

			gotLatency, err := time.ParseDuration(output)
			s.Require().NoError(err)

			deviationPercent := math.Abs(float64(gotLatency-tc.targetLatency)/float64(tc.targetLatency)) * 100
			s.Assert().LessOrEqual(deviationPercent, float64(tc.tolerancePercent), "Deviation is too high")

			s.T().Logf("Latency expected: %v \tgot: %v \tdeviation: %.2f%% \ttolerance: %v%%",
				tc.targetLatency.String(),
				gotLatency.String(),
				deviationPercent,
				tc.tolerancePercent)
		})
	}
}
func (s *Suite) TestNetShaperJitter() {
	const (
		namePrefix       = "ntshp-jit"
		numOfPingPackets = 100
		gopingPort       = 8001
		packetTimeout    = 1 * time.Second
	)
	ctx := context.Background()

	mother, err := s.Knuu.NewInstance(namePrefix + "mother")
	s.Require().NoError(err)

	err = mother.Build().SetImage(ctx, gopingImage)
	s.Require().NoError(err)

	s.Require().NoError(mother.Network().AddPortTCP(gopingPort))
	s.Require().NoError(mother.Build().Commit(ctx))

	err = mother.Build().SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	s.Require().NoError(err)

	target, err := mother.CloneWithName(namePrefix + "target")
	s.Require().NoError(err)

	btSidecar := netshaper.New()
	s.Require().NoError(target.Sidecars().Add(ctx, btSidecar))

	executor, err := mother.CloneWithName(namePrefix + "executor")
	s.Require().NoError(err)

	// Prepare ping executor & target

	s.Require().NoError(target.Execution().Start(ctx))
	s.Require().NoError(btSidecar.WaitForStart(ctx))

	s.Require().NoError(executor.Execution().Start(ctx))

	// Perform the test

	type testCase struct {
		name            string
		maxTargetJitter time.Duration
	}
	tt := make([]testCase, maxRandomTestCases)
	for i := 0; i < maxRandomTestCases; i++ {
		tt[i] = testCase{
			name: fmt.Sprintf("random test case %d", i+1),

			// 10ms to 500ms
			maxTargetJitter: time.Duration(rand.Intn(490)+10) * time.Millisecond,
		}
	}

	targetIP, err := target.Network().GetIP(ctx)
	s.Require().NoError(err)

	for _, tc := range tt {
		tc := tc
		s.Run(tc.name, func() {
			s.T().Logf("Max jitter: %v", tc.maxTargetJitter.Milliseconds())

			err = btSidecar.SetLatencyAndJitter(0, tc.maxTargetJitter.Milliseconds())
			s.Require().NoError(err)

			s.T().Log("Starting jitter test. It takes a while.")
			startTime := time.Now()

			targetAddress := fmt.Sprintf("%s:%d", targetIP, gopingPort)
			output, err := executor.Execution().ExecuteCommand(ctx,
				"goping", "ping", "-q",
				"-c", fmt.Sprint(numOfPingPackets),
				// we need to make sure the client waits long enough for the server to respond
				"-t", (packetTimeout + tc.maxTargetJitter).String(),
				"-m", "jitter",
				targetAddress)
			s.Require().NoError(err)

			elapsed := time.Since(startTime)
			s.T().Logf("Test took %d seconds", int64(elapsed.Seconds()))

			gotAvgJitter, err := time.ParseDuration(output)
			s.Require().NoError(err)

			s.Assert().LessOrEqual(gotAvgJitter, tc.maxTargetJitter, "Jitter is too high")
			s.T().Logf("Max Jitter expected: %v \tgot (average): %v", tc.maxTargetJitter.String(), gotAvgJitter.String())
		})
	}
}

func formatBandwidth(bandwidth float64) string {
	units := []string{"bps", "Kbps", "Mbps", "Gbps"}
	if bandwidth < 0 {
		return ""
	}
	unitIndex := 0
	for bandwidth >= 1000 && unitIndex < len(units)-1 {
		bandwidth /= 1000
		unitIndex++
	}

	bandwidth = math.Round(bandwidth*100) / 100
	return fmt.Sprintf("%.2f %s", bandwidth, units[unitIndex])
}
