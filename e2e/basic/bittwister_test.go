package basic

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxRandomTestCases = 2
	iperfImage         = "docker.io/clearlinux/iperf:latest"
	gopingImage        = "ghcr.io/celestiaorg/goping:4803195"
)

func TestBittwisterBandwidth(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		iperfTestDuration    = 45 * time.Second
		iperfParallelClients = 4
		commandTimeout       = 60 * time.Minute
	)

	iperfMother, err := knuu.NewInstance("iperf")
	require.NoError(t, err, "error creating instance")

	err = iperfMother.SetImage(iperfImage)
	require.NoError(t, err, "error setting image")

	err = iperfMother.SetCommand("iperf3", "-s")
	require.NoError(t, err, "error executing command")

	require.NoError(t, iperfMother.AddPortTCP(5201), "rror adding port")
	require.NoError(t, iperfMother.Commit(), "error committing instance")

	iperfServer, err := iperfMother.CloneWithName("iperf-server")
	require.NoError(t, err, "error cloning instance")

	iperfClient, err := iperfMother.CloneWithName("iperf-client")
	require.NoError(t, err, "error cloning instance")

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(iperfServer, iperfClient))
	})

	// Prepare iperf client & server

	iperfServerIP, err := iperfServer.GetIP()
	require.NoError(t, err, "error getting IP")

	require.NoError(t, iperfServer.EnableBitTwister(), "error enabling BitTwister")
	require.NoError(t, iperfServer.Start(), "error starting iperf-server instance")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, iperfServer.BitTwister.WaitForStart(ctx), "error waiting for BitTwister to start")

	require.NoError(t, iperfClient.Start(), "error starting iperf-client instance")

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
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Max bandwidth: %v \t tolerance: %v%%", formatBandwidth(float64(tc.targetBandwidth)), tc.tolerancePercent)

			err = iperfServer.SetBandwidthLimit(tc.targetBandwidth)
			require.NoError(t, err, "error setting bandwidth limit")

			t.Log("Starting bandwidth test. It takes a while.")
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
			defer cancel()
			output, err := iperfClient.ExecuteCommandWithContext(ctx,
				"iperf3", "-c", iperfServerIP,
				"-t", fmt.Sprint(int64(iperfTestDuration.Seconds())),
				"-P", fmt.Sprint(iperfParallelClients), "--json")
			require.NoError(t, err, "error executing command")

			elapsed := time.Since(startTime)
			t.Logf("test took %d seconds", int64(elapsed.Seconds()))

			var iperfOutput struct {
				End struct {
					SumReceived struct {
						BitsPerSecond float64 `json:"bits_per_second"`
					} `json:"sum_received"`
				} `json:"end"`
			}
			err = json.Unmarshal([]byte(output), &iperfOutput)
			require.NoError(t, err, "error unmarshalling JSON")

			deviationPercent := math.Abs(iperfOutput.End.SumReceived.BitsPerSecond-float64(tc.targetBandwidth)) / float64(tc.targetBandwidth) * 100
			assert.LessOrEqual(t, deviationPercent, float64(tc.tolerancePercent), "deviation is too high")

			t.Logf("bandwidth expected: %v \tgot: %v \tdeviation: %v%% \ttolerance: %v%%",
				formatBandwidth(float64(tc.targetBandwidth)),
				formatBandwidth(iperfOutput.End.SumReceived.BitsPerSecond),
				math.Round(deviationPercent),
				tc.tolerancePercent)
		})
	}
}

func TestBittwisterPacketloss(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		numOfPingPackets = 1000
		packetTimeout    = 1 * time.Second
		commandTimeout   = 60 * time.Minute
	)

	mother, err := knuu.NewInstance("mother")
	require.NoError(t, err, "error creating instance")

	err = mother.SetImage(gopingImage)
	require.NoError(t, err, "error setting image")

	gopingPort := 8001

	require.NoError(t, mother.AddPortTCP(gopingPort), "error adding port")
	require.NoError(t, mother.Commit(), "error committing instance")

	err = mother.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	require.NoError(t, err, "error setting environment variable")

	target, err := mother.CloneWithName("target")
	require.NoError(t, err, "error cloning instance")

	executor, err := mother.CloneWithName("executor")
	require.NoError(t, err, "error cloning instance")

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(executor, target))
	})

	// Prepare ping executor & target

	require.NoError(t, target.EnableBitTwister(), "error enabling BitTwister")
	require.NoError(t, target.Start(), "error starting target instance")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, target.BitTwister.WaitForStart(ctx), "error waiting for BitTwister to start")

	require.NoError(t, executor.Start(), "error starting executor instance")

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

	targetIP, err := target.GetIP()
	require.NoError(t, err, "error getting IP")

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Target packetloss: %v%% \t tolerance: %v%%", tc.targetPacketlossRate, tc.tolerancePercent)

			err = target.SetPacketLoss(tc.targetPacketlossRate)
			require.NoError(t, err, "error setting packetloss rate")

			t.Log("Starting packetloss test. It takes a while.")
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
			defer cancel()

			targetAddress := fmt.Sprintf("%s:%d", targetIP, gopingPort)
			output, err := executor.ExecuteCommandWithContext(ctx, "goping", "ping", "-q",
				"-c", fmt.Sprint(numOfPingPackets),
				"-t", packetTimeout.String(),
				"-m", "packetloss",
				targetAddress)
			require.NoError(t, err, "error executing command")

			elapsed := time.Since(startTime)
			t.Logf("test took %d seconds", int64(elapsed.Seconds()))

			gotPacketloss, err := strconv.ParseFloat(output, 64)
			require.NoError(t, err, fmt.Sprintf("error parsing output: `%s`", output))

			deviationPercent := math.Abs(gotPacketloss - float64(tc.targetPacketlossRate))
			assert.LessOrEqual(t, deviationPercent, float64(tc.tolerancePercent), "deviation is too high")

			t.Logf("Packetloss expected: %v%% \tgot: %.2f%% \tdeviation: %.2f%% \ttolerance: %v%%",
				tc.targetPacketlossRate,
				gotPacketloss,
				deviationPercent,
				tc.tolerancePercent)
		})
	}
}

