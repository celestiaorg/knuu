package instance

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
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

func (i *Instance) createOtelCollectorInstance(ctx context.Context) (*Instance, error) {
	otelAgent, err := New("otel-agent", i.SystemDependencies)
	if err != nil {
		return nil, ErrCreatingOtelAgentInstance.Wrap(err)
	}

	if err := otelAgent.SetImage(ctx, fmt.Sprintf(otelAgentImage, i.obsyConfig.otelCollectorVersion)); err != nil {
		return nil, ErrSettingOtelAgentImage.Wrap(err)
	}
	if err := otelAgent.AddPortTCP(8888); err != nil {
		return nil, ErrAddingOtelAgentPort.Wrap(err)
	}
	if err := otelAgent.AddPortTCP(9090); err != nil {
		return nil, ErrAddingOtelAgentPort.Wrap(err)
	}
	if err := otelAgent.SetCPU(otelCpuLimit); err != nil {
		return nil, ErrSettingOtelAgentCPU.Wrap(err)
	}
	if err := otelAgent.SetMemory(otelMemoryRequest, otelMemoryLimit); err != nil {
		return nil, ErrSettingOtelAgentMemory.Wrap(err)
	}

	config := OTelConfig{
		Extensions: i.createExtensions(),
		Receivers:  i.createReceivers(),
		Exporters:  i.createExporters(),
		Service:    i.createService(),
		Processors: i.createProcessors(),
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return nil, ErrMarshalingYAML.Wrap(err)
	}

	if err := otelAgent.AddFileBytes(bytes, "/config/otel-agent.yaml", "0:0"); err != nil {
		return nil, ErrAddingOtelAgentConfigFile.Wrap(err)
	}

	if err := otelAgent.Commit(); err != nil {
		return nil, ErrCommittingOtelAgentInstance.Wrap(err)
	}

	if err := otelAgent.SetCommand("/otelcol-contrib", "--config=/config/otel-agent.yaml"); err != nil {
		return nil, ErrSettingOtelAgentCommand.Wrap(err)
	}

	return otelAgent, nil
}

func (i *Instance) createExtensions() Extensions {
	if i.obsyConfig.otlpEndpoint == "" {
		return Extensions{}
	}

	return Extensions{
		BasicAuthOTLP: BasicAuthOTLP{
			ClientAuth: ClientAuth{
				Username: i.obsyConfig.otlpUsername,
				Password: i.obsyConfig.otlpPassword,
			},
		},
	}
}

func (i *Instance) createOtlpReceiver() OTLP {
	return OTLP{
		Protocols: OTLPProtocols{
			HTTP: OTLPHTTP{
				Endpoint: fmt.Sprintf("localhost:%d", i.obsyConfig.otlpPort),
			},
		},
	}
}

func (i *Instance) createPrometheusReceiver() Prometheus {
	return Prometheus{
		Config: PrometheusConfig{
			ScrapeConfigs: []ScrapeConfig{
				{
					JobName:        i.obsyConfig.prometheusEndpointJobName,
					ScrapeInterval: i.obsyConfig.prometheusEndpointScrapeInterval,
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{fmt.Sprintf("localhost:%d", i.obsyConfig.prometheusEndpointPort)},
						},
					},
				},
				{
					JobName:        "internal-telemetry",
					ScrapeInterval: "10s",
					StaticConfigs: []StaticConfig{
						{
							Targets: []string{"localhost:8888"},
						},
					},
				},
			},
		},
	}
}

func (i *Instance) createJaegerReceiver() Jaeger {
	return Jaeger{
		Protocols: JaegerProtocols{
			GRPC: JaegerGRPC{Endpoint: fmt.Sprintf("localhost:%d", i.obsyConfig.jaegerGrpcPort)},
			ThriftCompact: JaegerThriftCompact{
				Endpoint: fmt.Sprintf("localhost:%d", i.obsyConfig.jaegerThriftCompactPort),
			},
			ThriftHTTP: JaegerThriftHTTP{
				Endpoint: fmt.Sprintf("localhost:%d", i.obsyConfig.jaegerThriftHttpPort),
			},
		},
	}
}

func (i *Instance) createReceivers() Receivers {
	receivers := Receivers{}

	if i.obsyConfig.otlpPort != 0 {
		receivers.OTLP = i.createOtlpReceiver()
	}

	if i.obsyConfig.prometheusEndpointPort != 0 {
		receivers.Prometheus = i.createPrometheusReceiver()
	}

	if i.obsyConfig.jaegerGrpcPort != 0 {
		receivers.Jaeger = i.createJaegerReceiver()
	}

	return receivers
}

