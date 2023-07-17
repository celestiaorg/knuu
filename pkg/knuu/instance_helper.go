package knuu

import (
	"fmt"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// getImageRegistry returns the name of the temporary image registry
func (i *Instance) getImageRegistry() (string, error) {
	if i.imageName != "" {
		return i.imageName, nil
	}
	// If not already set, generate a random name using ttl.sh
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("error generating UUID: %w", err)
	}
	imageName := fmt.Sprintf("ttl.sh/%s:1h", uuid.String())
	return imageName, nil
}

// validatePort validates the port
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number '%d' is out of range", port)
	}
	return nil
}

// isTCPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (i *Instance) isTCPPortRegistered(port int) bool {
	for _, p := range i.portsTCP {
		if p == port {
			return true
		}
	}
	return false
}

// isUDPPortRegistered returns true if the given port is registered
// with the instance, and false otherwise
func (i *Instance) isUDPPortRegistered(port int) bool {
	for _, p := range i.portsUDP {
		if p == port {
			return true
		}
	}
	return false
}

// getLabels returns the labels for the instance
func (i *Instance) getLabels() map[string]string {
	return map[string]string{
		"app":                          i.k8sName,
		"k8s.kubernetes.io/managed-by": "knuu",
		"knuu.sh/test-run-id":          identifier,
		"knuu.sh/test-started":         startTime,
		"knuu.sh/name":                 i.name,
		"knuu.sh/k8s-name":             i.k8sName,
		"knuu.sh/type":                 i.instanceType.String(),
	}
}

// deployService deploys the service for the instance
func (i *Instance) deployService() error {
	svc, _ := k8s.GetService(k8s.Namespace(), i.k8sName)
	if svc != nil {
		// Service already exists, so we patch it
		err := i.patchService()
		if err != nil {
			return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
		}
	}

	labels := i.getLabels()
	var annotations map[string]string
	selectorMap := i.getLabels()
	serviceType := v1.ServiceTypeClusterIP

	if len(i.externalDns) > 0 {
		annotations = map[string]string{
			"external-dns.alpha.kubernetes.io/hostname": strings.Join(i.externalDns, ","),
		}
		serviceType = v1.ServiceTypeLoadBalancer
	}

	service, err := k8s.DeployService(k8s.Namespace(), i.k8sName, labels, annotations, selectorMap, i.portsTCP, i.portsUDP, serviceType)
	if err != nil {
		return fmt.Errorf("error deploying service '%s': %w", i.k8sName, err)
	}
	i.kubernetesService = service
	logrus.Debugf("Started service '%s'", i.k8sName)
	return nil
}

// patchService patches the service for the instance
func (i *Instance) patchService() error {
	if i.kubernetesService == nil {
		svc, err := k8s.GetService(k8s.Namespace(), i.k8sName)
		if err != nil {
			return fmt.Errorf("error getting service '%s': %w", i.k8sName, err)
		}
		i.kubernetesService = svc
	}
	err := k8s.PatchService(k8s.Namespace(), i.k8sName, i.kubernetesService.ObjectMeta.Labels, i.kubernetesService.ObjectMeta.Annotations, i.kubernetesService.Spec.Selector, i.portsTCP, i.portsUDP, i.kubernetesService.Spec.Type)
	if err != nil {
		return fmt.Errorf("error patching service '%s': %w", i.k8sName, err)
	}
	logrus.Debugf("Patched service '%s'", i.k8sName)
	return nil
}

// destroyService destroys the service for the instance
func (i *Instance) destroyService() error {
	k8s.DeleteService(k8s.Namespace(), i.k8sName)

	return nil
}

