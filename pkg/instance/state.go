package instance

import "github.com/sirupsen/logrus"

// InstanceState represents the state of the instance
type InstanceState int

// Possible states of the instance
const (
	StateNone InstanceState = iota
	StatePreparing
	StateCommitted
	StateStarted
	StateStopped
	StateDestroyed
)

// String returns the string representation of the state
func (s InstanceState) String() string {
	if s < 0 || s > 5 {
		return "Unknown"
	}
	return [...]string{"None", "Preparing", "Committed", "Started", "Stopped", "Destroyed"}[s]
}

// IsInState checks if the instance is in one of the provided states
func (i *Instance) IsInState(states ...InstanceState) bool {
	for _, s := range states {
		if i.state == s {
			return true
		}
	}
	return false
}

func (i *Instance) SetState(state InstanceState) {
	i.state = state
	i.Logger.WithFields(logrus.Fields{
		"instance": i.name,
		"state":    i.state.String(),
	}).Debug("set state of instance")
}

func (i *Instance) IsState(state InstanceState) bool {
	return i.state == state
}
