package minio

import (
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/minio/minio-go/v7"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const (
	ServiceName         = "minio-service"
	ServicePort         = 9000
	ServiceExternalPort = 30505 // some random port
	DeploymentName      = "minio"
	Image               = "minio/minio:RELEASE.2024-01-28T22-35-53Z"
	StorageClassName    = "standard" // standard | gp2 | default
	VolumeClaimName     = "minio-data"
	VolumeMountPath     = "/data"
	PVCStorageSize      = "1Gi"

	// The minio service is used internally, so not sure if it is ok to use constant key/secret
	accessKey = "minioaccesskey"
	secretKey = "miniosecretkey"

	waitRetry = 5 * time.Second
)

type Minio struct {
	Clientset kubernetes.Interface
	Namespace string
}

func (m *Minio) DeployMinio(ctx context.Context) error {
	deploymentClient := m.Clientset.AppsV1().Deployments(m.Namespace)

	deployed, err := m.IsMinioDeployed(ctx)
	if err != nil {
		return fmt.Errorf("failed to check Minio deployment status: %v", err)
	}
	if deployed {
		return nil
	}

	if err := m.createPVC(ctx, VolumeClaimName, PVCStorageSize); err != nil {
		return fmt.Errorf("failed to create PVC: %v", err)
	}

	// Create Minio deployment
	minioDeployment := &appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName,
			Namespace: m.Namespace,
		},
		Spec: appsV1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "minio"},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "minio"}},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  DeploymentName,
						Image: Image,
						Env: []v1.EnvVar{
							{Name: "MINIO_ACCESS_KEY", Value: accessKey},
							{Name: "MINIO_SECRET_KEY", Value: secretKey},
						},
						Ports: []v1.ContainerPort{{ContainerPort: ServicePort}},
						VolumeMounts: []v1.VolumeMount{{
							Name:      VolumeClaimName,
							MountPath: VolumeMountPath,
						}},
						Command: []string{"minio", "server", VolumeMountPath},
					}},
					Volumes: []v1.Volume{{
						Name: VolumeClaimName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
								ClaimName: VolumeClaimName,
							},
						},
					}},
				},
			},
		},
	}

	_, err = deploymentClient.Create(ctx, minioDeployment, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Minio deployment: %v", err)
	}

	if err = m.waitForMinio(ctx); err != nil {
		return fmt.Errorf("failed waiting for Minio to be ready: %v", err)
	}

	if err := m.createService(ctx); err != nil {
		return fmt.Errorf("failed to create Minio service: %v", err)
	}

	if err := m.waitForMinioService(ctx); err != nil {
		return fmt.Errorf("failed waiting for Minio service to be ready: %v", err)
	}

	logrus.Info("Minio deployed successfully.")
	return nil
}

func (m *Minio) IsMinioDeployed(ctx context.Context) (bool, error) {
	deploymentClient := m.Clientset.AppsV1().Deployments(m.Namespace)

	_, err := deploymentClient.Get(ctx, DeploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get Minio deployment: %v", err)
	}

	return true, nil
}

// PushToMinio pushes data (i.e. a reader) to Minio
func (m *Minio) PushToMinio(ctx context.Context, localReader io.Reader, minioFilePath, bucketName string) error {
	endpoint, err := m.getEndpoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to get Minio endpoint: %v", err)
	}

	cli, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Minio client: %v", err)
	}

	if err := m.createBucketIfNotExists(ctx, cli, bucketName); err != nil {
		return fmt.Errorf("failed to create bucket: %v", err)
	}

	uploadInfo, err := cli.PutObject(ctx, bucketName, minioFilePath, localReader, -1, miniogo.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload data to Minio: %v", err)
	}

	logrus.Debugf("Data uploaded successfully to %s in bucket %s", uploadInfo.Key, bucketName)
	return nil
}

// GetMinioURL returns an S3-compatible URL for a Minio file
func (m *Minio) GetMinioURL(ctx context.Context, minioFilePath, bucketName string) (string, error) {
	minioEndpoint, err := m.getEndpoint(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get Minio endpoint: %v", err)
	}
	// Initialize Minio client
	minioClient, err := miniogo.New(minioEndpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to initialize Minio client: %v", err)
	}

	// Set the expiration time for the URL (e.g., 24h from now)
	expiration := time.Duration(24 * time.Hour)

	// Generate a presigned URL for the object
	presignedURL, err := minioClient.PresignedGetObject(ctx, bucketName, minioFilePath, expiration, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for Minio object: %v", err)
	}

	return presignedURL.String(), nil
}

func (m *Minio) createService(ctx context.Context) error {
	serviceClient := m.Clientset.CoreV1().Services(m.Namespace)

	// Check if Minio service already exists
	_, err := serviceClient.Get(ctx, ServiceName, metav1.GetOptions{})
	if err == nil {
		logrus.Debugf("Service `%s` already exists.", ServiceName)
		return nil
	}

	// Create Minio service
	minioService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: m.Namespace,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"app": "minio"},
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       ServicePort,
					TargetPort: intstr.FromInt(ServicePort),
					NodePort:   ServiceExternalPort,
				},
			},
			// Expose the service port outside the cluster, so client can push their data to Minio
			Type: v1.ServiceTypeNodePort,
		},
	}

	_, err = serviceClient.Create(ctx, minioService, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create Minio service: %v", err)
	}

	logrus.Debugf("Service %s created successfully.", ServiceName)
	return nil
}