// deployPod deploys the pod for the instance
func (i *Instance) deployPod() error {
	// Get labels for the pod
	labels := i.getLabels()

	imageName, err := i.getImageRegistry()
	if err != nil {
		return fmt.Errorf("failed to get image name: %v", err)
	}

	// create a service account for the pod
	if err := k8s.CreateServiceAccount(k8s.Namespace(), i.k8sName, labels); err != nil {
		return fmt.Errorf("failed to create service account: %v", err)
	}

	// create a role and role binding for the pod if there are policy rules
	if len(i.policyRules) > 0 {
		if err := k8s.CreateRole(k8s.Namespace(), i.k8sName, labels, i.policyRules); err != nil {
			return fmt.Errorf("failed to create role: %v", err)
		}
		if err := k8s.CreateRoleBinding(k8s.Namespace(), i.k8sName, labels, i.k8sName, i.k8sName); err != nil {
			return fmt.Errorf("failed to create role binding: %v", err)
		}
	}

	// Generate the container configuration
	containerConfig := k8s.ContainerConfig{
		Name:           i.k8sName,
		Image:          imageName,
		Command:        i.command,
		Args:           i.args,
		Env:            i.env,
		Volumes:        i.volumes,
		MemoryRequest:  i.memoryRequest,
		MemoryLimit:    i.memoryLimit,
		CPURequest:     i.cpuRequest,
		LivenessProbe:  i.livenessProbe,
		ReadinessProbe: i.readinessProbe,
		StartupProbe:   i.startupProbe,
		Files:          i.files,
	}
	// Generate the sidecar configurations
	sidecarConfigs := make([]k8s.ContainerConfig, 0)
	for _, sidecar := range i.sidecars {
		sidecarConfigs = append(sidecarConfigs, k8s.ContainerConfig{
			Name:           sidecar.k8sName,
			Image:          sidecar.imageName,
			Command:        sidecar.command,
			Args:           sidecar.args,
			Env:            sidecar.env,
			Volumes:        sidecar.volumes,
			MemoryRequest:  sidecar.memoryRequest,
			MemoryLimit:    sidecar.memoryLimit,
			CPURequest:     sidecar.cpuRequest,
			LivenessProbe:  sidecar.livenessProbe,
			ReadinessProbe: sidecar.readinessProbe,
			StartupProbe:   sidecar.startupProbe,
			Files:          sidecar.files,
		})
	}
	// Generate the pod configuration
	podConfig := k8s.PodConfig{
		Namespace:          k8s.Namespace(),
		Name:               i.k8sName,
		Labels:             labels,
		ServiceAccountName: i.k8sName,
		ContainerConfig:    containerConfig,
		SidecarConfigs:     sidecarConfigs,
	}
	// Generate the statefulset configuration
	statefulSetConfig := k8s.StatefulSetConfig{
		Namespace: k8s.Namespace(),
		Name:      i.k8sName,
		Labels:    labels,
		Replicas:  1,
		PodConfig: podConfig,
	}

	// Deploy the statefulSet
	statefulSet, err := k8s.DeployStatefulSet(statefulSetConfig, true)
	if err != nil {
		return fmt.Errorf("failed to deploy pod: %v", err)
	}

	// Set the state of the instance to started
	i.kubernetesStatefulSet = statefulSet

	// Log the deployment of the pod
	logrus.Debugf("Started statefulSet '%s'", i.k8sName)
	logrus.Debugf("Set state of instance '%s' to '%s'", i.k8sName, i.state.String())

	return nil
}

// destroyPod destroys the pod for the instance (no grace period)
// Skips if the pod is already destroyed
func (i *Instance) destroyPod() error {
	grace := int64(0)
	err := k8s.DeleteStatefulSetWithGracePeriod(k8s.Namespace(), i.k8sName, &grace)
	if err != nil {
		return fmt.Errorf("failed to delete pod: %v", err)
	}

	// Delete the service account for the pod
	if err := k8s.DeleteServiceAccount(k8s.Namespace(), i.k8sName); err != nil {
		return fmt.Errorf("failed to delete service account: %v", err)
	}
	// Delete the role and role binding for the pod if there are policy rules
	if len(i.policyRules) > 0 {
		if err := k8s.DeleteRole(k8s.Namespace(), i.k8sName); err != nil {
			return fmt.Errorf("failed to delete role: %v", err)
		}
		if err := k8s.DeleteRoleBinding(k8s.Namespace(), i.k8sName); err != nil {
			return fmt.Errorf("failed to delete role binding: %v", err)
		}
	}

	return nil
}

// deployVolume deploys the volume for the instance
func (i *Instance) deployVolume() error {
	size := resource.Quantity{}
	for _, volume := range i.volumes {
		size.Add(resource.MustParse(volume.Size))
	}
	k8s.DeployPersistentVolumeClaim(k8s.Namespace(), i.k8sName, i.getLabels(), size)
	logrus.Debugf("Deployed persistent volume '%s'", i.k8sName)

	return nil
}

