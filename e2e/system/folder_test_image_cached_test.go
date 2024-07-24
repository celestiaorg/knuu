package system

import (
	"context"
	"fmt"
	"sync"

	"github.com/celestiaorg/knuu/pkg/instance"
)

func (s *Suite) TestFolderCached() {
	const (
		namePrefix        = "folder-cached"
		numberOfInstances = 10
	)
	s.T().Parallel()

	// Setup
	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	instances := make([]*instance.Instance, numberOfInstances)
	for i := 0; i < numberOfInstances; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i+1)
		instances[i] = s.createNginxInstanceWithVolume(ctx, name)
	}

	var wgFolders sync.WaitGroup
	for _, i := range instances {
		wgFolders.Add(1)
		go func(i *instance.Instance) {
			defer wgFolders.Done()
			// adding the folder after the Commit, it will help us to use a cached image.
			err := i.AddFolder(resourcesHTML, nginxHTMLPath, "0:0")
			s.Require().NoError(err, "adding file to '%v'", i.Name())
		}(i)
	}
	wgFolders.Wait()

	// Cleanup
	s.T().Cleanup(func() {
		all := append(instances, executor)
		err := instance.BatchDestroy(ctx, all...)
		if err != nil {
			s.T().Logf("error destroying instance: %v", err)
		}
	})

	// Test logic
	for _, i := range instances {
		s.Require().NoError(i.Commit())
		s.Require().NoError(i.StartAsync(ctx))
	}

	for _, i := range instances {
		webIP, err := i.GetIP(ctx)
		s.Require().NoError(err)

		s.Require().NoError(i.WaitInstanceIsRunning(ctx))

		wget, err := executor.ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
		s.Require().NoError(err)

		s.Assert().Contains(wget, "Hello World!")
	}
}
