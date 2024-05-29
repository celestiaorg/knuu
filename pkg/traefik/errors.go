package traefik

import (
	"github.com/celestiaorg/knuu/pkg/errors"
)

type Error = errors.Error

var (
	ErrTraefikDeploymentCreationFailed   = errors.New("TraefikDeploymentCreationFailed", "error creating Traefik deployment")
	ErrTraefikServiceCreationFailed      = errors.New("TraefikServiceCreationFailed", "error creating Traefik service")
	ErrTraefikClientNotInitialized       = errors.New("TraefikClientNotInitialized", "Traefik client not initialized")
	ErrTraefikIPNotFound                 = errors.New("TraefikIPNotFound", "Traefik IP not found")
	ErrTraefikFailedToGetService         = errors.New("TraefikFailedToGetService", "error getting Traefik service")
	ErrTraefikLoadBalancerIPNotAvailable = errors.New("TraefikLoadBalancerIPNotAvailable", "Traefik LoadBalancer IP not available")
	ErrTraefikFailedToGetNodes           = errors.New("TraefikFailedToGetNodes", "error getting Traefik nodes")
	ErrTraefikNoNodesFound               = errors.New("TraefikNoNodesFound", "no Traefik nodes found")
	ErrTraefikTimeoutWaitingForReady     = errors.New("TraefikTimeoutWaitingForReady", "Traefik timeout waiting for ready")
	ErrTraefikFailedToCreateService      = errors.New("TraefikFailedToCreateService", "error creating Traefik service")
	ErrTraefikRoleCreationFailed         = errors.New("TraefikRoleCreationFailed", "error creating Traefik role")
	ErrTraefikRoleBindingCreationFailed  = errors.New("TraefikRoleBindingCreationFailed", "error creating Traefik role binding")
	ErrFailedToCreateServiceAccount      = errors.New("FailedToCreateServiceAccount", "error creating service account")
	ErrTraefikMiddlewareCreationFailed   = errors.New("TraefikMiddlewareCreationFailed", "error creating Traefik middleware")
	ErrTraefikIngressRouteCreationFailed = errors.New("TraefikIngressRouteCreationFailed", "error creating Traefik ingress route")
	ErrGeneratingRandomK8sName           = errors.New("GeneratingRandomK8sName", "error generating random K8s name")
	ErrTraefikFailedToParseQuantity      = errors.New("TraefikFailedToParseQuantity", "error parsing resource quantity")
)
