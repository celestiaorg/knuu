package kaniko

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/google/uuid"
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
	k8sClientset kubernetes.Interface
	k8sNamespace string
}

var _ builder.Builder = &Kaniko{}

func NewKaniko(k8sClientset kubernetes.Interface, namespace string) *Kaniko {
	return &Kaniko{
		k8sClientset: k8sClientset,
		k8sNamespace: namespace,
	}
}

func (k *Kaniko) Build(ctx context.Context, b *builder.BuilderOptions) (logs string, err error) {
	job, err := prepareJob(b)
	if err != nil {
		return "", fmt.Errorf("error preparing Job: %w", err)
	}

	fmt.Printf("Creating Job: %s\n", job.Name)

	cJob, err := k.k8sClientset.BatchV1().Jobs(k.k8sNamespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("error creating Job: %w", err)
	}

	fmt.Printf("Waiting for Job completion: %s\n", cJob.Name)

	kJob, err := k.waitForJobCompletion(ctx, cJob)
	if err != nil {
		return "", fmt.Errorf("error waiting for Job completion: %w", err)
	}

	fmt.Printf("Getting Pod from Job: %s\n", kJob.Name)

	pod, err := k.getFirstPodFromJob(ctx, kJob)
	if err != nil {
		return "", fmt.Errorf("error getting Pod from Job: %w", err)
	}

	fmt.Printf("Getting container logs from Pod: %s\n", pod.Name)

	logs, err = k.getContainerLogs(ctx, pod)
	if err != nil {
		return "", fmt.Errorf("error getting container logs: %w", err)
	}

	fmt.Printf("Cleaning up Job: %s\n", kJob.Name)

	if err := k.cleanup(ctx, kJob); err != nil {
		return "", fmt.Errorf("error cleaning up: %w", err)
	}

	fmt.Printf("Build completed successfully\n")

	return logs, nil
}

func (k *Kaniko) waitForJobCompletion(ctx context.Context, job *batchv1.Job) (*batchv1.Job, error) {
	watcher, err := k.k8sClientset.BatchV1().Jobs(k.k8sNamespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", job.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("error watching Job: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return nil, fmt.Errorf("watch channel closed unexpectedly")
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
			return nil, fmt.Errorf("context cancelled")
		}
	}
}

func (k *Kaniko) getFirstPodFromJob(ctx context.Context, job *batchv1.Job) (*v1.Pod, error) {
	// Assuming there's only one Pod template in the Job
	podList, err := k.k8sClientset.CoreV1().Pods(k.k8sNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing Pods: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no Pods found for the Job: %s", job.Name)
	}

	return &podList.Items[0], nil
}

func (k *Kaniko) getContainerLogs(ctx context.Context, pod *v1.Pod) (string, error) {
	if len(pod.Spec.Containers) == 0 {
		return "", fmt.Errorf("no containers found in Pod: %s", pod.Name)
	}

	containerName := pod.Spec.Containers[0].Name

	logOptions := v1.PodLogOptions{
		Container: containerName,
	}

	req := k.k8sClientset.CoreV1().Pods(k.k8sNamespace).GetLogs(pod.Name, &logOptions)
	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("error retrieving container logs: %w", err)
	}

	return string(logs), nil
}

func (k *Kaniko) cleanup(ctx context.Context, job *batchv1.Job) error {
	err := k.k8sClientset.BatchV1().Jobs(k.k8sNamespace).
		Delete(ctx, job.Name, metav1.DeleteOptions{
			PropagationPolicy: &[]metav1.DeletionPropagation{metav1.DeletePropagationBackground}[0],
		})
	if err != nil {
		return fmt.Errorf("error deleting Job: %w", err)
	}

	// Delete the associated Pods
	err = k.k8sClientset.CoreV1().Pods(k.k8sNamespace).
		DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", job.Name),
		})
	if err != nil {
		return fmt.Errorf("error deleting Pods: %w", err)
	}

	return nil
}

func GetDefaultCacheOptions(buildContext string) (*builder.CacheOptions, error) {
	if buildContext == "" {
		return nil, fmt.Errorf("build context cannot be empty")
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
	jobName, err := generateK8sName(kanikoJobNamePrefix)
	if err != nil {
		return nil, fmt.Errorf("error generating Job name: %w", err)
	}

	oneInt32 := int32(1)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Spec: batchv1.JobSpec{
			Parallelism: &oneInt32, // Set parallelism to 1 to ensure only one Pod
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
					RestartPolicy: "IfFailed",
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

func generateK8sName(name string) (string, error) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("error generating UUID: %w", err)
	}
	return fmt.Sprintf("%s-%s", name, uuid.String()[:8]), nil
}
