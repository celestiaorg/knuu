package knuu

import (
	"fmt"
	"gopkg.in/yaml.v3"
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
	OTLPHTTP OTLPHTTPExporter `yaml:"otlphttp,omitempty"`
	Jaeger   JaegerExporter   `yaml:"jaeger,omitempty"`
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

type TLS struct {
	Insecure bool `yaml:"insecure,omitempty"`
}

type Service struct {
	Extensions []string  `yaml:"extensions,omitempty"`
	Pipelines  Pipelines `yaml:"pipelines,omitempty"`
}

type Pipelines struct {
	Metrics Metrics `yaml:"metrics,omitempty"`
	Traces  Traces  `yaml:"traces,omitempty"`
}

type Metrics struct {
	Receivers []string `yaml:"receivers,omitempty"`
	Exporters []string `yaml:"exporters,omitempty"`
}

type Traces struct {
	Receivers  []string `yaml:"receivers,omitempty"`
	Exporters  []string `yaml:"exporters,omitempty"`
	Processors []string `yaml:"processors,omitempty"`
}

type Processors struct {
	Batch         Batch         `yaml:"batch,omitempty"`
	MemoryLimiter MemoryLimiter `yaml:"memory_limiter,omitempty"`
}

type Batch struct{}

type MemoryLimiter struct {
	LimitMiB      int    `yaml:"limit_mib,omitempty"`
	SpikeLimitMiB int    `yaml:"spike_limit_mib,omitempty"`
	CheckInterval string `yaml:"check_interval,omitempty"`
}

func (i *Instance) createOtelCollectorInstance() (*Instance, error) {
	otelAgent, err := NewInstance("otel-agent")
	if err != nil {
		return nil, fmt.Errorf("error creating otel-agent instance: %w", err)
	}

	if err := otelAgent.SetImage("otel/opentelemetry-collector-contrib:0.71.0"); err != nil {
		return nil, fmt.Errorf("error setting image for otel-agent instance: %w", err)
	}
	if err := otelAgent.AddPortTCP(8888); err != nil {
		return nil, fmt.Errorf("error adding port for otel-agent instance: %w", err)
	}
	if err := otelAgent.AddPortTCP(9090); err != nil {
		return nil, fmt.Errorf("error adding port for otel-agent instance: %w", err)
	}
	if err := otelAgent.SetCPU("100m"); err != nil {
		return nil, fmt.Errorf("error setting CPU for otel-agent instance: %w", err)
	}
	if err := otelAgent.SetMemory("100Mi", "200Mi"); err != nil {
		return nil, fmt.Errorf("error setting memory for otel-agent instance: %w", err)
	}
	if err := otelAgent.Commit(); err != nil {
		return nil, fmt.Errorf("error committing otel-agent instance: %w", err)
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
		return nil, fmt.Errorf("error marshaling YAML: %w", err)
	}

	if err := otelAgent.AddFileBytes(bytes, "/etc/otel-agent.yaml", "root:root"); err != nil {
		return nil, fmt.Errorf("error adding otel-agent config file: %w", err)
	}

	if err := otelAgent.SetCommand("/otelcol-contrib", "--config=/etc/otel-agent.yaml"); err != nil {
		return nil, fmt.Errorf("error setting command for otel-agent instance: %w", err)
	}

	return otelAgent, nil
}

func (i *Instance) createExtensions() Extensions {
	if i.otlpEndpoint == "" {
		return Extensions{}
	}

	return Extensions{
		BasicAuthOTLP: BasicAuthOTLP{
			ClientAuth: ClientAuth{
				Username: i.otlpUsername,
				Password: i.otlpPassword,
			},
		},
	}
}

func (i *Instance) createReceivers() Receivers {
	receivers := Receivers{}

	if i.otlpPort != 0 {
		receivers.OTLP = OTLP{
			Protocols: OTLPProtocols{
				HTTP: OTLPHTTP{
					Endpoint: fmt.Sprintf("localhost:%d", i.otlpPort),
				},
			},
		}
	}

	if i.prometheusPort != 0 {
		receivers.Prometheus = Prometheus{
			Config: PrometheusConfig{
				ScrapeConfigs: []ScrapeConfig{
					{
						JobName:        i.prometheusJobName,
						ScrapeInterval: i.prometheusScrapeInterval,
						StaticConfigs: []StaticConfig{
							{
								Targets: []string{fmt.Sprintf("localhost:%d", i.prometheusPort)},
							},
						},
					},
				},
			},
		}
	}

	if i.jaegerGrpcPort != 0 {
		receivers.Jaeger = Jaeger{
			Protocols: JaegerProtocols{
				GRPC: JaegerGRPC{Endpoint: fmt.Sprintf("localhost:%d", i.jaegerGrpcPort)},
				ThriftCompact: JaegerThriftCompact{
					Endpoint: fmt.Sprintf("localhost:%d", i.jaegerThriftCompactPort),
				},
				ThriftHTTP: JaegerThriftHTTP{
					Endpoint: fmt.Sprintf("localhost:%d", i.jaegerThriftHttpPort),
				},
			},
		}
	}

	return receivers
}

func (i *Instance) createExporters() Exporters {
	exporters := Exporters{}

	if i.otlpEndpoint != "" {
		exporters.OTLPHTTP = OTLPHTTPExporter{
			Auth: OTLPAuth{
				Authenticator: "basicauth/otlp",
			},
			Endpoint: i.otlpEndpoint,
		}
	}

	if i.jaegerEndpoint != "" {
		exporters.Jaeger = JaegerExporter{
			Endpoint: i.jaegerEndpoint,
			TLS: TLS{
				Insecure: true,
			},
		}
	}

	return exporters
}

func (i *Instance) createService() Service {
	var extensions []string
	if i.otlpEndpoint != "" {
		extensions = append(extensions, "basicauth/otlp")
	}

	pipelines := Pipelines{}
	if i.otlpPort != 0 || i.prometheusPort != 0 || i.otlpEndpoint != "" {
		metrics := Metrics{}
		if i.otlpPort != 0 {
			metrics.Receivers = append(metrics.Receivers, "otlp")
		}
		if i.prometheusPort != 0 {
			metrics.Receivers = append(metrics.Receivers, "prometheus")
		}
		if i.otlpEndpoint != "" {
			metrics.Exporters = append(metrics.Exporters, "otlphttp")
		}
		pipelines.Metrics = metrics
	}
	if i.jaegerGrpcPort != 0 || i.jaegerThriftCompactPort != 0 || i.jaegerThriftHttpPort != 0 || i.jaegerEndpoint != "" {
		traces := Traces{}
		if i.otlpPort != 0 {
			traces.Receivers = append(traces.Receivers, "otlp")
		}
		if i.jaegerGrpcPort != 0 || i.jaegerThriftCompactPort != 0 || i.jaegerThriftHttpPort != 0 {
			traces.Receivers = append(traces.Receivers, "jaeger")
		}
		if i.otlpEndpoint != "" {
			traces.Exporters = append(traces.Exporters, "otlphttp")
		}
		if i.jaegerEndpoint != "" {
			traces.Exporters = append(traces.Exporters, "jaeger")
		}
		pipelines.Traces = traces
	}

	return Service{
		Extensions: extensions,
		Pipelines:  pipelines,
	}
}

func (i *Instance) createProcessors() Processors {
	processors := Processors{}

	if i.jaegerGrpcPort != 0 {
		processors.Batch = Batch{}
		processors.MemoryLimiter = MemoryLimiter{
			LimitMiB:      400,
			SpikeLimitMiB: 100,
			CheckInterval: "5s",
		}
	}

	return processors
}
