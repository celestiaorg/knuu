package obsy

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	localhostEndpoint = "localhost:%d"

	internalTelemetryEndpoint = "localhost:8888"
	internalTelemetryJobName  = "internal-telemetry"
	internalTelemetryInterval = "10s"
	basicTelemetryLevel       = "basic"

	basicAuthOTLPAuthAuthenticator = "basicauth/otlp"

	otlpReceiverName       = "otlp"
	prometheusReceiverName = "prometheus"
	jaegerReceiverName     = "jaeger"

	otlpHttpExporterName              = "otlphttp"
	jaegerExporterName                = "jaeger"
	prometheusExporterName            = "prometheus"
	prometheusRemoteWriteExporterName = "prometheusremotewrite"
	attributesProcessorName           = "attributes"

	scopeAttributeKey = "scope"
	insertAction      = "insert"

	// %s is the image tag, e.g. version
	otelAgentImage = "otel/opentelemetry-collector-contrib:%s"
)

var (
	otelMemoryRequest = resource.MustParse("100Mi")
	otelMemoryLimit   = resource.MustParse("200Mi")
	otelCpuLimit      = resource.MustParse("100m")
)

type OTelConfig struct {
	Extensions Extensions `yaml:"extensions,omitempty"`
	Receivers  Receivers  `yaml:"receivers,omitempty"`
	Exporters  Exporters  `yaml:"exporters,omitempty"`
	Service    Service    `yaml:"service,omitempty"`
	Processors Processors `yaml:"processors,omitempty"`
}

type Extensions struct {
	BasicAuthOTLP BasicAuthOTLP `yaml:"basicauth/otlp,omitempty"`
}

type BasicAuthOTLP struct {
	ClientAuth ClientAuth `yaml:"client_auth,omitempty"`
}

type ClientAuth struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

type Receivers struct {
	OTLP       OTLP       `yaml:"otlp,omitempty"`
	Prometheus Prometheus `yaml:"prometheus,omitempty"`
	Jaeger     Jaeger     `yaml:"jaeger,omitempty"`
}

type OTLP struct {
	Protocols OTLPProtocols `yaml:"protocols,omitempty"`
}

type OTLPProtocols struct {
	HTTP OTLPHTTP `yaml:"http,omitempty"`
}

type OTLPHTTP struct {
	Endpoint string `yaml:"endpoint,omitempty"`
}

type Prometheus struct {
	Config PrometheusConfig `yaml:"config,omitempty"`
}

type PrometheusConfig struct {
	ScrapeConfigs []ScrapeConfig `yaml:"scrape_configs,omitempty"`
}

type ScrapeConfig struct {
	JobName        string         `yaml:"job_name,omitempty"`
	ScrapeInterval string         `yaml:"scrape_interval,omitempty"`
	StaticConfigs  []StaticConfig `yaml:"static_configs,omitempty"`
}

type StaticConfig struct {
	Targets []string `yaml:"targets,omitempty"`
}

type Jaeger struct {
	Protocols JaegerProtocols `yaml:"protocols,omitempty"`
}

type JaegerProtocols struct {
	GRPC          JaegerGRPC          `yaml:"grpc,omitempty"`
	ThriftCompact JaegerThriftCompact `yaml:"thrift_compact,omitempty"`
	ThriftHTTP    JaegerThriftHTTP    `yaml:"thrift_http,omitempty"`
}

type JaegerGRPC struct {
	Endpoint string `yaml:"endpoint,omitempty"`
}

type JaegerThriftCompact struct {
	Endpoint string `yaml:"endpoint,omitempty"`
}

type JaegerThriftHTTP struct {
	Endpoint string `yaml:"endpoint,omitempty"`
}

type Exporters struct {
	OTLPHTTP              OTLPHTTPExporter              `yaml:"otlphttp,omitempty"`
	Jaeger                JaegerExporter                `yaml:"jaeger,omitempty"`
	Prometheus            PrometheusExporter            `yaml:"prometheus,omitempty"`
	PrometheusRemoteWrite PrometheusRemoteWriteExporter `yaml:"prometheusremotewrite,omitempty"`
}

type OTLPHTTPExporter struct {
	Auth     OTLPAuth `yaml:"auth,omitempty"`
	Endpoint string   `yaml:"endpoint,omitempty"`
}

type OTLPAuth struct {
	Authenticator string `yaml:"authenticator,omitempty"`
}

type JaegerExporter struct {
	Endpoint string `yaml:"endpoint,omitempty"`
	TLS      TLS    `yaml:"tls,omitempty"`
}

type PrometheusExporter struct {
	Endpoint string `yaml:"endpoint,omitempty"`
}

