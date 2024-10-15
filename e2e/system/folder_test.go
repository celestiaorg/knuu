package system

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/knuu/e2e"
)

func (s *Suite) TestFolder() {
	const namePrefix = "folder"

	// Setup
	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	require.NoError(s.T(), err)

	web := s.CreateNginxInstanceWithVolume(ctx, namePrefix+"-web")
	err = web.Storage().AddFolder(resourcesHTML, e2e.NginxHTMLPath, "0:0")
	require.NoError(s.T(), err)

	require.NoError(s.T(), web.Build().Commit(ctx))
	s.Require().NoError(web.Execution().Start(ctx))

	// Test logic
	webIP, err := web.Network().GetIP(ctx)
	s.Require().NoError(err)

	wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
	s.Require().NoError(err)

	s.Assert().Contains(wget, "Hello World!")
}
