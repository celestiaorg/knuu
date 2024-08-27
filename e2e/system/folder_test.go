package system

import (
	"context"

	"github.com/stretchr/testify/require"
)

func (s *Suite) TestFolder() {
	const namePrefix = "folder"

	// Setup
	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	require.NoError(s.T(), err)

	web := s.createNginxInstanceWithVolume(ctx, namePrefix+"-web")
	err = web.Storage().AddFolder(resourcesHTML, nginxHTMLPath, "0:0")
	require.NoError(s.T(), err)

	require.NoError(s.T(), web.Build().Commit(ctx))

	// Test logic
	webIP, err := web.Network().GetIP(ctx)
	s.Require().NoError(err)

	s.Require().NoError(web.Execution().Start(ctx))

	wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	s.Require().NoError(err)

	s.Assert().Contains(wget, "Hello World!")
}