type PrometheusRemoteWriteExporter struct {
	Endpoint string `yaml:"endpoint,omitempty"`
	TLS      TLS    `yaml:"tls,omitempty"`
}

type TLS struct {
	Insecure bool `yaml:"insecure,omitempty"`
}

type Service struct {
	Extensions []string  `yaml:"extensions,omitempty"`
	Pipelines  Pipelines `yaml:"pipelines,omitempty"`
	Telemetry  Telemetry `yaml:"telemetry,omitempty"` // Added Telemetry field
}

type Telemetry struct {
	Metrics MetricsTelemetry `yaml:"metrics,omitempty"`
}

type MetricsTelemetry struct {
	Address string `yaml:"address,omitempty"`
	Level   string `yaml:"level,omitempty"` // Options are basic, normal, detailed
}

type Pipelines struct {
	Metrics Metrics `yaml:"metrics,omitempty"`
	Traces  Traces  `yaml:"traces,omitempty"`
}

type Metrics struct {
	Receivers  []string `yaml:"receivers,omitempty"`
	Exporters  []string `yaml:"exporters,omitempty"`
	Processors []string `yaml:"processors,omitempty"`
}

type Traces struct {
	Receivers  []string `yaml:"receivers,omitempty"`
	Exporters  []string `yaml:"exporters,omitempty"`
	Processors []string `yaml:"processors,omitempty"`
}

type Processors struct {
	Batch         Batch         `yaml:"batch,omitempty"`
	MemoryLimiter MemoryLimiter `yaml:"memory_limiter,omitempty"`
	Attributes    Attributes    `yaml:"attributes,omitempty"`
}

type Batch struct{}

type MemoryLimiter struct {
	LimitMiB      int    `yaml:"limit_mib,omitempty"`
	SpikeLimitMiB int    `yaml:"spike_limit_mib,omitempty"`
	CheckInterval string `yaml:"check_interval,omitempty"`
}

type Attributes struct {
	Actions []Action `yaml:"actions,omitempty"`
}

type Action struct {
	Key    string `yaml:"key,omitempty"`
	Value  string `yaml:"value,omitempty"`
	Action string `yaml:"action,omitempty"`
}

func (o *Obsy) createExtensions() Extensions {
	if o.obsyConfig.otlpEndpoint == "" || o.obsyConfig.otlpUsername == "" || o.obsyConfig.otlpPassword == "" {
		return Extensions{}
	}

	return Extensions{
		BasicAuthOTLP: BasicAuthOTLP{
			ClientAuth: ClientAuth{
				Username: o.obsyConfig.otlpUsername,
				Password: o.obsyConfig.otlpPassword,
			},
		},
	}
}

func (o *Obsy) createOtlpReceiver() OTLP {
	return OTLP{
		Protocols: OTLPProtocols{
			HTTP: OTLPHTTP{
				Endpoint: fmt.Sprintf(localhostEndpoint, o.obsyConfig.otlpPort),
			},
		},
	}
}

func (o *Obsy) createPrometheusReceiver() Prometheus {
	return Prometheus{
		Config: PrometheusConfig{
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName:        o.obsyConfig.prometheusEndpointJobName,
					ScrapeInterval: o.obsyConfig.prometheusEndpointScrapeInterval,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{fmt.Sprintf(localhostEndpoint, o.obsyConfig.prometheusEndpointPort)},
						},
					},
				},
				{
					JobName:        internalTelemetryJobName,
					ScrapeInterval: internalTelemetryInterval,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{internalTelemetryEndpoint},
						},
					},
				},
			},
		},
	}
}

func (o *Obsy) createJaegerReceiver() Jaeger {
	return Jaeger{
		Protocols: JaegerProtocols{
			GRPC: JaegerGRPC{Endpoint: fmt.Sprintf(localhostEndpoint, o.obsyConfig.jaegerGrpcPort)},
			ThriftCompact: JaegerThriftCompact{
				Endpoint: fmt.Sprintf(localhostEndpoint, o.obsyConfig.jaegerThriftCompactPort),
			},
			ThriftHTTP: JaegerThriftHTTP{
				Endpoint: fmt.Sprintf(localhostEndpoint, o.obsyConfig.jaegerThriftHttpPort),
			},
		},
	}
}

func (o *Obsy) createReceivers() Receivers {
	receivers := Receivers{}

	if o.obsyConfig.otlpPort != 0 {
		receivers.OTLP = o.createOtlpReceiver()
	}

	if o.obsyConfig.prometheusEndpointPort != 0 {
		receivers.Prometheus = o.createPrometheusReceiver()
	}

	if o.obsyConfig.jaegerGrpcPort != 0 {
		receivers.Jaeger = o.createJaegerReceiver()
	}

	return receivers
}

