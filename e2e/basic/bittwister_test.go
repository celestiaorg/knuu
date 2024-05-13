package basic

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/celestiaorg/knuu/pkg/knuu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	iperfImage  = "docker.io/clearlinux/iperf:latest"
	gopingImage = "ghcr.io/celestiaorg/goping:4803195"
)

func TestBittwister_Bandwidth(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		iperfTestDuration    = 45 * time.Second
		iperfParallelClients = 4
		commandTimeout       = 60 * time.Minute
	)

	iperfMother, err := knuu.NewInstance("iperf")
	require.NoError(t, err, "Error creating instance")

	err = iperfMother.SetImage(iperfImage)
	require.NoError(t, err, "Error setting image")

	err = iperfMother.SetCommand("iperf3", "-s")
	require.NoError(t, err, "Error executing command")

	require.NoError(t, iperfMother.AddPortTCP(5201), "Error adding port")
	require.NoError(t, iperfMother.Commit(), "Error committing instance")

	iperfServer, err := iperfMother.CloneWithName("iperf-server")
	require.NoError(t, err, "Error cloning instance")

	iperfClient, err := iperfMother.CloneWithName("iperf-client")
	require.NoError(t, err, "Error cloning instance")

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		require.NoError(t, iperfServer.Destroy(), "Error destroying iperf-server instance")
		require.NoError(t, iperfClient.Destroy(), "Error destroying iperf-client instance")
	})

	// Prepare iperf client & server

	iperfServerIP, err := iperfServer.GetIP()
	require.NoError(t, err, "Error getting IP")

	require.NoError(t, iperfServer.EnableBitTwister(), "Error enabling BitTwister")
	require.NoError(t, iperfServer.Start(), "Error starting iperf-server instance")

	forwardBitTwisterPort(t, iperfServer)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, iperfServer.BitTwister.WaitForStart(ctx), "Error waiting for BitTwister to start")

	require.NoError(t, iperfClient.Start(), "Error starting iperf-client instance")

	// Perform the test

	tt := []struct {
		name             string
		targetBandwidth  int64
		tolerancePercent int
	}{{
		name:             "512 Kbps",
		targetBandwidth:  512 * 1000,
		tolerancePercent: 50,
	},
		{
			name:             "1 Mbps",
			targetBandwidth:  1024 * 1000,
			tolerancePercent: 50,
		},
		{
			name:             "2 Mbps",
			targetBandwidth:  2 * 1024 * 1000,
			tolerancePercent: 50,
		},
		{
			name:             "4 Mbps",
			targetBandwidth:  4 * 1024 * 1000,
			tolerancePercent: 50,
		},
		{
			name:             "8 Mbps",
			targetBandwidth:  8 * 1024 * 1000,
			tolerancePercent: 55,
		},
		{
			name:             "16 Mbps",
			targetBandwidth:  16 * 1024 * 1000,
			tolerancePercent: 50,
		},
		{
			name:             "32 Mbps",
			targetBandwidth:  32 * 1024 * 1000,
			tolerancePercent: 50,
		},
	}

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err = iperfServer.SetBandwidthLimit(tc.targetBandwidth)
			require.NoError(t, err, "Error setting bandwidth limit")

			t.Log("Starting bandwidth test. It takes a while.")
			startTime := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
			defer cancel()
			output, err := iperfClient.ExecuteCommandWithContext(ctx,
				"iperf3", "-c", iperfServerIP,
				"-t", fmt.Sprint(int64(iperfTestDuration.Seconds())),
				"-P", fmt.Sprint(iperfParallelClients), "--json")
			require.NoError(t, err, "Error executing command")

			elapsed := time.Since(startTime)
			t.Logf("Test took %d seconds", int64(elapsed.Seconds()))

			var iperfOutput struct {
				End struct {
					SumReceived struct {
						BitsPerSecond float64 `json:"bits_per_second"`
					} `json:"sum_received"`
				} `json:"end"`
			}
			err = json.Unmarshal([]byte(output), &iperfOutput)
			require.NoError(t, err, "Error unmarshalling JSON")

			deviationPercent := math.Abs(iperfOutput.End.SumReceived.BitsPerSecond-float64(tc.targetBandwidth)) / float64(tc.targetBandwidth) * 100
			assert.LessOrEqual(t, deviationPercent, float64(tc.tolerancePercent), "Deviation is too high")

			t.Logf("Bandwidth expected: %v \tgot: %v \tdeviation: %v%% \ttolerance: %v%%",
				formatBandwidth(float64(tc.targetBandwidth)),
				formatBandwidth(iperfOutput.End.SumReceived.BitsPerSecond),
				math.Round(deviationPercent),
				tc.tolerancePercent)
		})
	}
}

