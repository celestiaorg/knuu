package instance

// InstanceType represents the type of the instance
type InstanceType int

// Possible types of the instance
const (
	UnknownInstance InstanceType = iota
	BasicInstance
	TimeoutHandlerInstance
)

// String returns the string representation of the type
func (s InstanceType) String() string {
	switch s {
	case BasicInstance:
		return "BasicInstance"
	case TimeoutHandlerInstance:
		return "TimeoutHandlerInstance"
	case UnknownInstance:
	default:
	}
	return "Unknown"

}
