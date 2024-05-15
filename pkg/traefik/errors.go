package traefik

import (
	"fmt"
)

type Error struct {
	Code    string
	Message string
	Err     error
	Params  []interface{}
}

func (e *Error) Error() string {
	msg := fmt.Sprintf(e.Message, e.Params...)
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", msg, e.Err)
	}
	return msg
}

func (e *Error) Wrap(err error) error {
	e.Err = err
	return e
}

func (e *Error) WithParams(params ...interface{}) *Error {
	e.Params = params
	return e
}

var (
	ErrTraefikDeploymentCreationFailed   = &Error{Code: "TraefikDeploymentCreationFailed", Message: "error creating Traefik deployment"}
	ErrTraefikServiceCreationFailed      = &Error{Code: "TraefikServiceCreationFailed", Message: "error creating Traefik service"}
	ErrTraefikClientNotInitialized       = &Error{Code: "TraefikClientNotInitialized", Message: "Traefik client not initialized"}
	ErrTraefikIPNotFound                 = &Error{Code: "TraefikIPNotFound", Message: "Traefik IP not found"}
	ErrTraefikFailedToGetService         = &Error{Code: "TraefikFailedToGetService", Message: "error getting Traefik service"}
	ErrTraefikLoadBalancerIPNotAvailable = &Error{Code: "TraefikLoadBalancerIPNotAvailable", Message: "Traefik LoadBalancer IP not available"}
	ErrTraefikFailedToGetNodes           = &Error{Code: "TraefikFailedToGetNodes", Message: "error getting Traefik nodes"}
	ErrTraefikNoNodesFound               = &Error{Code: "TraefikNoNodesFound", Message: "no Traefik nodes found"}
	ErrTraefikTimeoutWaitingForReady     = &Error{Code: "TraefikTimeoutWaitingForReady", Message: "Traefik timeout waiting for ready"}
	ErrTraefikFailedToCreateService      = &Error{Code: "TraefikFailedToCreateService", Message: "error creating Traefik service"}
	ErrTraefikRoleCreationFailed         = &Error{Code: "TraefikRoleCreationFailed", Message: "error creating Traefik role"}
	ErrTraefikRoleBindingCreationFailed  = &Error{Code: "TraefikRoleBindingCreationFailed", Message: "error creating Traefik role binding"}
	ErrFailedToCreateServiceAccount      = &Error{Code: "FailedToCreateServiceAccount", Message: "error creating service account"}
	ErrTraefikMiddlewareCreationFailed   = &Error{Code: "TraefikMiddlewareCreationFailed", Message: "error creating Traefik middleware"}
	ErrTraefikIngressRouteCreationFailed = &Error{Code: "TraefikIngressRouteCreationFailed", Message: "error creating Traefik ingress route"}
	ErrGeneratingRandomK8sName           = &Error{Code: "GeneratingRandomK8sName", Message: "error generating random K8s name"}
)
