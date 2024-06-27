package minio

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/celestiaorg/knuu/pkg/k8s"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ServiceName      = "minio-service"
	ServiceAPIPort   = 9000 // API port
	ServiceWebUIPort = 9001 // WebUI port
	DeploymentName   = "minio"
	Image            = "minio/minio:RELEASE.2024-03-30T09-41-56Z"
	StorageClassName = "standard" // standard | gp2 | default
	VolumeClaimName  = "minio-data"
	VolumeMountPath  = "/data"

	// The minio service is used internally, so not sure if it is ok to use constant key/secret
	rootUser     = "minioUser"     // Previously accessKey
	rootPassword = "minioPassword" // Previously secretKey

	waitRetry            = 5 * time.Second
	pvPrefix             = "minio-pv-"
	pvHostPath           = "/tmp/minio-pv"
	deploymentAppLabel   = "app"
	deploymentMinioLabel = "minio"
)

var (
	PVCStorageSize = resource.MustParse("1Gi")
)

type Minio struct {
	client    *miniogo.Client
	k8sClient k8s.KubeManager
}

type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func New(ctx context.Context, k8sClient k8s.KubeManager) (*Minio, error) {
	m := &Minio{
		k8sClient: k8sClient,
	}

	if err := m.deployMinio(ctx); err != nil {
		return nil, err
	}

	endpoint, err := m.getEndpoint(ctx)
	if err != nil {
		return nil, ErrMinioFailedToGetEndpoint.Wrap(err)
	}

	m.client, err = miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(rootUser, rootPassword, ""),
		Secure: false,
	})
	if err != nil {
		return nil, ErrMinioFailedToInitializeClient.Wrap(err)
	}

	return m, nil
}

// Push pushes data (i.e. a reader) to Minio
func (m *Minio) Push(ctx context.Context, localReader io.Reader, minioFilePath, bucketName string) error {
	if m == nil {
		return ErrMinioNotInitialized
	}

	if err := m.createBucketIfNotExists(ctx, bucketName); err != nil {
		return ErrMinioFailedToCreateBucket.Wrap(err)
	}

	uploadInfo, err := m.client.PutObject(ctx, bucketName, minioFilePath, localReader, -1, miniogo.PutObjectOptions{})
	if err != nil {
		return ErrMinioFailedToUploadData.Wrap(err)
	}

	logrus.Debugf("Data uploaded successfully to %s in bucket %s", uploadInfo.Key, bucketName)
	return nil
}

// Delete deletes a file from Minio and fails if the content does not exist
func (m *Minio) Delete(ctx context.Context, minioFilePath, bucketName string) error {
	if m == nil {
		return ErrMinioNotInitialized
	}

	// Check if the object exists before attempting to delete
	_, err := m.client.StatObject(ctx, bucketName, minioFilePath, miniogo.StatObjectOptions{})
	if err != nil {
		return ErrMinioFailedToFindFileBeforeDeletion.Wrap(err)
	}

	err = m.client.RemoveObject(ctx, bucketName, minioFilePath, miniogo.RemoveObjectOptions{})
	if err != nil {
		return ErrMinioFailedToDeleteFile.Wrap(err)
	}

	logrus.Debugf("File %s deleted successfully from bucket %s", minioFilePath, bucketName)
	return nil
}

// GetURL returns an S3-compatible URL for a Minio file
func (m *Minio) GetURL(ctx context.Context, minioFilePath, bucketName string) (string, error) {
	if m == nil {
		return "", ErrMinioNotInitialized
	}

	// Set the expiration time for the URL (e.g., 24h from now)
	expiration := 24 * time.Hour

	// Generate a presigned URL for the object
	presignedURL, err := m.client.PresignedGetObject(ctx, bucketName, minioFilePath, expiration, nil)
	if err != nil {
		return "", ErrMinioFailedToGeneratePresignedURL.Wrap(err)
	}

	return presignedURL.String(), nil
}

func (m *Minio) GetConfigs(ctx context.Context) (*Config, error) {
	if m == nil {
		return nil, ErrMinioNotInitialized
	}

	endpoint, err := m.getEndpoint(ctx)
	if err != nil {
		return nil, ErrMinioFailedToGetEndpoint.Wrap(err)
	}

	return &Config{
		Endpoint:        endpoint,
		AccessKeyID:     rootUser,
		SecretAccessKey: rootPassword,
	}, nil
}

