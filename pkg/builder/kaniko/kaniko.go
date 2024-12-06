package kaniko

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/names"
	"github.com/celestiaorg/knuu/pkg/system"
)

const (
	kanikoImage         = "gcr.io/kaniko-project/executor:latest"
	kanikoContainerName = "kaniko-container"
	kanikoJobNamePrefix = "kaniko-build-job"

	kanikoRegistryCertsVolName   = "registry-certs"
	kanikoRegistryCertsMountPath = "/kaniko/certs/"

	DefaultParallelism  = int32(1)
	DefaultBackoffLimit = int32(5)

	MinioBucketName  = "kaniko"
	EphemeralStorage = "10Gi"
)

type Kaniko struct {
	*system.SystemDependencies
	Registry *RegistryOptions
}

// RegistryOptions contains the options for the registry
type RegistryOptions struct {
	Address    string
	Cert       []byte
	SecretName string
}

var _ builder.Builder = &Kaniko{}

func (k *Kaniko) Build(ctx context.Context, b builder.BuilderOptions) (logs string, err error) {
	if b.ImageName == "" {
		image, err := k.ResolveImageName(b.BuildContext)
		if err != nil {
			return "", err
		}
		b.ImageName = image.ToString()
	}

	job, err := k.prepareJob(ctx, &b)
	if err != nil {
		return "", ErrPreparingJob.Wrap(err)
	}

	cJob, err := k.K8sClient.Clientset().BatchV1().Jobs(k.K8sClient.Namespace()).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", ErrCreatingJob.Wrap(err)
	}

	kJob, err := k.waitForJobCompletion(ctx, cJob)
	if err != nil {
		return "", ErrWaitingJobCompletion.Wrap(err)
	}

	pod, err := k.firstPodFromJob(ctx, kJob)
	if err != nil {
		return "", ErrGettingPodFromJob.Wrap(err)
	}

	logs, err = k.containerLogs(ctx, pod)
	if err != nil {
		return "", ErrGettingContainerLogs.Wrap(err)
	}

	if kJob.Status.Succeeded == 0 {
		return logs, ErrBuildFailed
	}

	return logs, nil
}

func (k *Kaniko) CacheOptions() *builder.CacheOptions {
	if k.Registry == nil {
		return builder.DefaultCacheOptions()
	}

	return &builder.CacheOptions{
		Enabled: true,
		Repo:    builder.ImageWithRegistry(builder.DefaultCacheRepoName, k.Registry.Address),
	}
}

func (k *Kaniko) ResolveImageName(buildContext string) (*builder.ResolvedImage, error) {
	imageName, err := builder.DefaultImageName(buildContext)
	if err != nil {
		return nil, err
	}

	var (
		registry = builder.DefaultRegistryAddress
		tag      = builder.DefaultImageTTL
	)

	if k.Registry != nil {
		registry = k.Registry.Address
		tag = builder.DefaultImageTag // Use default tag instead of TTL if custom registry
	}

	return &builder.ResolvedImage{
		Name:     imageName,
		Registry: registry,
		Tag:      tag,
	}, nil
}

func (k *Kaniko) waitForJobCompletion(ctx context.Context, job *batchv1.Job) (*batchv1.Job, error) {
	watcher, err := k.K8sClient.Clientset().BatchV1().Jobs(k.K8sClient.Namespace()).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", job.Name),
	})
	if err != nil {
		return nil, ErrWatchingJob.Wrap(err)
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil, ErrWatchingChannelCloseUnexpectedly
			}

			j, ok := event.Object.(*batchv1.Job)
			if !ok {
				continue
			}

			if j.Status.Succeeded > 0 || j.Status.Failed > 0 {
				// Job completed (successfully or failed)
				return j, nil
			}
		case <-ctx.Done():
			return nil, ErrContextCancelled
		}
	}
}

func (k *Kaniko) firstPodFromJob(ctx context.Context, job *batchv1.Job) (*v1.Pod, error) {
	podList, err := k.K8sClient.Clientset().CoreV1().Pods(k.K8sClient.Namespace()).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	if err != nil {
		return nil, ErrListingPods.Wrap(err)
	}

	if len(podList.Items) == 0 {
		return nil, ErrNoPodsFound.Wrap(fmt.Errorf("job: %s", job.Name))
	}

	return &podList.Items[0], nil
}

