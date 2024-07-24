package system

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/pkg/instance"
)

func (s *Suite) TestFolder() {
	const namePrefix = "folder"
	s.T().Parallel()

	// Setup
	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix)
	require.NoError(s.T(), err)

	web := s.createNginxInstanceWithVolume(ctx, namePrefix)
	err = web.AddFolder(resourcesHTML, nginxHTMLPath, "0:0")
	require.NoError(s.T(), err)

	require.NoError(s.T(), web.Commit())

	s.T().Cleanup(func() {
		err := instance.BatchDestroy(ctx, web, executor)
		if err != nil {
			s.T().Logf("Error destroying instance: %v", err)
		}
	})

	// Test logic
	webIP, err := web.GetIP(ctx)
	s.Require().NoError(err)

	s.Require().NoError(web.Start(ctx))

	wget, err := executor.ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	s.Require().NoError(err)

	s.Assert().Contains(wget, "Hello World!")
}