// destroyVolume destroys the volume for the instance
func (i *Instance) destroyVolume() error {
	k8s.DeletePersistentVolumeClaim(k8s.Namespace(), i.k8sName)
	logrus.Debugf("Destroyed persistent volume '%s'", i.k8sName)

	return nil
}

// deployFiles deploys the files for the instance
func (i *Instance) deployFiles() error {

	data := map[string]string{}

	n := 0

	for _, file := range i.files {
		// read out file content and assign to variable
		srcFile, err := os.Open(file.Source)
		if err != nil {
			return fmt.Errorf("failed to open file: %v", err)
		}
		fileContentBytes, err := io.ReadAll(srcFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %v", err)
		}
		srcFile.Close()
		fileContent := string(fileContentBytes)

		keyName := fmt.Sprintf("%d", n)

		data[keyName] = fileContent

		n++
	}

	// create configmap
	if _, err := k8s.CreateConfigMap(k8s.Namespace(), i.k8sName, i.getLabels(), data); err != nil {
		return fmt.Errorf("failed to create configmap: %v", err)
	}

	return nil
}

// destroyFiles destroys the files for the instance
func (i *Instance) destroyFiles() error {
	if err := k8s.DeleteConfigMap(k8s.Namespace(), i.k8sName); err != nil {
		return fmt.Errorf("failed to delete configmap: %v", err)
	}
	return nil
}

// deployResources deploys the resources for the instance
func (i *Instance) deployResources() error {
	if len(i.portsTCP) != 0 || len(i.portsUDP) != 0 {
		logrus.Debugf("Ports not empty, deploying service for instance '%s'", i.k8sName)
		svc, _ := k8s.GetService(k8s.Namespace(), i.k8sName)
		if svc == nil {
			err := i.deployService()
			if err != nil {
				return fmt.Errorf("error deploying service for instance '%s': %w", i.k8sName, err)
			}
		} else if svc != nil {
			err := i.patchService()
			if err != nil {
				return fmt.Errorf("error patching service for instance '%s': %w", i.k8sName, err)
			}
		}
	}
	if len(i.volumes) != 0 {
		err := i.deployVolume()
		if err != nil {
			return fmt.Errorf("error deploying volume for instance '%s': %w", i.k8sName, err)
		}
	}
	if i.ingress != nil {
		err := i.deployIngress()
		if err != nil {
			return fmt.Errorf("error deploying ingress for instance '%s': %w", i.k8sName, err)
		}
	}
	if len(i.files) != 0 {
		err := i.deployFiles()
		if err != nil {
			return fmt.Errorf("error deploying files for instance '%s': %w", i.k8sName, err)
		}
	}

	return nil
}

// destroyResources destroys the resources for the instance
func (i *Instance) destroyResources() error {
	if len(i.volumes) != 0 {
		err := i.destroyVolume()
		if err != nil {
			return fmt.Errorf("error destroying volume for instance '%s': %w", i.k8sName, err)
		}
	}
	if len(i.files) != 0 {
		err := i.destroyFiles()
		if err != nil {
			return fmt.Errorf("error destroying files for instance '%s': %w", i.k8sName, err)
		}
	}
	err := i.destroyService()
	if err != nil {
		return fmt.Errorf("error destroying service for instance '%s': %w", i.k8sName, err)
	}
	if i.ingress != nil {
		err = i.destroyIngress()
		if err != nil {
			return fmt.Errorf("error destroying ingress for instance '%s': %w", i.k8sName, err)
		}
	}

	return nil
}