func (m *Minio) createBucketIfNotExists(ctx context.Context, cli *minio.Client, bucketName string) error {
	exists, err := cli.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %v", err)
	}
	if exists {
		return nil
	}

	if err := cli.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("failed to create bucket: %v", err)
	}
	logrus.Debugf("Bucket `%s` created successfully.", bucketName)

	return nil
}

func (m *Minio) getEndpoint(ctx context.Context) (string, error) {
	minioService, err := m.Clientset.CoreV1().Services(m.Namespace).Get(ctx, ServiceName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Minio service: %v", err)
	}

	if minioService.Spec.Type == v1.ServiceTypeLoadBalancer {
		// Use the LoadBalancer's external IP
		if len(minioService.Status.LoadBalancer.Ingress) > 0 {
			return fmt.Sprintf("%s:%d", minioService.Status.LoadBalancer.Ingress[0].IP, minioService.Spec.Ports[0].Port), nil
		}
		return "", fmt.Errorf("LoadBalancer IP not available yet")
	}

	if minioService.Spec.Type == v1.ServiceTypeNodePort {
		// Use the Node IP and NodePort
		nodes, err := m.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get nodes: %v", err)
		}
		if len(nodes.Items) == 0 {
			return "", fmt.Errorf("no nodes found")
		}

		// Use the first node for simplicity, you might need to handle multiple nodes
		nodeIP := nodes.Items[0].Status.Addresses[0].Address
		return fmt.Sprintf("%s:%d", nodeIP, minioService.Spec.Ports[0].NodePort), nil
	}

	return fmt.Sprintf("%s:%d", minioService.Spec.ClusterIP, minioService.Spec.Ports[0].Port), nil
}

func (m *Minio) waitForMinio(ctx context.Context) error {
	for {
		deployment, err := m.Clientset.AppsV1().Deployments(m.Namespace).Get(ctx, DeploymentName, metav1.GetOptions{})
		if err == nil && deployment.Status.ReadyReplicas > 0 {
			break
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Minio to be ready")
		case <-time.After(waitRetry):
			// Retry after some seconds
		}
	}

	return nil
}

func (m *Minio) waitForMinioService(ctx context.Context) error {
	for {
		service, err := m.Clientset.CoreV1().Services(m.Namespace).Get(ctx, ServiceName, metav1.GetOptions{})
		if err == nil &&
			(service.Spec.Type == v1.ServiceTypeLoadBalancer ||
				service.Spec.Type == v1.ServiceTypeNodePort) &&
			// Check if LoadBalancer IP, NodePort, or externalIPs are available
			(len(service.Status.LoadBalancer.Ingress) > 0 ||
				service.Spec.Ports[0].NodePort > 0 ||
				len(service.Spec.ExternalIPs) > 0) {

			// Check if Minio is reachable
			endpoint, err := m.getEndpoint(ctx)
			if err != nil {
				return fmt.Errorf("failed to get Minio endpoint: %v", err)
			}

			if err := checkServiceConnectivity(endpoint); err == nil {
				break
			}
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Minio service to be ready")
		case <-time.After(waitRetry):
			// Retry after some seconds
		}
	}

	return nil
}

func checkServiceConnectivity(serviceEndpoint string) error {
	conn, err := net.DialTimeout("tcp", serviceEndpoint, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", serviceEndpoint, err)
	}
	defer conn.Close()
	return nil // success
}

func (m *Minio) createPVC(ctx context.Context, pvcName string, storageSize string) error {
	storageQt, err := resource.ParseQuantity(storageSize)
	if err != nil {
		return fmt.Errorf("failed to parse storage size: %v", err)
	}

	pvcClient := m.Clientset.CoreV1().PersistentVolumeClaims(m.Namespace)

	// Check if PVC already exists
	_, err = pvcClient.Get(ctx, pvcName, metav1.GetOptions{})
	if err == nil {
		logrus.Debugf("PersistentVolumeClaim `%s` already exists.", pvcName)
		return nil
	}

	// Create a simple PersistentVolume if no suitable one is found
	pvList, err := m.Clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list PersistentVolumes: %v", err)
	}

	var existingPV *v1.PersistentVolume
	for _, pv := range pvList.Items {
		// Not sure if this condition is ok
		if pv.Spec.Capacity[v1.ResourceStorage].Equal(storageQt) {
			existingPV = &pv
			break
		}
	}

	if existingPV == nil {
		// Create a simple PV if no existing PV is suitable
		_, err = m.Clientset.CoreV1().PersistentVolumes().Create(ctx, &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "minio-pv-",
			},
			Spec: v1.PersistentVolumeSpec{
				Capacity: v1.ResourceList{
					v1.ResourceStorage: storageQt,
				},
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				PersistentVolumeSource: v1.PersistentVolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/tmp/minio-pv", // Replace with your desired host path
					},
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("failed to create PersistentVolume: %v", err)
		}

		logrus.Debugf("PersistentVolume `%s` created successfully.", existingPV.Name)
	}

	// Create PVC with the existing or newly created PV
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: m.Namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: storageQt,
				},
			},
		},
	}

	_, err = pvcClient.Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create PersistentVolumeClaim: %v", err)
	}

	logrus.Debugf("PersistentVolumeClaim `%s` created successfully.", pvcName)
	return nil
}