func (o *Obsy) createOtlpHttpExporter() OTLPHTTPExporter {
	exporter := OTLPHTTPExporter{
		Endpoint: o.obsyConfig.otlpEndpoint,
	}

	if o.obsyConfig.otlpUsername != "" && o.obsyConfig.otlpPassword != "" {
		exporter.Auth = OTLPAuth{
			Authenticator: basicAuthOTLPAuthAuthenticator,
		}
	}

	return exporter
}

func (o *Obsy) createJaegerExporter() JaegerExporter {
	return JaegerExporter{
		Endpoint: o.obsyConfig.jaegerEndpoint,
		TLS: TLS{
			Insecure: true,
		},
	}
}

func (o *Obsy) createPrometheusExporter() PrometheusExporter {
	return PrometheusExporter{
		Endpoint: o.obsyConfig.prometheusExporterEndpoint,
	}
}

func (o *Obsy) createPrometheusRemoteWriteExporter() PrometheusRemoteWriteExporter {
	return PrometheusRemoteWriteExporter{
		Endpoint: o.obsyConfig.prometheusRemoteWriteExporterEndpoint,
		TLS: TLS{
			Insecure: true,
		},
	}
}

func (o *Obsy) createExporters() Exporters {
	exporters := Exporters{}

	if o.obsyConfig.otlpEndpoint != "" {
		exporters.OTLPHTTP = o.createOtlpHttpExporter()
	}

	if o.obsyConfig.jaegerEndpoint != "" {
		exporters.Jaeger = o.createJaegerExporter()
	}

	if o.obsyConfig.prometheusEndpointPort != 0 {
		exporters.Prometheus = o.createPrometheusExporter()
	}

	if o.obsyConfig.prometheusRemoteWriteExporterEndpoint != "" {
		exporters.PrometheusRemoteWrite = o.createPrometheusRemoteWriteExporter()
	}

	return exporters
}

func (o *Obsy) prepareMetricsForServicePipeline() Metrics {
	metrics := Metrics{}
	if o.obsyConfig.otlpPort != 0 {
		metrics.Receivers = append(metrics.Receivers, otlpReceiverName)
	}
	if o.obsyConfig.prometheusEndpointPort != 0 {
		metrics.Receivers = append(metrics.Receivers, prometheusReceiverName)
	}
	if o.obsyConfig.otlpEndpoint != "" {
		metrics.Exporters = append(metrics.Exporters, otlpHttpExporterName)
	}
	if o.obsyConfig.prometheusExporterEndpoint != "" {
		metrics.Exporters = append(metrics.Exporters, prometheusExporterName)
	}
	if o.obsyConfig.prometheusRemoteWriteExporterEndpoint != "" {
		metrics.Exporters = append(metrics.Exporters, prometheusRemoteWriteExporterName)
	}
	metrics.Processors = []string{attributesProcessorName}
	return metrics
}

func (o *Obsy) prepareTracesForServicePipeline() Traces {
	traces := Traces{}
	if o.obsyConfig.otlpPort != 0 {
		traces.Receivers = append(traces.Receivers, otlpReceiverName)
	}
	if o.obsyConfig.jaegerGrpcPort != 0 || o.obsyConfig.jaegerThriftCompactPort != 0 || o.obsyConfig.jaegerThriftHttpPort != 0 {
		traces.Receivers = append(traces.Receivers, jaegerReceiverName)
	}
	if o.obsyConfig.otlpEndpoint != "" {
		traces.Exporters = append(traces.Exporters, otlpHttpExporterName)
	}
	if o.obsyConfig.jaegerEndpoint != "" {
		traces.Exporters = append(traces.Exporters, jaegerExporterName)
	}
	traces.Processors = []string{attributesProcessorName}
	return traces
}

func (o *Obsy) createService() Service {
	var extensions []string
	if o.obsyConfig.otlpEndpoint != "" && o.obsyConfig.otlpUsername != "" && o.obsyConfig.otlpPassword != "" {
		extensions = append(extensions, basicAuthOTLPAuthAuthenticator)
	}

	pipelines := Pipelines{}
	pipelines.Metrics = o.prepareMetricsForServicePipeline()
	pipelines.Traces = o.prepareTracesForServicePipeline()

	telemetry := Telemetry{
		Metrics: MetricsTelemetry{
			Address: internalTelemetryEndpoint,
			Level:   basicTelemetryLevel,
		},
	}

	return Service{
		Extensions: extensions,
		Pipelines:  pipelines,
		Telemetry:  telemetry,
	}
}

func (o *Obsy) createProcessors(scope string) Processors {
	processors := Processors{}

	processors.Attributes = Attributes{
		Actions: []Action{
			{
				Key:    scopeAttributeKey,
				Value:  scope,
				Action: insertAction,
			},
		},
	}

	return processors
}
