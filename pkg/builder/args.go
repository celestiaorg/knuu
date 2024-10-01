package builder

const buildArgKey = "--build-arg"

type ArgInterface interface {
	GetKey() string
	GetValue() string
}

// BuildArg is a build argument that can be passed to the builder.
type BuildArg struct {
	Value string
}

var _ ArgInterface = &BuildArg{}

func (b *BuildArg) GetKey() string {
	return buildArgKey
}

func (b *BuildArg) GetValue() string {
	return b.Value
}

// CustomArg is a custom argument that can be passed to the builder.
type CustomArg struct {
	Key   string
	Value string
}

var _ ArgInterface = &CustomArg{}

func (c *CustomArg) GetKey() string {
	return c.Key
}

func (c *CustomArg) GetValue() string {
	return c.Value
}
