package kaniko

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/names"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	kanikoImage         = "gcr.io/kaniko-project/executor:debug" // debug has a shell
	kanikoContainerName = "kaniko-container"
	kanikoJobNamePrefix = "kaniko-build-job"
)

type Kaniko struct {
	K8sClientset kubernetes.Interface
	K8sNamespace string
}

var _ builder.Builder = &Kaniko{}

func (k *Kaniko) Build(ctx context.Context, b *builder.BuilderOptions) (logs string, err error) {
	job, err := prepareJob(b)
	if err != nil {
		return "", ErrPreparingJob.Wrap(err)
	}

	cJob, err := k.K8sClientset.BatchV1().Jobs(k.K8sNamespace).Create(ctx, job, metav1.CreateOptions{})
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

	return logs, nil
}

func (k *Kaniko) waitForJobCompletion(ctx context.Context, job *batchv1.Job) (*batchv1.Job, error) {
	watcher, err := k.K8sClientset.BatchV1().Jobs(k.K8sNamespace).Watch(ctx, metav1.ListOptions{
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
	podList, err := k.K8sClientset.CoreV1().Pods(k.K8sNamespace).List(ctx, metav1.ListOptions{
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

	containerName := pod.Spec.Containers[0].Name

	logOptions := v1.PodLogOptions{
		Container: containerName,
	}

	req := k.K8sClientset.CoreV1().Pods(k.K8sNamespace).GetLogs(pod.Name, &logOptions)
	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

func (k *Kaniko) cleanup(ctx context.Context, job *batchv1.Job) error {
	err := k.K8sClientset.BatchV1().Jobs(k.K8sNamespace).
		Delete(ctx, job.Name, metav1.DeleteOptions{
			PropagationPolicy: &[]metav1.DeletionPropagation{metav1.DeletePropagationBackground}[0],
		})
	if err != nil {
		return ErrDeletingJob.Wrap(err)
	}

	// Delete the associated Pods
	err = k.K8sClientset.CoreV1().Pods(k.K8sNamespace).
		DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
		})
	if err != nil {
		return ErrDeletingPods.Wrap(err)
	}

	return nil
}

func DefaultCacheOptions(buildContext string) (*builder.CacheOptions, error) {
	if buildContext == "" {
		return nil, ErrBuildContextEmpty
	}
	hash := sha256.New()
	_, err := hash.Write([]byte(buildContext))
	if err != nil {
		return nil, err
	}
	hashStr := hex.EncodeToString(hash.Sum(nil))

	return &builder.CacheOptions{
		Enabled: true,
		Dir:     "",
		Repo:    fmt.Sprintf("ttl.sh/%s:24h", hashStr),
	}, nil
}

func prepareJob(b *builder.BuilderOptions) (*batchv1.Job, error) {
	jobName, err := names.NewRandomK8(kanikoJobNamePrefix)
	if err != nil {
		return nil, ErrGeneratingUUID.Wrap(err)
	}

	oneInt32 := int32(1)
	fiveInt32 := int32(5)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Spec: batchv1.JobSpec{
			Parallelism:  &oneInt32,  // Set parallelism to 1 to ensure only one Pod
			BackoffLimit: &fiveInt32, // Retry the Job at most 5 times
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
							},
						},
					},
					RestartPolicy: "Never", // Ensure that the Pod does not restart
				},
			},
		},
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