// cloneWithSuffix clones the instance with a suffix
func (i *Instance) cloneWithSuffix(suffix string) *Instance {
	return &Instance{
		name:                     i.name + suffix,
		k8sName:                  i.k8sName + suffix,
		imageName:                i.imageName,
		state:                    i.state,
		instanceType:             i.instanceType,
		kubernetesService:        i.kubernetesService,
		builderFactory:           i.builderFactory,
		kubernetesStatefulSet:    i.kubernetesStatefulSet,
		portsTCP:                 i.portsTCP,
		portsUDP:                 i.portsUDP,
		command:                  i.command,
		args:                     i.args,
		env:                      i.env,
		volumes:                  i.volumes,
		memoryRequest:            i.memoryRequest,
		memoryLimit:              i.memoryLimit,
		cpuRequest:               i.cpuRequest,
		policyRules:              i.policyRules,
		livenessProbe:            i.livenessProbe,
		readinessProbe:           i.readinessProbe,
		startupProbe:             i.startupProbe,
		files:                    i.files,
		ingress:                  i.ingress,
		externalDns:              i.externalDns,
		isSidecar:                false,
		sidecars:                 i.sidecars,
		otlpPort:                 i.otlpPort,
		prometheusPort:           i.prometheusPort,
		prometheusJobName:        i.prometheusJobName,
		prometheusScrapeInterval: i.prometheusScrapeInterval,
		jaegerGrpcPort:           i.jaegerGrpcPort,
		jaegerThriftCompactPort:  i.jaegerThriftCompactPort,
		jaegerThriftHttpPort:     i.jaegerThriftHttpPort,
		otlpEndpoint:             i.otlpEndpoint,
		otlpUsername:             i.otlpUsername,
		otlpPassword:             i.otlpPassword,
		jaegerEndpoint:           i.jaegerEndpoint,
	}
}

func generateK8sName(name string) (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("error generating UUID: %w", err)
	}
	return fmt.Sprintf("%s-%s", name, uuid.String()[:8]), nil
}

// getFreePort returns a free port
func getFreePortTCP() (int, error) {
	// Get a random port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("error getting free port: %w", err)
	}
	defer listener.Close()

	// Get the port from the listener
	port := listener.Addr().(*net.TCPAddr).Port

	return port, nil
}

// getBuildDir returns the build directory for the instance
func (i *Instance) getBuildDir() string {
	return filepath.Join("/tmp", "knuu", i.k8sName)
}

// validateFileArgs validates the file arguments
func (i *Instance) validateFileArgs(src string, dest string, chown string) error {
	// check src
	if src == "" {
		return fmt.Errorf("src must be set")
	}
	// check dest
	if dest == "" {
		return fmt.Errorf("dest must be set")
	}
	// check chown
	if chown == "" {
		return fmt.Errorf("chown must be set")
	}
	// validate chown format
	if !strings.Contains(chown, ":") || len(strings.Split(chown, ":")) != 2 {
		return fmt.Errorf("chown must be in format 'user:group'")
	}

	return nil
}

// addFileToBuilder adds a file to the builder
func (i *Instance) addFileToBuilder(src string, dest string, chown string) error {
	// dest is the same as src here, as we copy the file to the build dir with the subfolder structure of dest
	err := i.builderFactory.AddToBuilder(dest, dest, chown)
	if err != nil {
		return fmt.Errorf("error adding file '%s' to instance '%s': %w", dest, i.name, err)
	}
	return nil
}

// deployIngress deploys the ingress for the instance
func (i *Instance) deployIngress() error {
	ingress := i.ingress

	annotations := map[string]string{}

	if ingress.BackendProtocol != "" {
		annotations["nginx.ingress.kubernetes.io/backend-protocol"] = ingress.BackendProtocol
	}

	if ingress.CertManagerEnabled {
		annotations["cert-manager.io/cluster-issuer"] = clusterIssuer
	}

	if ingress.SslPassthrough {
		annotations["nginx.ingress.kubernetes.io/ssl-passthrough"] = "true"
	}

	if ingress.ForceSslRedirect {
		annotations["nginx.ingress.kubernetes.io/force-ssl-redirect"] = "true"
	}

	if ingress.EnableCors {
		annotations["nginx.ingress.kubernetes.io/enable-cors"] = "true"
	}

	if err := k8s.CreateIngress(k8s.Namespace(), i.k8sName, i.getLabels(), annotations, ingressClass, ingress.Host, ingress.Path, ingress.PathType, i.k8sName, ingress.Port, ingress.TlsEnabled); err != nil {
		return fmt.Errorf("failed to create ingress: %v", err)
	}

	return nil
}

// destroyIngress destroys the ingress for the instance
func (i *Instance) destroyIngress() error {
	k8s.DeleteIngress(k8s.Namespace(), i.k8sName)
	logrus.Debugf("Destroyed ingress '%s'", i.k8sName)

	return nil
}

// isObservabilityEnabled returns true if observability is enabled
func (i *Instance) isObservabilityEnabled() bool {
	return i.otlpPort != 0 || i.prometheusPort != 0 || i.jaegerGrpcPort != 0 || i.jaegerThriftCompactPort != 0 || i.jaegerThriftHttpPort != 0
}

