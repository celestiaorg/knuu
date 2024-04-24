package celestia_app

import (
	"fmt"
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"

	app_utils "github.com/celestiaorg/knuu-example/celestia_app/utils"
	"github.com/celestiaorg/knuu-example/celestia_node/utils"
)

func TestOtel(t *testing.T) {
	t.Parallel()
	// Setup

	grafanaEndpoint := os.Getenv("GRAFANA_ENDPOINT")
	if grafanaEndpoint == "" {
		t.Fatal("GRAFANA_ENDPOINT env var must be set")
	}

	grafanaUsername := os.Getenv("GRAFANA_USERNAME")
	if grafanaUsername == "" {
		t.Fatal("GRAFANA_USERNAME env var must be set")
	}
	grafanaToken := os.Getenv("GRAFANA_TOKEN")
	if grafanaToken == "" {
		t.Fatal("GRAFANA_TOKEN env var must be set")
	}

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	t.Log("Starting consensus")
	consensus, err := app_utils.CreateAndStartConsensus(executor)
	if err != nil {
		t.Fatalf("Error creating and starting consensus: %v", err)
	}

	t.Log("Starting bridge")
	bridge, err := utils.CreateBridge(executor, "bridge", consensus)
	if err := bridge.SetOtelEndpoint(4318); err != nil {
		t.Fatalf("Error setting otel endpoint: %v", err)
	}
	if err := bridge.SetPrometheusEndpoint(8890, fmt.Sprintf("knuu-%s", knuu.Scope()), "1m"); err != nil {
		t.Fatalf("Error setting prometheus endpoint: %v", err)
	}
	if err := bridge.SetJaegerEndpoint(14250, 6831, 14268); err != nil {
		t.Fatalf("Error setting jaeger endpoint: %v", err)
	}
	if err := bridge.SetOtlpExporter(grafanaEndpoint, grafanaUsername, grafanaToken); err != nil {
		t.Fatalf("Error setting otlp exporter: %v", err)
	}
	if err := bridge.SetJaegerExporter("jaeger-collector.jaeger-cluster.svc.cluster.local:14250"); err != nil {
		t.Fatalf("Error setting jaeger exporter: %v", err)
	}
	if err := bridge.Start(); err != nil {
		t.Fatalf("Error starting bridge: %v", err)
	}

	t.Log("Starting full node")
	full, err := utils.CreateNode(executor, "full", "full", consensus, bridge)
	if err := full.SetOtelEndpoint(4318); err != nil {
		t.Fatalf("Error setting otel endpoint: %v", err)
	}
	if err := full.SetPrometheusEndpoint(8890, knuu.Scope(), "10s"); err != nil {
		t.Fatalf("Error setting prometheus endpoint: %v", err)
	}
	if err := full.SetJaegerEndpoint(14250, 6831, 14268); err != nil {
		t.Fatalf("Error setting jaeger endpoint: %v", err)
	}
	if err := full.SetOtlpExporter(grafanaEndpoint, grafanaUsername, grafanaToken); err != nil {
		t.Fatalf("Error setting otlp exporter: %v", err)
	}
	if err := full.SetJaegerExporter("jaeger-collector.jaeger-cluster.svc.cluster.local:14250"); err != nil {
		t.Fatalf("Error setting jaeger exporter: %v", err)
	}
	if err := full.Start(); err != nil {
		t.Fatalf("Error starting bridge: %v", err)
	}

	t.Log("Starting light node")
	light, err := utils.CreateNode(executor, "light", "light", consensus, full)
	if err := light.SetOtelEndpoint(4318); err != nil {
		t.Fatalf("Error setting otel endpoint: %v", err)
	}
	if err := light.SetPrometheusEndpoint(8890, knuu.Scope(), "10s"); err != nil {
		t.Fatalf("Error setting prometheus endpoint: %v", err)
	}
	if err := light.SetJaegerEndpoint(14250, 6831, 14268); err != nil {
		t.Fatalf("Error setting jaeger endpoint: %v", err)
	}
	if err := light.SetOtlpExporter(grafanaEndpoint, grafanaUsername, grafanaToken); err != nil {
		t.Fatalf("Error setting otlp exporter: %v", err)
	}
	if err := light.SetJaegerExporter("jaeger-collector.jaeger-cluster.svc.cluster.local:14250"); err != nil {
		t.Fatalf("Error setting jaeger exporter: %v", err)
	}
	if err := light.Start(); err != nil {
		t.Fatalf("Error starting bridge: %v", err)
	}

	t.Cleanup(func() {
		// Cleanup
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		err = executor.Destroy()
		if err != nil {
			t.Fatalf("Error destroying executor: %v", err)
		}
		err = consensus.Destroy()
		if err != nil {
			t.Fatalf("Error destroying executor: %v", err)
		}
		err = bridge.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
		err = full.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
		err = light.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
	})

	// Test logic

}
