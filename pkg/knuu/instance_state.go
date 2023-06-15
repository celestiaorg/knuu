package knuu

// InstanceState represents the state of the instance
type InstanceState int

// Possible states of the instance
const (
	None InstanceState = iota
	Preparing
	Committed
	Started
	Stopped
	Destroyed
)

// String returns the string representation of the state
func (s InstanceState) String() string {
	if s < 0 || s > 4 {
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