// createOtelCollectorInstance deploys the OpenTelemetry collector
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

	// Specify extensions

	extensions := ""

	if i.otlpEndpoint != "" {
		extensions = fmt.Sprintf(`
extensions:
  basicauth/otlp:
    client_auth:
      username: %s
      password: "%s"
`, i.otlpUsername, i.otlpPassword)
	}

	// Specify receivers

	receivers := `
receivers:
`

	if i.otlpPort != 0 {
		receivers += fmt.Sprintf(`
    otlp:
      protocols:
        http:
          endpoint: "localhost:%d"
`, i.otlpPort)
	}

	if i.prometheusPort != 0 {
		receivers += fmt.Sprintf(`
    prometheus:
      config:
        scrape_configs:
        - job_name: '%s'
          scrape_interval: %s
          static_configs:
          - targets: ['localhost:%d']
`, i.prometheusJobName, i.prometheusScrapeInterval, i.prometheusPort)
	}

	if i.jaegerGrpcPort != 0 {
		// if grpc is set the others are also set
		receivers += fmt.Sprintf(`
    jaeger:
      protocols:
        grpc:
          endpoint: localhost:%d
        thrift_compact:
          endpoint: localhost:%d
        thrift_http:
          endpoint: localhost:%d
`, i.jaegerGrpcPort, i.jaegerThriftCompactPort, i.jaegerThriftHttpPort)
	}

	// Specify exporters

	exporters := ""

	if i.otlpEndpoint != "" {
		exporters += `
exporters:
`
		if i.otlpEndpoint != "" {
			exporters += fmt.Sprintf(`
    otlphttp:
      auth:
        authenticator: basicauth/otlp
      endpoint: %s
`, i.otlpEndpoint)
		}

		if i.jaegerEndpoint != "" {
			exporters += fmt.Sprintf(`
    jaeger:
      endpoint: %s
      tls:
        insecure: true
`, i.jaegerEndpoint)
		}
	}

	// Specify service

	service := `
service:
`

	if i.otlpEndpoint != "" {
		service += `
    extensions: [basicauth/otlp]
`
	}

	if i.otlpPort != 0 || i.prometheusPort != 0 || i.otlpEndpoint != "" || i.jaegerGrpcPort != 0 {
		service += `
    pipelines:
`

		if i.otlpPort != 0 || i.prometheusPort != 0 || i.otlpEndpoint != "" {
			service += `
      metrics:
`
			if i.otlpPort != 0 || i.prometheusPort != 0 {
				service += `
        receivers:
 `
			}
			if i.otlpPort != 0 {
				service += `
        - otlp
`
			}
			if i.prometheusPort != 0 {
				service += `
        - prometheus
`
			}

			if i.otlpEndpoint != "" {
				service += `
        exporters:
        - otlphttp
`
			}
		}

		if i.jaegerGrpcPort != 0 || i.jaegerThriftCompactPort != 0 || i.jaegerThriftHttpPort != 0 || i.jaegerEndpoint != "" {
			service += `
      traces:
`
			if i.jaegerGrpcPort != 0 || i.jaegerThriftCompactPort != 0 || i.jaegerThriftHttpPort != 0 {
				service += `
        receivers:
        - jaeger
`
			}
			if i.jaegerEndpoint != "" {
				service += `
        exporters:
        - jaeger
        processors:
        - batch
        - memory_limiter
`
			}
		}
	}

	// Specify processors

	processors := `
processors:
`

	if i.jaegerGrpcPort != 0 {
		processors += `
    batch:
    memory_limiter:
      # 80 percent of maximum memory up to 2G
      limit_mib: 400
      # 25 percent of limit up to 2G
      spike_limit_mib: 100
      check_interval: 5s
`
	}

	config := fmt.Sprintf(`
%s
%s
%s
%s
%s
`, extensions, receivers, exporters, processors, service)

	bytes := []byte(config)
	if err := otelAgent.AddFileBytes(bytes, "/etc/otel-agent.yaml", "root:root"); err != nil {
		return nil, fmt.Errorf("error adding file to otel-agent instance: %w", err)
	}
	if err := otelAgent.SetCommand("/otelcol-contrib", "--config=/etc/otel-agent.yaml"); err != nil {
		return nil, fmt.Errorf("error setting command for otel-agent instance: %w", err)
	}

	return otelAgent, nil
}
