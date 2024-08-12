// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/k8s"
	"github.com/celestiaorg/knuu/pkg/log"
	"github.com/celestiaorg/knuu/pkg/minio"
	"github.com/celestiaorg/knuu/pkg/system"
	"github.com/celestiaorg/knuu/pkg/traefik"
)

const (
	defaultTimeout     = 60 * time.Minute
	timeoutHandlerName = "timeout-handler"
	// FIXME: use supported kubernetes version images (use of latest could break) (https://github.com/celestiaorg/knuu/issues/116)
	timeoutHandlerImage = "docker.io/bitnami/kubectl:latest"

	TimeFormat = "20060102T150405Z"
)

type Knuu struct {
	system.SystemDependencies
	timeout time.Duration
}

type Options struct {
	K8sClient    k8s.KubeManager
	MinioClient  *minio.Minio
	ImageBuilder builder.Builder
	Scope        string
	ProxyEnabled bool
	Timeout      time.Duration
	Logger       *logrus.Logger
}

func New(ctx context.Context, opts Options) (*Knuu, error) {
	if err := validateOptions(opts); err != nil {
		return nil, err
	}

	k := &Knuu{
		SystemDependencies: system.SystemDependencies{
			K8sClient:    opts.K8sClient,
			MinioClient:  opts.MinioClient,
			ImageBuilder: opts.ImageBuilder,
			Logger:       opts.Logger,
			Scope:        opts.Scope,
			StartTime:    time.Now().UTC().Format(TimeFormat),
		},
		timeout: opts.Timeout,
	}

	if err := setDefaults(ctx, k); err != nil {
		return nil, err
	}

	if opts.ProxyEnabled {
		if err := setupProxy(ctx, k); err != nil {
			return nil, err
		}
	}

	return k, nil
}

func (k *Knuu) CleanUp(ctx context.Context) error {
	return k.K8sClient.DeleteNamespace(ctx, k.Scope)
}

func (k *Knuu) HandleStopSignal(ctx context.Context) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-stop
		k.Logger.Info("Received signal to stop, cleaning up resources...")
		if err := k.CleanUp(ctx); err != nil {
			k.Logger.Errorf("Error deleting namespace: %v", err)
		}
	}()
}

// handleTimeout creates a timeout handler that will delete all resources with the scope after the timeout
func (k *Knuu) handleTimeout(ctx context.Context) error {
	inst, err := k.NewInstance(timeoutHandlerName)
	if err != nil {
		return ErrCannotCreateInstance.Wrap(err)
	}
	inst.SetInstanceType(instance.TimeoutHandlerInstance)

	if err := inst.Build().SetImage(ctx, timeoutHandlerImage); err != nil {
		return ErrCannotSetImage.Wrap(err)
	}
	if err := inst.Build().Commit(ctx); err != nil {
		return ErrCannotCommitInstance.Wrap(err)
	}

	var commands []string

	// Wait for a specific period before executing the next operation.
	// This is useful to ensure that any previous operation has time to complete.
	commands = append(commands, fmt.Sprintf("sleep %d", int64(k.timeout.Seconds())))
	// Collects all resources (pods, services, etc.) within the specified namespace that match a specific label, excluding certain types,
	// and then deletes them. This is useful for cleaning up specific test resources before proceeding to delete the namespace.
	commands = append(commands,
		fmt.Sprintf("kubectl get all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/scope=%s -n %s -o json | jq -r '.items[] | select(.metadata.labels.\"knuu.sh/type\" != \"%s\") | \"\\(.kind)/\\(.metadata.name)\"' | xargs -r kubectl delete -n %s",
			k.Scope, k.K8sClient.Namespace(), instance.TimeoutHandlerInstance.String(), k.K8sClient.Namespace()))

	// Delete the namespace as it was created by knuu.
	k.Logger.Debugf("The namespace generated [%s] will be deleted", k.K8sClient.Namespace())
	commands = append(commands, fmt.Sprintf("kubectl delete namespace %s", k.K8sClient.Namespace()))

	// Delete all labeled resources within the namespace.
	// Unlike the previous command that excludes certain types, this command ensures that everything remaining is deleted.
	commands = append(commands, fmt.Sprintf("kubectl delete all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/scope=%s -n %s", k.Scope, k.K8sClient.Namespace()))

	finalCmd := strings.Join(commands, " && ")

	// Run the command
	if err := inst.Build().SetStartCommand("sh", "-c", finalCmd); err != nil {
		k.Logger.Debugf("The full command generated is [%s]", finalCmd)
		return ErrCannotSetStartCommand.Wrap(err)
	}

	rule := rbacv1.PolicyRule{
		Verbs:     []string{"*"},
		APIGroups: []string{"*"},
		Resources: []string{"*"},
	}

	if err := inst.Security().AddPolicyRule(rule); err != nil {
		return ErrCannotAddPolicyRule.Wrap(err)
	}
	if err := inst.Execution().Start(ctx); err != nil {
		return ErrCannotStartInstance.Wrap(err)
	}

	return nil
}

func DefaultScope() string {
	t := time.Now()
	return fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)
}

func validateOptions(opts Options) error {
	// When Minio is set, K8sClient must be set too
	// to make sure that there is only one source of truth for the k8s client
	if opts.MinioClient != nil && opts.K8sClient == nil {
		return ErrK8sClientNotSet
	}

	if opts.Scope != "" && opts.K8sClient != nil &&
		k8s.SanitizeName(opts.Scope) != opts.K8sClient.Namespace() {
		return ErrScopeMismatch.WithParams(opts.Scope, opts.K8sClient.Namespace())
	}
	return nil
}

func setDefaults(ctx context.Context, k *Knuu) error {
	if k.Logger == nil {
		k.Logger = log.DefaultLogger()
	}

	if k.Scope == "" {
		if k.K8sClient != nil {
			k.Scope = k.K8sClient.Namespace()
		} else {
			k.Scope = DefaultScope()
		}
	}
	k.Scope = k8s.SanitizeName(k.Scope)

	if k.timeout == 0 {
		k.timeout = defaultTimeout
	}

	if k.K8sClient == nil {
		var err error
		k.K8sClient, err = k8s.NewClient(ctx, k.Scope, k.Logger)
		if err != nil {
			return ErrCannotInitializeK8s.Wrap(err)
		}
	}

	if err := k.handleTimeout(ctx); err != nil {
		return ErrHandleTimeout.Wrap(err)
	}

	if k.ImageBuilder == nil {
		k.ImageBuilder = &kaniko.Kaniko{
			SystemDependencies: k.SystemDependencies,
		}
	}

	return nil
}

func setupProxy(ctx context.Context, k *Knuu) error {
	k.Proxy = &traefik.Traefik{
		K8sClient: k.K8sClient,
		Logger:    k.Logger,
	}
	if !k.Proxy.IsTraefikAPIAvailable(ctx) {
		return ErrTraefikAPINotAvailable
	}

	if err := k.Proxy.Deploy(ctx); err != nil {
		return ErrCannotDeployTraefik.Wrap(err)
	}
	endpoint, err := k.Proxy.Endpoint(ctx)
	if err != nil {
		return ErrCannotGetTraefikEndpoint.Wrap(err)
	}
	k.Logger.Debugf("Proxy endpoint: %s", endpoint)
	return nil
}
