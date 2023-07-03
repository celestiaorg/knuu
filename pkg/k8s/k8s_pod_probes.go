package k8s

import "fmt"

// To keep the structs minimal, everything that is not needed right now is commented out
// If you need it, uncomment it and add the needed fields
// This way we can keep the structs minimal and add more without a breaking change

// Probe is a structure to hold the probes for an instance
type Probe struct {
	//Exec                          *ExecAction
	HTTPGet             *HTTPGetAction   // Only one action is allowed, deployment will fail otherwise
	TCPSocket           *TCPSocketAction // Only one action is allowed, deployment will fail otherwise
	InitialDelaySeconds int32            // Optional
	//TerminationGracePeriodSeconds int64
	//PeriodSeconds                 int32
	//TimeoutSeconds                int32
	//FailureThreshold              int32
	//SuccessThreshold              int32
	//GRPC                          *GRPCAction
}

func (p *Probe) String() string {
	if p.HTTPGet != nil {
		return fmt.Sprintf("HTTPGet: %s, InitialDelaySeconds: %d", p.HTTPGet.String(), p.InitialDelaySeconds)
	}
	if p.TCPSocket != nil {
		return fmt.Sprintf("TCPSocket: %s, InitialDelaySeconds: %d", p.TCPSocket.String(), p.InitialDelaySeconds)
	}
	return ""
}

//type ExecAction struct {
//	Command []string
//}

type HTTPGetAction struct {
	Port int32
	//Host        string
	//HttpHeaders []HTTPHeader
	Path string
	//Scheme      string
}

func (h *HTTPGetAction) String() string {
	return fmt.Sprintf("http://%s%s", h.Port, h.Path)
}

type TCPSocketAction struct {
	Port string
	//Host string
}

func (t *TCPSocketAction) String() string {
	return fmt.Sprintf("tcp://%s", t.Port)
}

//type GRPCAction struct {
//	Port    int32
//	Service string
//}

//type HTTPHeader struct {
//	Name  string
//	Value string
//}
