package longrunning

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	alpineImage = "alpine:3.20.3"
	testTimeout = time.Minute * 15
)

func TestSimple(t *testing.T) {
	const (
		instanceName = "simple-id"
		fileContent  = "identifier:12345"
	)

	ctx := context.Background()

	ins1, err := createInstance(ctx, instanceName, "")
	require.NoError(t, err)
	testScope := ins1.Scope

	t.Logf("Scope: %s", testScope)

	_, err = ins1.Execution().ExecuteCommand(ctx, "echo", fileContent, ">", "/tmp/test-id")
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	ins2, err := createInstance(ctx, instanceName, testScope)
	require.NoError(t, err)

	out, err := ins2.Execution().ExecuteCommand(ctx, "cat", "/tmp/test-id")
	require.NoError(t, err)
	require.Contains(t, out, fileContent)
}

func createInstance(ctx context.Context, name, testScope string) (*instance.Instance, error) {
	knOpts := knuu.Options{Timeout: testTimeout}
	if testScope != "" {
		knOpts.Scope = testScope
	}

	kn, err := knuu.New(ctx, knOpts)
	if err != nil {
		return nil, err
	}

	kn.HandleStopSignal(ctx)

	ins, err := kn.NewInstance(name)
	if err != nil {
		return nil, err
	}

	if err := ins.Build().SetImage(ctx, alpineImage); err != nil {
		return nil, err
	}

	if err := ins.Build().Commit(ctx); err != nil {
		return nil, err
	}

	if err := ins.Build().SetStartCommand("sleep", "infinity"); err != nil {
		return nil, err
	}

	if err := ins.Execution().Start(ctx); err != nil {
		return nil, err
	}

	return ins, nil
}