func (m *Minio) deployMinio(ctx context.Context) error {
	if err := m.createOrUpdateDeployment(ctx); err != nil {
		return ErrMinioFailedToStart.Wrap(err)
	}

	if err := m.waitForMinio(ctx); err != nil {
		return ErrMinioFailedToBeReady.Wrap(err)
	}

	if err := m.createOrUpdateService(ctx); err != nil {
		return ErrMinioFailedToCreateOrUpdateService.Wrap(err)
	}

	if err := m.k8sClient.WaitForService(ctx, ServiceName); err != nil {
		return ErrMinioFailedToBeReadyService.Wrap(err)
	}

	logrus.Debug("Minio deployed or updated successfully.")
	return nil
}

func (m *Minio) createOrUpdateDeployment(ctx context.Context) error {
	deploymentClient := m.k8sClient.Clientset().AppsV1().Deployments(m.k8sClient.Namespace())

	// Define the Minio deployment
	minioDeployment := &appsV1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DeploymentName,
			Namespace: m.k8sClient.Namespace(),
		},
		Spec: appsV1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{deploymentAppLabel: deploymentMinioLabel},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{deploymentAppLabel: deploymentMinioLabel}},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  DeploymentName,
						Image: Image,
						Env: []v1.EnvVar{
							{Name: "MINIO_ROOT_USER", Value: rootUser},
							{Name: "MINIO_ROOT_PASSWORD", Value: rootPassword},
						},
						Ports: []v1.ContainerPort{
							{ContainerPort: ServiceAPIPort},
							{ContainerPort: ServiceWebUIPort},
						},
						VolumeMounts: []v1.VolumeMount{{
							Name:      VolumeClaimName,
							MountPath: VolumeMountPath,
						}},
						Command: []string{
							"minio",
							"server",
							VolumeMountPath,
							"--console-address=:9001",
						},
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

	// Check if the deployment already exists
	_, err := deploymentClient.Get(ctx, DeploymentName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Deployment does not exist, create it
			err := m.createPVC(ctx,
				VolumeClaimName,
				PVCStorageSize,
				metav1.CreateOptions{},
			)
			if err != nil {
				return ErrMinioFailedToCreatePVC.Wrap(err)
			}
			_, err = deploymentClient.Create(ctx, minioDeployment, metav1.CreateOptions{})
			if err != nil {
				return ErrMinioFailedToCreateDeployment.Wrap(err)
			}
			logrus.Debug("Minio deployment created successfully.")
		} else {
			return ErrMinioFailedToGetDeployment.Wrap(err)
		}
	} else {
		// Deployment exists, update it
		_, err = deploymentClient.Update(ctx, minioDeployment, metav1.UpdateOptions{})
		if err != nil {
			return ErrMinioFailedToUpdateDeployment.Wrap(err)
		}
		logrus.Debug("Minio deployment updated successfully.")
	}

	return nil
}

func (m *Minio) createOrUpdateService(ctx context.Context) error {
	serviceClient := m.k8sClient.Clientset().CoreV1().Services(m.k8sClient.Namespace())

	// Define Minio service
	minioService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ServiceName,
			Namespace: m.k8sClient.Namespace(),
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"app": "minio"},
			Ports: []v1.ServicePort{
				{
					Name:       "api",
					Protocol:   v1.ProtocolTCP,
					Port:       ServiceAPIPort,
					TargetPort: intstr.FromInt(ServiceAPIPort),
				},
				{
					Name:       "webui",
					Protocol:   v1.ProtocolTCP,
					Port:       ServiceWebUIPort,
					TargetPort: intstr.FromInt(ServiceWebUIPort),
				},
			},
			Type: v1.ServiceTypeLoadBalancer,
		},
	}

	// Check if Minio service already exists
	existingService, err := serviceClient.Get(ctx, ServiceName, metav1.GetOptions{})
	if err == nil {
		logrus.Debugf("Service `%s` already exists, updating.", ServiceName)
		minioService.ResourceVersion = existingService.ResourceVersion // Retain the existing resource version
		if _, err := serviceClient.Update(ctx, minioService, metav1.UpdateOptions{}); err != nil {
			return ErrMinioFailedToUpdateService.Wrap(err)
		}
		logrus.Debugf("Service %s updated successfully.", ServiceName)
		return nil
	}

	// Create Minio service if it does not exist
	if _, err := serviceClient.Create(ctx, minioService, metav1.CreateOptions{}); err != nil {
		return ErrMinioFailedToCreateService.Wrap(err)
	}

	logrus.Debugf("Service %s created successfully.", ServiceName)
	return nil
}

