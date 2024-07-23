package obsy

import (
	"github.com/celestiaorg/knuu/pkg/instance"
)

func (o *Obsy) SetImage(image string) {
	o.image = image
}

// SetOtelCollectorVersion sets the OpenTelemetry collector version for the instance
func (o *Obsy) SetOtelCollectorVersion(version string) error {
	if err := o.validateStateForObsy("OpenTelemetry collector version"); err != nil {
		return err
	}
	o.obsyConfig.otelCollectorVersion = version
	return nil
}

// SetOtelEndpoint sets the OpenTelemetry endpoint for the instance
func (o *Obsy) SetOtelEndpoint(port int) error {
	if err := o.validateStateForObsy("OpenTelemetry endpoint"); err != nil {
		return err
	}
	o.obsyConfig.otlpPort = port
	return nil
}

// SetPrometheusEndpoint sets the Prometheus endpoint for the instance
func (o *Obsy) SetPrometheusEndpoint(port int, jobName, scapeInterval string) error {
	if err := o.validateStateForObsy("Prometheus endpoint"); err != nil {
		return err
	}
	o.obsyConfig.prometheusEndpointPort = port
	o.obsyConfig.prometheusEndpointJobName = jobName
	o.obsyConfig.prometheusEndpointScrapeInterval = scapeInterval
	return nil
}

// SetJaegerEndpoint sets the Jaeger endpoint for the instance
func (o *Obsy) SetJaegerEndpoint(grpcPort, thriftCompactPort, thriftHttpPort int) error {
	if err := o.validateStateForObsy("Jaeger endpoint"); err != nil {
		return err
	}
	o.obsyConfig.jaegerGrpcPort = grpcPort
	o.obsyConfig.jaegerThriftCompactPort = thriftCompactPort
	o.obsyConfig.jaegerThriftHttpPort = thriftHttpPort
	return nil
}

// SetOtlpExporter sets the OTLP exporter for the instance
func (o *Obsy) SetOtlpExporter(endpoint, username, password string) error {
	if err := o.validateStateForObsy("OTLP exporter"); err != nil {
		return err
	}
	o.obsyConfig.otlpEndpoint = endpoint
	o.obsyConfig.otlpUsername = username
	o.obsyConfig.otlpPassword = password
	return nil
}

// SetJaegerExporter sets the Jaeger exporter for the instance
func (o *Obsy) SetJaegerExporter(endpoint string) error {
	if err := o.validateStateForObsy("Jaeger exporter"); err != nil {
		return err
	}
	o.obsyConfig.jaegerEndpoint = endpoint
	return nil
}

// SetPrometheusExporter sets the Prometheus exporter for the instance
func (o *Obsy) SetPrometheusExporter(endpoint string) error {
	if err := o.validateStateForObsy("Prometheus exporter"); err != nil {
		return err
	}
	o.obsyConfig.prometheusExporterEndpoint = endpoint
	return nil
}

// SetPrometheusRemoteWriteExporter sets the Prometheus remote write exporter for the instance
func (o *Obsy) SetPrometheusRemoteWriteExporter(endpoint string) error {
	if err := o.validateStateForObsy("Prometheus remote write exporter"); err != nil {
		return err
	}
	o.obsyConfig.prometheusRemoteWriteExporterEndpoint = endpoint
	return nil
}

func (o *Obsy) validateStateForObsy(endpoint string) error {
	if o.instance != nil && !o.instance.IsInState(instance.StateNone) {
		return ErrSettingNotAllowed.WithParams(endpoint, o.instance.State().String())
	}
	return nil
}
