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

	DefaultParallelism  = int32(1)
	DefaultBackoffLimit = int32(5)

	MinioBucketName  = "kaniko"
	EphemeralStorage = "10Gi"
)

type Kaniko struct {
	system.SystemDependencies
	ContentName string // Name of the content pushed to Minio
}

var _ builder.Builder = &Kaniko{}

func (k *Kaniko) Build(ctx context.Context, b *builder.BuilderOptions) (logs string, err error) {
	job, err := k.prepareJob(ctx, b)
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

	if err := k.cleanup(ctx, kJob); err != nil {
		return "", ErrCleaningUp.Wrap(err)
	}

	if kJob.Status.Succeeded == 0 {
		return logs, ErrBuildFailed
	}

	return logs, nil
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

func (k *Kaniko) cleanup(ctx context.Context, job *batchv1.Job) error {
	err := k.K8sClient.Clientset().BatchV1().Jobs(k.K8sClient.Namespace()).
		Delete(ctx, job.Name, metav1.DeleteOptions{
			PropagationPolicy: &[]metav1.DeletionPropagation{metav1.DeletePropagationBackground}[0],
		})
	if err != nil {
		return ErrDeletingJob.Wrap(err)
	}

	// Delete the associated Pods
	err = k.K8sClient.Clientset().CoreV1().Pods(k.K8sClient.Namespace()).
		DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
		})
	if err != nil {
		return ErrDeletingPods.Wrap(err)
	}

	// Delete the content pushed to Minio
	if k.ContentName != "" {
		if err := k.MinioClient.Delete(ctx, k.ContentName, MinioBucketName); err != nil {
			return ErrDeletingMinioContent.Wrap(err)
		}
	}

	return nil
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

	parallelism := DefaultParallelism
	backoffLimit := DefaultBackoffLimit
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
							Args: []string{
								`--context=` + b.BuildContext,
								// TODO: see if we need it or not
								// --git gitoptions    Branch to clone if build context is a git repository (default branch=,single-branch=false,recurse-submodules=false)

								// TODO: we might need to add some options to get the auth token for the registry
								"--destination=" + b.Destination,
								// "--verbosity=debug", // log level
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceEphemeralStorage: ephemeralStorage,
								},
							},
						},
					},
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

	// TODO: we need to add some configs to get the auth token for the cache repo
	if b.Cache != nil && b.Cache.Enabled {
		cacheArgs := []string{"--cache=true"}
		if b.Cache.Dir != "" {
			cacheArgs = append(cacheArgs, "--cache-dir="+b.Cache.Dir)
		}
		if b.Cache.Repo != "" {
			cacheArgs = append(cacheArgs, "--cache-repo="+b.Cache.Repo)
		}
		job.Spec.Template.Spec.Containers[0].Args = append(job.Spec.Template.Spec.Containers[0].Args, cacheArgs...)
	}

	// Add extra args
	job.Spec.Template.Spec.Containers[0].Args = append(job.Spec.Template.Spec.Containers[0].Args, b.Args...)

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
	k.ContentName = hex.EncodeToString(hash.Sum(nil))

	if err := k.MinioClient.Push(ctx, bytes.NewReader(archiveData), k.ContentName, MinioBucketName); err != nil {
		return nil, err
	}

	s3URL, err := k.MinioClient.GetURL(ctx, k.ContentName, MinioBucketName)
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