func TestBittwister_Packetloss(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		numOfPingPackets = 1000
		packetTimeout    = 1 * time.Second
		commandTimeout   = 60 * time.Minute
	)

	mother, err := knuu.NewInstance("mother")
	require.NoError(t, err, "Error creating instance")

	err = mother.SetImage(gopingImage)
	require.NoError(t, err, "Error setting image")

	gopingPort := 8001

	require.NoError(t, mother.AddPortTCP(gopingPort), "Error adding port")
	require.NoError(t, mother.Commit(), "Error committing instance")

	err = mother.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	require.NoError(t, err, "Error setting environment variable")

	target, err := mother.CloneWithName("target")
	require.NoError(t, err, "Error cloning instance")

	executor, err := mother.CloneWithName("executor")
	require.NoError(t, err, "Error cloning instance")

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		require.NoError(t, executor.Destroy(), "Error destroying executor instance")
		require.NoError(t, target.Destroy(), "Error destroying target instance")
	})

	// Prepare ping executor & target

	require.NoError(t, target.EnableBitTwister(), "Error enabling BitTwister")
	require.NoError(t, target.Start(), "Error starting target instance")

	forwardBitTwisterPort(t, target)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, target.BitTwister.WaitForStart(ctx), "Error waiting for BitTwister to start")

	require.NoError(t, executor.Start(), "Error starting executor instance")

	// Perform the test
	tt := []struct {
		name                 string
		targetPacketlossRate int32
		tolerancePercent     int
	}{
		{
			name:                 "10%",
			targetPacketlossRate: 10,
			tolerancePercent:     50,
		},
		{
			name:                 "20%",
			targetPacketlossRate: 20,
			tolerancePercent:     30,
		},
		{
			name:                 "30%",
			targetPacketlossRate: 30,
			tolerancePercent:     10,
		},
		{
			name:                 "50%",
			targetPacketlossRate: 50,
			tolerancePercent:     10,
		},
		{
			name:                 "70%",
			targetPacketlossRate: 70,
			tolerancePercent:     10,
		},
		{
			name:                 "90%",
			targetPacketlossRate: 90,
			tolerancePercent:     10,
		},
		{
			name:                 "100%",
			targetPacketlossRate: 100,
			tolerancePercent:     10,
		},
	}

	targetIP, err := target.GetIP()
	require.NoError(t, err, "Error getting IP")

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err = target.SetPacketLoss(tc.targetPacketlossRate)
			require.NoError(t, err, "Error setting packetloss rate")

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
			require.NoError(t, err, "Error executing command")

			elapsed := time.Since(startTime)
			t.Logf("Test took %d seconds", int64(elapsed.Seconds()))

			gotPacketloss, err := strconv.ParseFloat(output, 64)
			require.NoError(t, err, fmt.Sprintf("Error parsing output: `%s`", output))

			deviationPercent := math.Abs(gotPacketloss - float64(tc.targetPacketlossRate))
			assert.LessOrEqual(t, deviationPercent, float64(tc.tolerancePercent), "Deviation is too high")

			t.Logf("Packetloss expected: %v%% \tgot: %.2f%% \tdeviation: %.2f%% \ttolerance: %v%%",
				tc.targetPacketlossRate,
				gotPacketloss,
				deviationPercent,
				tc.tolerancePercent)
		})
	}
}

