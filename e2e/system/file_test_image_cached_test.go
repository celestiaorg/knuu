package system

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"
)

func (s *Suite) TestFileCached() {
	const (
		namePrefix        = "file-cached"
		numberOfInstances = 10
		maxRetries        = 3
	)
	s.T().Parallel()

	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	instances := make([]*instance.Instance, numberOfInstances)

	instanceName := func(i int) string {
		return fmt.Sprintf("%s-web%d", namePrefix, i+1)
	}

	for i := 0; i < numberOfInstances; i++ {
		instances[i] = s.createNginxInstanceWithVolume(ctx, instanceName(i))
	}

	var wgFolders sync.WaitGroup
	for i, ins := range instances {
		wgFolders.Add(1)
		go func(i int, instance *instance.Instance) {
			defer wgFolders.Done()
			err := retryOperation(func() error {
				return instance.Storage().AddFile(resourcesHTML+"/index.html", nginxHTMLPath+"/index.html", "0:0")
			}, maxRetries)
			// adding the folder after the Commit, it will help us to use a cached image.
			s.Require().NoError(err, "adding file to '%v'", instanceName(i))
		}(i, ins)
	}
	wgFolders.Wait()

	s.T().Cleanup(func() {
		all := append(instances, executor)
		err := instance.BatchDestroy(ctx, all...)
		if err != nil {
			s.T().Logf("error destroying instance: %v", err)
		}
	})

	// Test logic
	for _, i := range instances {
		err := retryOperation(func() error {
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
		err := retryOperation(func() error {
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