func TestBittwisterLatency(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		numOfPingPackets = 100
		packetTimeout    = 1 * time.Second
		commandTimeout   = 60 * time.Minute
	)

	mother, err := knuu.NewInstance("mother")
	require.NoError(t, err, "error creating instance")

	err = mother.SetImage(gopingImage)
	require.NoError(t, err, "error setting image")

	gopingPort := 8001

	require.NoError(t, mother.AddPortTCP(gopingPort), "error adding port")
	require.NoError(t, mother.Commit(), "error committing instance")

	err = mother.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	require.NoError(t, err, "error setting environment variable")

	target, err := mother.CloneWithName("target")
	require.NoError(t, err, "error cloning instance")

	executor, err := mother.CloneWithName("executor")
	require.NoError(t, err, "error cloning instance")

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(executor, target))
	})

	// Prepare ping executor & target

	require.NoError(t, target.EnableBitTwister(), "error enabling BitTwister")
	require.NoError(t, target.Start(), "error starting target instance")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, target.BitTwister.WaitForStart(ctx), "error waiting for BitTwister to start")

	require.NoError(t, executor.Start(), "error starting executor instance")

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

	targetIP, err := target.GetIP()
	require.NoError(t, err, "error getting IP")

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Max latency: %v ms \t tolerance: %v%%", tc.targetLatency.Milliseconds(), tc.tolerancePercent)

			err = target.SetLatencyAndJitter(tc.targetLatency.Milliseconds(), 0)
			require.NoError(t, err, "error setting packetloss rate")

			t.Log("Starting latency test. It takes a while.")
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
			defer cancel()

			targetAddress := fmt.Sprintf("%s:%d", targetIP, gopingPort)
			output, err := executor.ExecuteCommandWithContext(ctx,
				"goping", "ping", "-q",
				"-c", fmt.Sprint(numOfPingPackets),
				// we need to make sure the client waits long enough for the server to respond
				"-t", (packetTimeout + tc.targetLatency).String(),
				"-m", "latency",
				targetAddress)
			require.NoError(t, err, "error executing command")

			elapsed := time.Since(startTime)
			t.Logf("Test took %d seconds", int64(elapsed.Seconds()))

			gotLatency, err := time.ParseDuration(output)
			require.NoError(t, err, "error parsing output")

			deviationPercent := math.Abs(float64(gotLatency-tc.targetLatency)/float64(tc.targetLatency)) * 100
			assert.LessOrEqual(t, deviationPercent, float64(tc.tolerancePercent), "Deviation is too high")

			t.Logf("Latency expected: %v \tgot: %v \tdeviation: %.2f%% \ttolerance: %v%%",
				tc.targetLatency.String(),
				gotLatency.String(),
				deviationPercent,
				tc.tolerancePercent)
		})
	}
}
func TestBittwisterJitter(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		numOfPingPackets = 100
		packetTimeout    = 1 * time.Second
		commandTimeout   = 60 * time.Minute
	)

	mother, err := knuu.NewInstance("mother")
	require.NoError(t, err, "error creating instance")

	err = mother.SetImage(gopingImage)
	require.NoError(t, err, "error setting image")

	gopingPort := 8001

	require.NoError(t, mother.AddPortTCP(gopingPort), "error adding port")
	require.NoError(t, mother.Commit(), "error committing instance")

	err = mother.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	require.NoError(t, err, "error setting environment variable")

	target, err := mother.CloneWithName("target")
	require.NoError(t, err, "error cloning instance")

	executor, err := mother.CloneWithName("executor")
	require.NoError(t, err, "error cloning instance")

	t.Cleanup(func() {
		require.NoError(t, knuu.BatchDestroy(executor, target))
	})

	// Prepare ping executor & target

	require.NoError(t, target.EnableBitTwister(), "error enabling BitTwister")
	require.NoError(t, target.Start(), "error starting target instance")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, target.BitTwister.WaitForStart(ctx), "error waiting for BitTwister to start")

	require.NoError(t, executor.Start(), "error starting executor instance")

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

	targetIP, err := target.GetIP()
	require.NoError(t, err, "error getting IP")

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Max jitter: %v", tc.maxTargetJitter.Milliseconds())

			err = target.SetLatencyAndJitter(0, tc.maxTargetJitter.Milliseconds())
			require.NoError(t, err, "error setting packetloss rate")

			t.Log("Starting jitter test. It takes a while.")
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
			defer cancel()

			targetAddress := fmt.Sprintf("%s:%d", targetIP, gopingPort)
			output, err := executor.ExecuteCommandWithContext(ctx,
				"goping", "ping", "-q",
				"-c", fmt.Sprint(numOfPingPackets),
				// we need to make sure the client waits long enough for the server to respond
				"-t", (packetTimeout + tc.maxTargetJitter).String(),
				"-m", "jitter",
				targetAddress)
			require.NoError(t, err, "error executing command")

			elapsed := time.Since(startTime)
			t.Logf("Test took %d seconds", int64(elapsed.Seconds()))

			gotAvgJitter, err := time.ParseDuration(output)
			require.NoError(t, err, "error parsing output")

			assert.LessOrEqual(t, gotAvgJitter, tc.maxTargetJitter, "Jitter is too high")
			t.Logf("Max Jitter expected: %v \tgot (average): %v", tc.maxTargetJitter.String(), gotAvgJitter.String())
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
