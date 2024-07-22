package system

import (
	"context"
	"fmt"
	"sync"

	"github.com/celestiaorg/knuu/pkg/instance"
)

func (s *Suite) TestFileCached() {
	const namePrefix = "file-cached"
	s.T().Parallel()
	// Setup
	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	const numberOfInstances = 10
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
			// adding the folder after the Commit, it will help us to use a cached image.
			err = instance.AddFile(resourcesHTML+"/index.html", nginxPath+"/index.html", "0:0")
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
