package basic

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/celestiaorg/knuu/pkg/sidecars/observability"
)

const (
	prometheusPort   = observability.DefaultOtelMetricsPort
	prometheusImage  = "prom/prometheus:latest"
	prometheusConfig = "/etc/prometheus/prometheus.yml"
	prometheusArgs   = "--config.file=/etc/prometheus/prometheus.yml"

	targetImage = "curlimages/curl:latest"
	otlpPort    = observability.DefaultOtelOtlpPort
)

// TestObservabilityCollector is a test function that verifies the functionality of the otel collector setup
func TestObservabilityCollector(t *testing.T) {
	t.Parallel()

	// Setup Prometheus
	prometheus, err := knuu.NewInstance("prometheus")
	require.NoError(t, err)

	require.NoError(t, prometheus.SetImage(prometheusImage))
	require.NoError(t, prometheus.AddPortTCP(prometheusPort))

	// enable proxy for this port
	err, prometheusEndpoint := prometheus.AddHost(prometheusPort)
	require.NoError(t, err)

	require.NoError(t, prometheus.Commit())

	// Add Prometheus config file
	prometheusConfigContent := fmt.Sprintf(`
global:
  scrape_interval: '10s'
scrape_configs:
  - job_name: 'otel-collector'
    static_configs:
      - targets: ['otel-collector:%d']
`, otlpPort)
	require.NoError(t, prometheus.AddFileBytes([]byte(prometheusConfigContent), prometheusConfig, "0:0"))

	require.NoError(t, prometheus.SetArgs(prometheusArgs))
	require.NoError(t, prometheus.Start())

	// Setup observabilitySidecar collector
	observabilitySidecar := observability.New()

	require.NoError(t, observabilitySidecar.SetOtelEndpoint(4318))

	err = observabilitySidecar.SetPrometheusEndpoint(otlpPort, fmt.Sprintf("knuu-%s", knuu.Scope()), "10s")
	require.NoError(t, err)

	require.NoError(t, observabilitySidecar.SetJaegerEndpoint(14250, 6831, 14268))

	require.NoError(t, observabilitySidecar.SetOtlpExporter("prometheus:9090", "", ""))

	// Create and start a target pod and configure it to use the obsySidecar to push metrics
	target, err := knuu.NewInstance("target")
	require.NoError(t, err, "Error creating target instance")

	err = target.SetImage(targetImage)
	require.NoError(t, err, "Error setting target image")

	err = target.SetStartCommand("sh", "-c", "while true; do curl -X POST http://localhost:8888/v1/traces; sleep 5; done")
	require.NoError(t, err, "Error setting target command")

	require.NoError(t, target.AddSidecar(context.Background(), observabilitySidecar))

	require.NoError(t, target.Commit(), "Error committing target instance")

	require.NoError(t, target.Start(), "Error starting target instance")

	t.Cleanup(func() {
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}
		err := knuu.BatchDestroy(prometheus, target)
		if err != nil {
			t.Log("Error destroying instances: ", err)
		}
	})

	// Wait for the target pod to push data to the otel collector
	time.Sleep(1 * time.Minute)

	// Verify that data has been pushed to Prometheus

	prometheusURL := fmt.Sprintf("%s/api/v1/query?query=up", prometheusEndpoint)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", prometheusURL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode, "Prometheus API is not accessible")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "otel-collector", "otel-collector data source not found in Prometheus")

	t.Log("otel-collector data source is available in Prometheus")
}