func (k *Kaniko) containerLogs(ctx context.Context, pod *v1.Pod) (string, error) {
	if len(pod.Spec.Containers) == 0 {
		return "", ErrNoContainersFound.Wrap(fmt.Errorf("pod: %s", pod.Name))
	}

	logOptions := v1.PodLogOptions{
		Container: pod.Spec.Containers[0].Name,
	}

	req := k.K8sClient.Clientset().CoreV1().Pods(k.K8sClient.Namespace()).GetLogs(pod.Name, &logOptions)
	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

func (k *Kaniko) prepareJob(ctx context.Context, b *builder.BuilderOptions) (*batchv1.Job, error) {
	jobName, err := names.NewRandomK8(kanikoJobNamePrefix)
	if err != nil {
		return nil, ErrGeneratingUUID.Wrap(err)
	}

	ephemeralStorage, err := resource.ParseQuantity(EphemeralStorage)
	if err != nil {
		return nil, ErrParsingQuantity.Wrap(err)
	}

	var (
		parallelism  = DefaultParallelism
		backoffLimit = DefaultBackoffLimit
	)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &parallelism,  // Set parallelism to 1 to ensure only one Pod
			BackoffLimit: &backoffLimit, // Retry the Job at most 5 times
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  kanikoContainerName,
							Image: kanikoImage, // debug has a shell
							Args:  k.prepareArgs(b),
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceEphemeralStorage: ephemeralStorage,
								},
							},
							// VolumeMounts: []v1.VolumeMount{
							// 	{
							// 		Name:      kanikoRegistryCertsVolName,
							// 		MountPath: kanikoRegistryCertsMountPath,
							// 	},
							// },
						},
					},
					// Volumes: []v1.Volume{
					// 	{
					// 		Name: kanikoRegistryCertsVolName,
					// 		VolumeSource: v1.VolumeSource{
					// 			Secret: &v1.SecretVolumeSource{
					// 				SecretName: k.Registry.SecretName,
					// 			},
					// 		},
					// 	},
					// },
					RestartPolicy: "Never", // Ensure that the Pod does not restart
				},
			},
		},
	}

	if builder.IsDirContext(b.BuildContext) {
		job, err = k.mountDir(ctx, b.BuildContext, job)
		if err != nil {
			return nil, ErrMountingDir.Wrap(err)
		}
	}

	return job, nil
}

// mountDir mounts the build context directory to the Kaniko container
// Since we cannot really mount a local directory to a k8s Pod,
// we create a tar.gz archive of the directory and upload it to Minio
// then we download it from the init container into a shared volume which is also mounted
// to the Kaniko container
// As kaniko also supports directly tar.gz archives, no need to extract it,
// we just need to set the context to tar://<path-to-archive>
func (k *Kaniko) mountDir(ctx context.Context, bCtx string, job *batchv1.Job) (*batchv1.Job, error) {
	// Create the tar.gz archive
	archiveData, err := createTarGz(builder.GetDirFromBuildContext(bCtx))
	if err != nil {
		return nil, err
	}

	// Create a SHA256 hash of for the name of the archive content
	hash := sha256.New()
	hash.Write(archiveData)
	contentName := hex.EncodeToString(hash.Sum(nil))

	if err := k.MinioClient.Push(ctx, bytes.NewReader(archiveData), contentName, MinioBucketName); err != nil {
		return nil, err
	}

	s3URL, err := k.MinioClient.GetURL(ctx, contentName, MinioBucketName)
	if err != nil {
		return nil, err
	}

	const (
		workspaceDir     = "/workspace"
		workspaceVolName = "workspace"
		archiveFilePath  = workspaceDir + "/archive.tar.gz"
	)

	// Configure the init container to download the tar.gz archive first
	initContainer := v1.Container{
		Name:    "download-container",
		Image:   "curlimages/curl:latest",
		Command: []string{"/bin/sh", "-c"},
		Args: []string{
			fmt.Sprintf("curl -L -o %s '%s'", archiveFilePath, s3URL),
		},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      workspaceVolName,
				MountPath: workspaceDir,
			},
		},
	}
	job.Spec.Template.Spec.InitContainers = append(job.Spec.Template.Spec.InitContainers, initContainer)

	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, v1.Volume{
		Name: workspaceVolName,
		VolumeSource: v1.VolumeSource{
			EmptyDir: &v1.EmptyDirVolumeSource{},
		},
	})
	job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
		Name:      workspaceVolName,
		MountPath: workspaceDir,
	})

	// Replace the context with the tar.gz archive
	job.Spec.Template.Spec.Containers[0].Args = append(job.Spec.Template.Spec.Containers[0].Args, "--context=tar://"+archiveFilePath)

	return job, nil
}

func (k *Kaniko) prepareArgs(b *builder.BuilderOptions) []string {
	args := []string{
		"--context=" + b.BuildContext,
		"--destination=" + b.ImageName,
		"--skip-tls-verify", // Skip TLS verification for all registries
	}

	if b.Cache != nil && b.Cache.Enabled {
		args = append(args, "--cache=true")
		if b.Cache.Dir != "" {
			args = append(args, "--cache-dir="+b.Cache.Dir)
		}
		if b.Cache.Repo != "" {
			args = append(args, "--cache-repo="+b.Cache.Repo)
		}
	}

	// Append other args e.g. build args
	for _, a := range b.Args {
		args = append(args, fmt.Sprintf("%s=%s", a.GetKey(), a.GetValue()))
	}

	return args
}