func (m *Minio) createBucketIfNotExists(ctx context.Context, bucketName string) error {
	if m.client == nil {
		return ErrMinioClientNotInitialized
	}

	exists, err := m.client.BucketExists(ctx, bucketName)
	if err != nil {
		return ErrMinioFailedToCheckBucket.Wrap(err)
	}
	if exists {
		return nil
	}

	if err := m.client.MakeBucket(ctx, bucketName, miniogo.MakeBucketOptions{}); err != nil {
		return ErrMinioFailedToCreateBucket.Wrap(err)
	}
	logrus.Debugf("Bucket `%s` created successfully.", bucketName)

	return nil
}

func (m *Minio) getEndpoint(ctx context.Context) (string, error) {
	minioService, err := m.k8sClient.Clientset().CoreV1().Services(m.k8sClient.Namespace()).Get(ctx, ServiceName, metav1.GetOptions{})
	if err != nil {
		return "", ErrMinioFailedToGetService.Wrap(err)
	}

	if minioService.Spec.Type == v1.ServiceTypeLoadBalancer {
		// Use the LoadBalancer's external IP
		if len(minioService.Status.LoadBalancer.Ingress) > 0 {
			return fmt.Sprintf("%s:%d", minioService.Status.LoadBalancer.Ingress[0].IP, minioService.Spec.Ports[0].Port), nil
		}
		return "", ErrMinioLoadBalancerIPNotAvailable
	}

	if minioService.Spec.Type == v1.ServiceTypeNodePort {
		// Use the Node IP and NodePort
		nodes, err := m.k8sClient.Clientset().CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return "", ErrMinioFailedToGetNodes.Wrap(err)
		}
		if len(nodes.Items) == 0 {
			return "", ErrMinioNoNodesFound
		}

		// Use the first node for simplicity, you might need to handle multiple nodes
		var nodeIP string
		for _, address := range nodes.Items[0].Status.Addresses {
			if address.Type == "ExternalIP" {
				nodeIP = address.Address
				break
			}
		}
		return fmt.Sprintf("%s:%d", nodeIP, minioService.Spec.Ports[0].NodePort), nil
	}

	return fmt.Sprintf("%s:%d", minioService.Spec.ClusterIP, minioService.Spec.Ports[0].Port), nil
}

func (m *Minio) waitForMinio(ctx context.Context) error {
	for {
		deployment, err := m.k8sClient.Clientset().AppsV1().Deployments(m.k8sClient.Namespace()).Get(ctx, DeploymentName, metav1.GetOptions{})
		if err == nil && deployment.Status.ReadyReplicas > 0 {
			break
		}

		select {
		case <-ctx.Done():
			return ErrMinioTimeoutWaitingForReady
		case <-time.After(waitRetry):
			// Retry after some seconds
		}
	}

	return nil
}


func (m *Minio) createPVC(ctx context.Context, pvcName string, storageSize resource.Quantity, createOptions metav1.CreateOptions) error {
	pvcClient := m.K8s.Clientset().CoreV1().PersistentVolumeClaims(m.K8s.Namespace())

	// Check if PVC already exists
	_, err := pvcClient.Get(ctx, pvcName, metav1.GetOptions{})
	if err == nil {
		logrus.Debugf("PersistentVolumeClaim `%s` already exists.", pvcName)
		return nil
	}

	// Create a simple PersistentVolume if no suitable one is found
	pvList, err := m.k8sClient.Clientset().CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return ErrMinioFailedToListPersistentVolumes.Wrap(err)
	}

	var existingPV *v1.PersistentVolume
	for _, pv := range pvList.Items {
		// Not sure if this condition is ok
		if pv.Spec.Capacity[v1.ResourceStorage].Equal(storageSize) {
			existingPV = &pv
			break
		}
	}

	if existingPV == nil {
		// Create a simple PV if no existing PV is suitable
		_, err = m.k8sClient.Clientset().CoreV1().PersistentVolumes().Create(ctx, &v1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: pvPrefix,
			},
			Spec: v1.PersistentVolumeSpec{
				Capacity: v1.ResourceList{
					v1.ResourceStorage: storageSize,
				},
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				PersistentVolumeSource: v1.PersistentVolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: pvHostPath,
					},
				},
			},
		}, createOptions)
		if err != nil {
			return ErrMinioFailedToCreatePersistentVolume.Wrap(err)
		}
	}
	logrus.Debugf("PersistentVolume `%s` created successfully.", existingPV.Name)

	// Create PVC with the existing or newly created PV
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: m.k8sClient.Namespace(),
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: storageSize,
				},
			},
		},
	}

	_, err = pvcClient.Create(ctx, pvc, createOptions)
	if err != nil {
		return ErrMinioFailedToCreatePersistentVolumeClaim.Wrap(err)
	}

	logrus.Debugf("PersistentVolumeClaim `%s` created successfully.", pvcName)
	return nil
}
