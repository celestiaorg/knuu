package system

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/celestiaorg/knuu/pkg/instance"
)

// TestOneVolumeNoFiles tests the scenario where we have one volume and no files.
// the initContainer command that it generates looks like:
// no initContainer command, as there is no volumes, nor files.
func (s *Suite) TestNoVolumesNoFiles() {
	const namePrefix = "no-volumes-no-files"
	// Setup

	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	target := s.createNginxInstance(ctx, namePrefix+"-target")
	s.Require().NoError(target.Build().Commit(ctx))

	// Test logic
	s.Require().NoError(target.Execution().StartAsync(ctx))

	webIP, err := target.Network().GetIP(ctx)
	s.Require().NoError(err)

	s.Require().NoError(target.Execution().WaitInstanceIsRunning(ctx))

	wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	s.Require().NoError(err)

	s.Assert().Contains(wget, "Welcome to nginx!")
}

// TestOneVolumeNoFiles tests the scenario where we have one volume and no files.
// the initContainer command that it generates looks like:
// mkdir -p /knuu && if [ -d /opt/vol1 ] && [ \"$(ls -A /opt/vol1)\" ]; then cp -r /opt/vol1/* /knuu//opt/vol1 && chown -R 0:0 /knuu/* ;fi
func (s *Suite) TestOneVolumeNoFiles() {
	const namePrefix = "one-volume-no-files"
	// Setup

	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	target := s.createNginxInstance(ctx, namePrefix+"-target")

	err = target.Storage().AddVolumeWithOwner("/opt/vol1", resource.MustParse("1Gi"), 0)
	s.Require().NoError(err)

	s.Require().NoError(target.Build().Commit(ctx))

	// Test logic
	s.Require().NoError(target.Execution().StartAsync(ctx))

	webIP, err := target.Network().GetIP(ctx)
	s.Require().NoError(err)

	s.Require().NoError(target.Execution().WaitInstanceIsRunning(ctx))

	wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	s.Require().NoError(err)

	s.Assert().Contains(wget, "Welcome to nginx!")
}

// TestNoVolumesOneFile tests the scenario where we have no volumes and one file.
// the initContainer command that it generates looks like:
// no initContainer command, as we do not have volumes.
func (s *Suite) TestNoVolumesOneFile() {
	const (
		namePrefix        = "no-volumes-one-file"
		numberOfInstances = 2
	)

	// Setup
	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	instances := make([]*instance.Instance, numberOfInstances)
	for i := 0; i < numberOfInstances; i++ {
		name := fmt.Sprintf("%s-%d", namePrefix, i+1)
		instances[i] = s.createNginxInstance(ctx, name)
	}

	var (
		wgFolders sync.WaitGroup
	)

	for _, i := range instances {
		wgFolders.Add(1)
		go func(i *instance.Instance) {
			defer wgFolders.Done()
			// adding the folder after the Commit, it will help us to use a cached image.
			err = i.Storage().AddFile(resourcesFileCMToFolder+"/test_1", nginxHTMLPath+"/index.html", "0:0")
			s.Require().NoError(err, "adding file to '%v'", i.Name())
		}(i)
	}
	wgFolders.Wait()

	// Test logic
	for _, i := range instances {
		s.Require().NoError(i.Build().Commit(ctx))
		s.Require().NoError(i.Execution().StartAsync(ctx))
	}

	for _, i := range instances {
		webIP, err := i.Network().GetIP(ctx)
		s.Require().NoError(err)

		err = i.Execution().WaitInstanceIsRunning(ctx)
		s.Require().NoError(err)

		wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
		s.Require().NoError(err)
		wget = strings.TrimSpace(wget)

		s.Assert().Equal("hello from 1", wget)
	}
}

// TestOneVolumeOneFile tests the scenario where we have one volume and one file.
// the initContainer command that it generates looks like:
// mkdir -p /knuu && mkdir -p /knuu/usr/share/nginx/html && chmod -R 777 /knuu//usr/share/nginx/html && if [ -d /usr/share/nginx/html ] && [ \"$(ls -A /usr/share/nginx/html)\" ]; then cp -r /usr/share/nginx/html/* /knuu//usr/share/nginx/html && chown -R 0:0 /knuu/* ;fi
func (s *Suite) TestOneVolumeOneFile() {
	const (
		namePrefix        = "one-volume-one-file"
		numberOfInstances = 2
	)
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
		go func(ins *instance.Instance) {
			defer wgFolders.Done()
			// adding the folder after the Commit, it will help us to use a cached image.
			err = ins.Storage().AddFile(resourcesFileCMToFolder+"/test_1", nginxHTMLPath+"/index.html", "0:0")
			s.Require().NoError(err, "adding file to '%v': %v", i.Name())
		}(i)
	}
	wgFolders.Wait()

	// Test logic
	for _, i := range instances {
		s.Require().NoError(i.Build().Commit(ctx))
		s.Require().NoError(i.Execution().StartAsync(ctx))
	}

	for _, i := range instances {
		webIP, err := i.Network().GetIP(ctx)
		s.Require().NoError(err)
		s.Require().NoError(i.Execution().WaitInstanceIsRunning(ctx))

		wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
		s.Require().NoError(err)
		wget = strings.TrimSpace(wget)

		s.Assert().Equal("hello from 1", wget)
	}
}

// TestOneVolumeTwoFiles tests the scenario where we have one volume and two files.
// the initContainer command that it generates looks like:
// mkdir -p /knuu && mkdir -p /knuu/usr/share/nginx/html && chmod -R 777 /knuu//usr/share/nginx/html && if [ -d /usr/share/nginx/html ] && [ \"$(ls -A /usr/share/nginx/html)\" ]; then cp -r /usr/share/nginx/html/* /knuu//usr/share/nginx/html && chown -R 0:0 /knuu/* ;fi
func (s *Suite) TestOneVolumeTwoFiles() {
	const (
		namePrefix        = "one-volume-two-files"
		numberOfInstances = 2
		maxRetries        = 3
	)

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
			err := s.retryOperation(func() error {
				err := i.Storage().AddFile(resourcesFileCMToFolder+"/test_1", nginxHTMLPath+"/index.html", "0:0")
				if err != nil {
					return fmt.Errorf("adding file to '%v': %w", i.Name(), err)
				}
				err = i.Storage().AddFile(resourcesFileCMToFolder+"/test_2", nginxHTMLPath+"/index-2.html", "0:0")
				if err != nil {
					return fmt.Errorf("adding file to '%v': %w", i.Name(), err)
				}
				return nil
			}, maxRetries)
			s.Require().NoError(err)
		}(i)
	}
	wgFolders.Wait()

	// Test logic
	for _, i := range instances {
		err := s.retryOperation(func() error {
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
		webIP, err := i.Network().GetIP(ctx)
		s.Require().NoError(err)
		s.Require().NoError(i.Execution().WaitInstanceIsRunning(ctx))

		wgetIndex, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
		s.Require().NoError(err)
		wgetIndex = strings.TrimSpace(wgetIndex)
		s.Assert().Equal("hello from 1", wgetIndex)

		webIP2 := webIP + "/index-2.html"
		wgetIndex2, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP2)
		s.Require().NoError(err)
		wgetIndex2 = strings.TrimSpace(wgetIndex2)
		s.Assert().Equal("hello from 2", wgetIndex2)
	}
}
