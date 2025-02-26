package longrunning

import (
	"context"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"
	"github.com/celestiaorg/knuu/pkg/knuu"
)

const (
	testTimeout = time.Minute * 5
)

func createInstance(ctx context.Context, name, testScope string, image string) (*instance.Instance, error) {
	knOpts := knuu.Options{Timeout: testTimeout, SkipCleanup: true}
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

	if err := ins.Build().SetImage(ctx, image); err != nil {
		return nil, err
	}

	if err := ins.Build().Commit(ctx); err != nil {
		return nil, err
	}

	if err := ins.Build().SetStartCommand("sleep", "infinity"); err != nil {
		return nil, err
	}

	return ins, nil
}

func createInstanceAndStart(ctx context.Context, name, testScope string, image string) (*instance.Instance, error) {
	ins, err := createInstance(ctx, name, testScope, image)
	if err != nil {
		return nil, err
	}

	if err := ins.Execution().Start(ctx); err != nil {
		return nil, err
	}

	return ins, nil
}
