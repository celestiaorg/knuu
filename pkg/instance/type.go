package instance

// InstanceType represents the type of the instance
type InstanceType int

// Possible types of the instance
const (
	BasicInstance InstanceType = iota
	ExecutorInstance
	TimeoutHandlerInstance
)

// String returns the string representation of the type
func (s InstanceType) String() string {
	if s < 0 || s > 2 {
		return "Unknown"
	}
	return [...]string{"BasicInstance", "ExecutorInstance", "TimeoutHandlerInstance"}[s]
}
