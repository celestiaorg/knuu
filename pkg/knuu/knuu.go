// Package knuu provides the core functionality of knuu.
package knuu

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/celestiaorg/knuu/pkg/builder"
	"github.com/celestiaorg/knuu/pkg/builder/kaniko"
	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/k8s"
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
	timeout      time.Duration
	proxyEnabled bool
}

type Option func(*Knuu)

func WithImageBuilder(builder builder.Builder) Option {
	return func(k *Knuu) {
		k.ImageBuilder = builder
	}
}

func WithTestScope(scope string) Option {
	return func(k *Knuu) {
		k.TestScope = k8s.SanitizeName(scope)
	}
}

// This timeout indicates how long the test will run before it is considered failed.
func WithTimeout(timeout time.Duration) Option {
	return func(k *Knuu) {
		k.timeout = timeout
	}
}

func WithMinio(minio *minio.Minio) Option {
	return func(k *Knuu) {
		k.MinioCli = minio
	}
}

func WithK8s(k8s k8s.KubeManager) Option {
	return func(k *Knuu) {
		k.K8sCli = k8s
	}
}

func WithLogger(logger *logrus.Logger) Option {
	return func(k *Knuu) {
		k.Logger = logger
	}
}

func WithProxyEnabled() Option {
	return func(k *Knuu) {
		k.proxyEnabled = true
	}
}

func New(ctx context.Context, opts ...Option) (*Knuu, error) {
	if err := godotenv.Load(); err != nil {
		if !os.IsNotExist(err) {
			return nil, ErrCannotLoadEnv.Wrap(err)
		}
		logrus.Info("The .env file does not exist, continuing without loading environment variables.")
	}

	k := &Knuu{}
	for _, opt := range opts {
		opt(k)
	}

	k.StartTime = time.Now().UTC().Format(TimeFormat)

	// handle default values
	if k.Logger == nil {
		k.Logger = defaultLogger()
	}

	if k.TestScope == "" {
		t := time.Now()
		k.TestScope = fmt.Sprintf("%s-%03d", t.Format("20060102-150405"), t.Nanosecond()/1e6)
	}

	if k.timeout == 0 {
		k.timeout = defaultTimeout
	}

	if k.K8sCli == nil {
		var err error
		k.K8sCli, err = k8s.NewClient(ctx, k.TestScope)
		if err != nil {
			return nil, ErrCannotInitializeK8s.Wrap(err)
		}
	}

	if k.MinioCli == nil {
		k.MinioCli = &minio.Minio{
			K8s: k.K8sCli,
		}
	}

	if k.ImageBuilder == nil {
		k.ImageBuilder = &kaniko.Kaniko{
			K8s:   k.K8sCli,
			Minio: k.MinioCli,
		}
	}

	if k.proxyEnabled {
		k.Proxy = &traefik.Traefik{
			K8s: k.K8sCli,
		}
		if err := k.Proxy.Deploy(ctx); err != nil {
			return nil, ErrCannotDeployTraefik.Wrap(err)
		}
		endpoint, err := k.Proxy.Endpoint(ctx)
		if err != nil {
			return nil, ErrCannotGetTraefikEndpoint.Wrap(err)
		}
		k.Logger.Debugf("Proxy endpoint: %s", endpoint)
	}

	if err := k.handleTimeout(ctx); err != nil {
		return nil, ErrCannotHandleTimeout.Wrap(err)
	}

	return k, nil
}

func (k *Knuu) Scope() string {
	return k.TestScope
}

func (k *Knuu) CleanUp(ctx context.Context) error {
	return k.K8sCli.DeleteNamespace(ctx, k.TestScope)
}

func (k *Knuu) HandleStopSignal() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		<-stop
		logrus.Info("Received signal to stop, cleaning up resources...")
		if err := CleanUp(); err != nil {
			logrus.Errorf("Error deleting namespace: %v", err)
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

	if err := inst.SetImage(ctx, timeoutHandlerImage); err != nil {
		return ErrCannotSetImage.Wrap(err)
	}
	if err := inst.Commit(); err != nil {
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
			k.TestScope, k.K8sCli.Namespace(), instance.TimeoutHandlerInstance.String(), k.K8sCli.Namespace()))

	// Delete the namespace as it was created by knuu.
	k.Logger.Debugf("The namespace generated [%s] will be deleted", k.K8sCli.Namespace())
	commands = append(commands, fmt.Sprintf("kubectl delete namespace %s", k.K8sCli.Namespace()))

	// Delete all labeled resources within the namespace.
	// Unlike the previous command that excludes certain types, this command ensures that everything remaining is deleted.
	commands = append(commands, fmt.Sprintf("kubectl delete all,pvc,netpol,roles,serviceaccounts,rolebindings,configmaps -l knuu.sh/scope=%s -n %s", k.TestScope, k.K8sCli.Namespace()))

	finalCmd := strings.Join(commands, " && ")

	// Run the command
	if err := inst.SetCommand("sh", "-c", finalCmd); err != nil {
		k.Logger.Debugf("The full command generated is [%s]", finalCmd)
		return ErrCannotSetCommand.Wrap(err)
	}

	rule := rbacv1.PolicyRule{
		Verbs:     []string{"*"},
		APIGroups: []string{"*"},
		Resources: []string{"*"},
	}

	if err := inst.AddPolicyRule(rule); err != nil {
		return ErrCannotAddPolicyRule.Wrap(err)
	}
	if err := inst.Start(ctx); err != nil {
		return ErrCannotStartInstance.Wrap(err)
	}

	return nil
}

func defaultLogger() *logrus.Logger {
	logger := logrus.New()

	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			directory := path.Base(path.Dir(f.File))
			return "", directory + "/" + filename + ":" + strconv.Itoa(f.Line)
		},
	})

	// Enable reporting the file and line
	logger.SetReportCaller(true)

	customLevel := os.Getenv("LOG_LEVEL")
	if customLevel != "" {
		err := logger.Level.UnmarshalText([]byte(customLevel))
		if err != nil {
			logger.Warnf("Failed to parse LOG_LEVEL: %v, defaulting to INFO", err)
		}
	}
	logger.Info("LOG_LEVEL: ", logger.GetLevel())

	return logger
}
