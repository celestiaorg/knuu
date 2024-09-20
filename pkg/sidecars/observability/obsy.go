package observability

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	DefaultOtelOtlpPort         = 8888
	DefaultOtelMetricsPort      = 9090
	DefaultImage                = "otel/opentelemetry-collector-contrib:%s"
	DefaultOtelCollectorVersion = "0.83.0"

	otelAgentName = "otel-agent"
	// %s will be replaced with the otelCollectorVersion
	otelAgentConfigFile            = "/etc/otel-agent.yaml"
	otelAgentConfigFilePermissions = "0:0"

	otelCollectorCommand   = "/otelcol-contrib"
	otelCollectorConfigArg = "--config=/etc/otel-agent.yaml"
)

type Obsy struct {
	instance   *instance.Instance
	obsyConfig *ObsyConfig
	image      string
}

var _ instance.SidecarManager = (*Obsy)(nil)

var (
	otelAgentCPU         = resource.MustParse("100m")
	otelAgentMemory      = resource.MustParse("100Mi")
	otelAgentMemoryLimit = resource.MustParse("200Mi")
)

// ObsyConfig represents the configuration for the obsy sidecar
type ObsyConfig struct {
	// otelCollectorVersion is the version of the otel collector to use
	otelCollectorVersion string

	// prometheusEndpointPort is the port on which the prometheus server will be exposed
	prometheusEndpointPort int
	// prometheusEndpointJobName is the name of the prometheus job
	prometheusEndpointJobName string
	// prometheusEndpointScrapeInterval is the scrape interval for the prometheus job
	prometheusEndpointScrapeInterval string

	// jaegerGrpcPort is the port on which the jaeger grpc server is exposed
	jaegerGrpcPort int
	// jaegerThriftCompactPort is the port on which the jaeger thrift compact server is exposed
	jaegerThriftCompactPort int
	// jaegerThriftHttpPort is the port on which the jaeger thrift http server is exposed
	jaegerThriftHttpPort int
	// jaegerEndpoint is the endpoint of the jaeger collector where spans will be sent to
	jaegerEndpoint string

	// otlpPort is the port on which the otlp server is exposed
	otlpPort int
	// otlpEndpoint is the endpoint of the otlp collector where spans will be sent to
	otlpEndpoint string
	// otlpUsername is the username to use for the otlp collector
	otlpUsername string
	// otlpPassword is the password to use for the otlp collector
	otlpPassword string

	// prometheusExporterEndpoint is the endpoint of the prometheus exporter
	prometheusExporterEndpoint string

	// prometheusRemoteWriteExporterEndpoint is the endpoint of the prometheus remote write
	prometheusRemoteWriteExporterEndpoint string
}

func New() *Obsy {
	return &Obsy{
		obsyConfig: &ObsyConfig{
			otelCollectorVersion: DefaultOtelCollectorVersion,
		},
		image: fmt.Sprintf(DefaultImage, DefaultOtelCollectorVersion),
	}
}

func (o *Obsy) Initialize(ctx context.Context, namePrefix string, sysDeps *system.SystemDependencies) error {
	var err error
	o.instance, err = instance.New(namePrefix+"-"+otelAgentName, sysDeps)
	if err != nil {
		return ErrCreatingOtelAgentInstance.Wrap(err)
	}
	o.instance.Sidecars().SetIsSidecar(true)

	err = o.instance.Build().SetImage(ctx, o.image)
	if err != nil {
		return ErrSettingOtelAgentImage.Wrap(err)
	}
	if err := o.instance.Network().AddPortTCP(DefaultOtelOtlpPort); err != nil {
		return ErrAddingOtelAgentPort.Wrap(err)
	}
	if err := o.instance.Network().AddPortTCP(DefaultOtelMetricsPort); err != nil {
		return ErrAddingOtelAgentPort.Wrap(err)
	}
	if err := o.instance.Resources().SetCPU(otelAgentCPU); err != nil {
		return ErrSettingOtelAgentCPU.Wrap(err)
	}
	if err := o.instance.Resources().SetMemory(otelAgentMemory, otelAgentMemoryLimit); err != nil {
		return ErrSettingOtelAgentMemory.Wrap(err)
	}
	if err := o.instance.Build().Commit(ctx); err != nil {
		return ErrCommittingOtelAgentInstance.Wrap(err)
	}

	config := OTelConfig{
		Extensions: o.createExtensions(),
		Receivers:  o.createReceivers(),
		Exporters:  o.createExporters(),
		Service:    o.createService(),
		Processors: o.createProcessors(sysDeps.Scope),
	}

	bytes, err := yaml.Marshal(config)
	if err != nil {
		return ErrMarshalingYAML.Wrap(err)
	}

	if err := o.instance.Storage().AddFileBytes(bytes, otelAgentConfigFile, otelAgentConfigFilePermissions); err != nil {
		return ErrAddingOtelAgentConfigFile.Wrap(err)
	}

	if err := o.instance.Build().SetStartCommand(otelCollectorCommand, otelCollectorConfigArg); err != nil {
		return ErrSettingOtelAgentCommand.Wrap(err)
	}

	return nil
}

func (o *Obsy) PreStart(ctx context.Context) error {
	if o.instance == nil {
		return ErrObsyInstanceNotInitialized
	}
	return nil
}

func (o *Obsy) Instance() *instance.Instance {
	return o.instance
}

func (o *Obsy) Clone(namePrefix string) (instance.SidecarManager, error) {
	clone, err := o.instance.CloneWithName(namePrefix + "-" + otelAgentName)
	if err != nil {
		return nil, err
	}
	conf := *o.obsyConfig
	return &Obsy{
		instance:   clone,
		obsyConfig: &conf,
	}, nil
}
