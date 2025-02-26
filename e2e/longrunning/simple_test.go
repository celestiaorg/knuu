package longrunning

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	alpineImage = "alpine:3.20.3"
)

func TestSimple(t *testing.T) {
	const (
		instanceName = "simple-id"
		fileContent  = "identifier:12345"
	)

	ctx := context.Background()

	ins1, err := createInstanceAndStart(ctx, instanceName, "", alpineImage)
	require.NoError(t, err)
	testScope := ins1.Scope

	t.Logf("Scope: %s", testScope)

	_, err = ins1.Execution().ExecuteCommand(ctx, "echo", fileContent, ">", "/tmp/test-id")
	require.NoError(t, err)

	t.Logf("Waiting for 5 seconds to simulate a long running process")
	time.Sleep(5 * time.Second)

	ins2, err := createInstanceAndStart(ctx, instanceName, testScope, alpineImage)
	require.NoError(t, err)

	out, err := ins2.Execution().ExecuteCommand(ctx, "cat", "/tmp/test-id")
	require.NoError(t, err)
	require.Contains(t, out, fileContent)
}