func TestBittwister_Latency(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		numOfPingPackets = 100
		packetTimeout    = 1 * time.Second
		commandTimeout   = 60 * time.Minute
	)

	mother, err := knuu.NewInstance("mother")
	require.NoError(t, err, "Error creating instance")

	err = mother.SetImage(gopingImage)
	require.NoError(t, err, "Error setting image")

	gopingPort := 8001

	require.NoError(t, mother.AddPortTCP(gopingPort), "Error adding port")
	require.NoError(t, mother.Commit(), "Error committing instance")

	err = mother.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	require.NoError(t, err, "Error setting environment variable")

	target, err := mother.CloneWithName("target")
	require.NoError(t, err, "Error cloning instance")

	executor, err := mother.CloneWithName("executor")
	require.NoError(t, err, "Error cloning instance")

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		require.NoError(t, executor.Destroy(), "Error destroying executor instance")
		require.NoError(t, target.Destroy(), "Error destroying target instance")
	})

	// Prepare ping executor & target

	require.NoError(t, target.EnableBitTwister(), "Error enabling BitTwister")
	require.NoError(t, target.Start(), "Error starting target instance")

	forwardBitTwisterPort(t, target)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, target.BitTwister.WaitForStart(ctx), "Error waiting for BitTwister to start")

	require.NoError(t, executor.Start(), "Error starting executor instance")

	// Perform the test

	tt := []struct {
		name             string
		targetLatency    time.Duration
		tolerancePercent int
	}{
		{
			name:             "10Maxms",
			targetLatency:    10 * time.Millisecond,
			tolerancePercent: 50,
		},
		{
			name:             "20Maxms",
			targetLatency:    20 * time.Millisecond,
			tolerancePercent: 50,
		},
		{
			name:             "50Maxms",
			targetLatency:    50 * time.Millisecond,
			tolerancePercent: 50,
		},
		{
			name:             "100Maxms",
			targetLatency:    100 * time.Millisecond,
			tolerancePercent: 50,
		},
		{
			name:             "200Maxms",
			targetLatency:    200 * time.Millisecond,
			tolerancePercent: 50,
		},
		{
			name:             "500Maxms",
			targetLatency:    500 * time.Millisecond,
			tolerancePercent: 50,
		},
		{
			name:             "Max1s",
			targetLatency:    1 * time.Second,
			tolerancePercent: 50,
		},
		{
			name:             "Max2s",
			targetLatency:    2 * time.Second,
			tolerancePercent: 50,
		},
		{
			name:             "Max3s",
			targetLatency:    3 * time.Second,
			tolerancePercent: 50,
		},
		{
			name:             "Max5s",
			targetLatency:    5 * time.Second,
			tolerancePercent: 50,
		},
	}

	targetIP, err := target.GetIP()
	require.NoError(t, err, "Error getting IP")

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err = target.SetLatencyAndJitter(tc.targetLatency.Milliseconds(), 0)
			require.NoError(t, err, "Error setting packetloss rate")

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
			require.NoError(t, err, "Error executing command")

			elapsed := time.Since(startTime)
			t.Logf("Test took %d seconds", int64(elapsed.Seconds()))

			gotLatency, err := time.ParseDuration(output)
			require.NoError(t, err, "Error parsing output")

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
func TestBittwister_Jitter(t *testing.T) {
	t.Parallel()
	// Setup

	const (
		numOfPingPackets = 100
		packetTimeout    = 1 * time.Second
		commandTimeout   = 60 * time.Minute
	)

	mother, err := knuu.NewInstance("mother")
	require.NoError(t, err, "Error creating instance")

	err = mother.SetImage(gopingImage)
	require.NoError(t, err, "Error setting image")

	gopingPort := 8001

	require.NoError(t, mother.AddPortTCP(gopingPort), "Error adding port")
	require.NoError(t, mother.Commit(), "Error committing instance")

	err = mother.SetEnvironmentVariable("SERVE_ADDR", fmt.Sprintf("0.0.0.0:%d", gopingPort))
	require.NoError(t, err, "Error setting environment variable")

	target, err := mother.CloneWithName("target")
	require.NoError(t, err, "Error cloning instance")

	executor, err := mother.CloneWithName("executor")
	require.NoError(t, err, "Error cloning instance")

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		require.NoError(t, executor.Destroy(), "Error destroying executor instance")
		require.NoError(t, target.Destroy(), "Error destroying target instance")
	})

	// Prepare ping executor & target

	require.NoError(t, target.EnableBitTwister(), "Error enabling BitTwister")
	require.NoError(t, target.Start(), "Error starting target instance")

	forwardBitTwisterPort(t, target)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	require.NoError(t, target.BitTwister.WaitForStart(ctx), "Error waiting for BitTwister to start")

	require.NoError(t, executor.Start(), "Error starting executor instance")

	// Perform the test

	tt := []struct {
		name            string
		maxTargetJitter time.Duration
	}{
		{
			name:            "Max Jitter 10ms",
			maxTargetJitter: 10 * time.Millisecond,
		},
		{
			name:            "Max Jitter 20ms",
			maxTargetJitter: 20 * time.Millisecond,
		},
		{
			name:            "Max Jitter 50ms",
			maxTargetJitter: 50 * time.Millisecond,
		},
		{
			name:            "Max Jitter 100ms",
			maxTargetJitter: 100 * time.Millisecond,
		},
		{
			name:            "Max Jitter 200ms",
			maxTargetJitter: 200 * time.Millisecond,
		},
		{
			name:            "Max Jitter 500ms",
			maxTargetJitter: 500 * time.Millisecond,
		},
		{
			name:            "Max Jitter 1s",
			maxTargetJitter: 1 * time.Second,
		},
		{
			name:            "Max Jitter 2s",
			maxTargetJitter: 2 * time.Second,
		},
		{
			name:            "Max Jitter 3s",
			maxTargetJitter: 3 * time.Second,
		},
	}

	targetIP, err := target.GetIP()
	require.NoError(t, err, "Error getting IP")

	for _, tc := range tt {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err = target.SetLatencyAndJitter(0, tc.maxTargetJitter.Milliseconds())
			require.NoError(t, err, "Error setting packetloss rate")

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
			require.NoError(t, err, "Error executing command")

			elapsed := time.Since(startTime)
			t.Logf("Test took %d seconds", int64(elapsed.Seconds()))

			gotAvgJitter, err := time.ParseDuration(output)
			require.NoError(t, err, "Error parsing output")

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

func forwardBitTwisterPort(t *testing.T, i *knuu.Instance) {
	fwdBtPort, err := i.PortForwardTCP(i.BitTwister.Port())
	require.NoError(t, err, "Error port forwarding")
	i.BitTwister.SetPort(fwdBtPort)
	i.BitTwister.SetNewClientByURL("http://localhost")
	t.Logf("BitTwister listening on http://localhost:%d", fwdBtPort)
}