func (i *Instance) createOtlpHttpExporter() OTLPHTTPExporter {
	return OTLPHTTPExporter{
		Auth: OTLPAuth{
			Authenticator: "basicauth/otlp",
		},
		Endpoint: i.obsyConfig.otlpEndpoint,
	}
}

func (i *Instance) createJaegerExporter() JaegerExporter {
	return JaegerExporter{
		Endpoint: i.obsyConfig.jaegerEndpoint,
		TLS: TLS{
			Insecure: true,
		},
	}
}

func (i *Instance) createPrometheusExporter() PrometheusExporter {
	return PrometheusExporter{
		Endpoint: i.obsyConfig.prometheusExporterEndpoint,
	}
}

func (i *Instance) createPrometheusRemoteWriteExporter() PrometheusRemoteWriteExporter {
	return PrometheusRemoteWriteExporter{
		Endpoint: i.obsyConfig.prometheusRemoteWriteExporterEndpoint,
		TLS: TLS{
			Insecure: true,
		},
	}
}

func (i *Instance) createExporters() Exporters {
	exporters := Exporters{}

	if i.obsyConfig.otlpEndpoint != "" {
		exporters.OTLPHTTP = i.createOtlpHttpExporter()
	}

	if i.obsyConfig.jaegerEndpoint != "" {
		exporters.Jaeger = i.createJaegerExporter()
	}

	if i.obsyConfig.prometheusExporterEndpoint != "" {
		exporters.Prometheus = i.createPrometheusExporter()
	}

	if i.obsyConfig.prometheusRemoteWriteExporterEndpoint != "" {
		exporters.PrometheusRemoteWrite = i.createPrometheusRemoteWriteExporter()
	}

	return exporters
}

func (i *Instance) prepareMetricsForServicePipeline() Metrics {
	metrics := Metrics{}
	if i.obsyConfig.otlpPort != 0 {
		metrics.Receivers = append(metrics.Receivers, "otlp")
	}
	if i.obsyConfig.prometheusEndpointPort != 0 {
		metrics.Receivers = append(metrics.Receivers, "prometheus")
	}
	if i.obsyConfig.otlpEndpoint != "" {
		metrics.Exporters = append(metrics.Exporters, "otlphttp")
	}
	if i.obsyConfig.prometheusExporterEndpoint != "" {
		metrics.Exporters = append(metrics.Exporters, "prometheus")
	}
	if i.obsyConfig.prometheusRemoteWriteExporterEndpoint != "" {
		metrics.Exporters = append(metrics.Exporters, "prometheusremotewrite")
	}
	metrics.Processors = []string{"attributes"}

	// if no trace receiver or exporter is added, remove any trace receiver
	if len(metrics.Receivers) == 0 || len(metrics.Exporters) == 0 {
		metrics = Metrics{}
	}
	return metrics
}

func (i *Instance) prepareTracesForServicePipeline() Traces {
	traces := Traces{}
	if i.obsyConfig.otlpPort != 0 {
		traces.Receivers = append(traces.Receivers, "otlp")
	}
	if i.obsyConfig.jaegerGrpcPort != 0 || i.obsyConfig.jaegerThriftCompactPort != 0 || i.obsyConfig.jaegerThriftHttpPort != 0 {
		traces.Receivers = append(traces.Receivers, "jaeger")
	}
	if i.obsyConfig.otlpEndpoint != "" {
		traces.Exporters = append(traces.Exporters, "otlphttp")
	}
	if i.obsyConfig.jaegerEndpoint != "" {
		traces.Exporters = append(traces.Exporters, "jaeger")
	}
	traces.Processors = []string{"attributes"}

	// if no trace receiver or exporter is added, remove any trace receiver
	if len(traces.Receivers) == 0 || len(traces.Exporters) == 0 {
		traces = Traces{}
	}

	return traces
}

func (i *Instance) createService() Service {
	var extensions []string
	if i.obsyConfig.otlpEndpoint != "" {
		extensions = append(extensions, "basicauth/otlp")
	}

	pipelines := Pipelines{}
	pipelines.Metrics = i.prepareMetricsForServicePipeline()
	pipelines.Traces = i.prepareTracesForServicePipeline()

	telemetry := Telemetry{
		Metrics: MetricsTelemetry{
			Address: "localhost:8888",
			Level:   "basic",
		},
	}

	return Service{
		Extensions: extensions,
		Pipelines:  pipelines,
		Telemetry:  telemetry,
	}
}

func (i *Instance) createProcessors() Processors {
	processors := Processors{}

	processors.Attributes = Attributes{
		Actions: []Action{
			{
				Key:    "namespace",
				Value:  i.K8sClient.Namespace(),
				Action: "insert",
			},
		},
	}

	return processors
}
