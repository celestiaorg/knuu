package system

import (
	"context"
	"fmt"
	"sync"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/instance"
)

func (s *Suite) TestFileCached() {
	const (
		namePrefix        = "file-cached"
		numberOfInstances = 10
		maxRetries        = 3
	)

	// Setup
	ctx := context.Background()

	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	instances := make([]*instance.Instance, numberOfInstances)

	instanceName := func(i int) string {
		return fmt.Sprintf("%s-web%d", namePrefix, i+1)
	}

	for i := 0; i < numberOfInstances; i++ {
		instances[i] = s.CreateNginxInstanceWithVolume(ctx, instanceName(i))
	}

	var wgFolders sync.WaitGroup
	for i, ins := range instances {
		wgFolders.Add(1)
		go func(i int, instance *instance.Instance) {
			defer wgFolders.Done()
			err := s.RetryOperation(func() error {
				return instance.Storage().AddFile(resourcesHTML+"/index.html", e2e.NginxHTMLPath+"/index.html", "0:0")
			}, maxRetries)
			// adding the folder after the Commit, it will help us to use a cached image.
			s.Require().NoError(err, "adding file to '%v'", instanceName(i))
		}(i, ins)
	}
	wgFolders.Wait()

	// Test logic
	for _, i := range instances {
		i := i
		err := s.RetryOperation(func() error {
			if err := i.Build().Commit(ctx); err != nil {
				return fmt.Errorf("committing instance: %w", err)
			}
			if err := i.Execution().StartAsync(ctx); err != nil {
				return fmt.Errorf("starting instance: %w", err)
			}
			return nil
		}, maxRetries)
		s.Require().NoError(err)
	}

	for _, i := range instances {
		err := s.RetryOperation(func() error {
			webIP, err := i.Network().GetIP(ctx)
			if err != nil {
				return fmt.Errorf("getting IP: %w", err)
			}

			if err := i.Execution().WaitInstanceIsRunning(ctx); err != nil {
				return fmt.Errorf("waiting for instance to run: %w", err)
			}

			wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
			if err != nil {
				return fmt.Errorf("executing wget: %w", err)
			}

			s.Assert().Contains(wget, "Hello World!")
			return nil
		}, maxRetries)
		s.Require().NoError(err)
	}
}
