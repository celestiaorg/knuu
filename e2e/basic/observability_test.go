package basic

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/celestiaorg/knuu/pkg/sidecars/observability"
)

const (
	prometheusPort   = observability.DefaultOtelMetricsPort
	prometheusImage  = "prom/prometheus:latest"
	prometheusConfig = "/etc/prometheus/prometheus.yml"
	prometheusArgs   = "--config.file=" + prometheusConfig

	curlImage = "curlimages/curl:latest"
	otlpPort  = observability.DefaultOtelOtlpPort

	retryInterval = 1 * time.Second
	retryTimeout  = 10 * time.Second
)

// TestObservabilityCollector is a test function that verifies the functionality of the otel collector setup
func (s *Suite) TestObservabilityCollector() {
	const (
		namePrefix             = "observability"
		scrapeInterval         = "2s"
		prometheusQueryTimeout = 30 * time.Second
	)
	var (
		targetStartCommand = fmt.Sprintf("while true; do curl -X POST http://localhost:%d/v1/traces; sleep 2; done", otlpPort)
		ctx                = context.Background()
	)

	// Setup Prometheus
	prometheus, err := s.Knuu.NewInstance(namePrefix + "-prometheus")
	s.Require().NoError(err)

	s.Require().NoError(prometheus.Build().SetImage(ctx, prometheusImage))
	s.Require().NoError(prometheus.Network().AddPortTCP(prometheusPort))

	// enable proxy for this port
	prometheusEndpoint, err := prometheus.Network().AddHost(ctx, prometheusPort)
	s.Require().NoError(err)

	s.Require().NoError(prometheus.Build().Commit(ctx))

	// Add Prometheus config file
	prometheusConfigContent := fmt.Sprintf(`
global:
  scrape_interval: '%s'

scrape_configs:
  - job_name: 'otel-collector'
    static_configs:
      - targets: ['otel-collector:%d']`, scrapeInterval, otlpPort)
	s.Require().NoError(prometheus.Storage().AddFileBytes([]byte(prometheusConfigContent), prometheusConfig, "0:0"))

	s.Require().NoError(prometheus.Build().SetArgs(prometheusArgs))
	s.Require().NoError(prometheus.Execution().Start(ctx))

	// Setup observabilitySidecar collector
	observabilitySidecar := observability.New()

	s.Require().NoError(observabilitySidecar.SetOtelEndpoint(4318))
	s.Require().NoError(observabilitySidecar.SetPrometheusEndpoint(otlpPort, fmt.Sprintf("knuu-%s", s.Knuu.Scope), scrapeInterval))
	s.Require().NoError(observabilitySidecar.SetJaegerEndpoint(14250, 6831, 14268))
	s.Require().NoError(observabilitySidecar.SetOtlpExporter("prometheus:9090", "", ""))

	// Create and start a target pod and configure it to use the obsySidecar to push metrics
	target, err := s.Knuu.NewInstance(namePrefix + "-target")
	s.Require().NoError(err)

	s.Require().NoError(target.Build().SetImage(ctx, curlImage))

	err = target.Build().SetStartCommand("sh", "-c", targetStartCommand)
	s.Require().NoError(err)

	s.Require().NoError(target.Sidecars().Add(ctx, observabilitySidecar))

	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	// Verify that data has been pushed to Prometheus
	s.Require().Eventually(func() bool {
		url := fmt.Sprintf("%s/api/v1/query?query=up", prometheusEndpoint)
		ctx, cancel := context.WithTimeout(context.Background(), prometheusQueryTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			s.T().Logf("Error creating request: %v", err)
			return false
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			s.T().Logf("Error sending request: %v", err)
			return false
		}
		if resp.StatusCode != http.StatusOK {
			s.T().Logf("Prometheus API returned status code: %d", resp.StatusCode)
			return false
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			s.T().Logf("Error reading response body: %v", err)
			return false
		}
		return strings.Contains(string(body), "otel-collector")

	}, retryTimeout, retryInterval, "otel-collector data source not found in Prometheus")
}

func (s *Suite) TestObservabilityCollectorWithLogging() {
	const (
		namePrefix         = "observability"
		targetStartCommand = "while true; do curl -X POST http://localhost:8888/v1/traces; sleep 2; done"
	)
	ctx := context.Background()

	// Setup obsySidecar collector
	obsySidecar := observability.New()

	s.Require().NoError(obsySidecar.SetOtelEndpoint(4318))
	s.Require().NoError(obsySidecar.SetLoggingExporter("debug"))

	// Create and start a target pod and configure it to use the obsySidecar to push metrics
	target, err := s.Knuu.NewInstance(namePrefix + "target")
	s.Require().NoError(err)

	s.Require().NoError(target.Build().SetImage(ctx, curlImage))

	err = target.Build().SetStartCommand("sh", "-c", targetStartCommand)
	s.Require().NoError(err)

	s.Require().NoError(target.Sidecars().Add(ctx, obsySidecar))
	s.Require().NoError(target.Build().Commit(ctx))
	s.Require().NoError(target.Execution().Start(ctx))

	// Verify that data has been pushed to the logging exporter
	s.Require().Eventually(func() bool {
		logsReader, err := obsySidecar.Instance().Monitoring().Logs(ctx)
		if err != nil {
			s.T().Logf("Error getting logs: %v", err)
			return false
		}
		logsOutput, err := io.ReadAll(logsReader)
		if err != nil {
			s.T().Logf("Error reading logs: %v", err)
			return false
		}

		loggingExporterPattern := regexp.MustCompile(`"kind": "exporter", "data_type": "metrics", "name": "logging"`)
		if !loggingExporterPattern.Match(logsOutput) {
			s.T().Logf("Logging exporter not found in the logs: `%s`", string(logsOutput))
			return false
		}
		return true
	}, retryTimeout, retryInterval, "Logging exporter not found in the logs")
}
