package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/celestiaorg/knuu/e2e"
	"github.com/celestiaorg/knuu/pkg/instance"
)

func (s *Suite) TestEnvToJSON() {
	const (
		namePrefix        = "env-to-json"
		numberOfInstances = 2
	)

	// Setup
	ctx := context.Background()
	executor, err := s.Executor.NewInstance(ctx, namePrefix+"-executor")
	s.Require().NoError(err)

	// Define the env vars
	envVars := map[string]string{
		"TESTKEY1": "testvalue1",
		"TESTKEY2": "testvalue2",
		"TESTKEY3": "testvalue3",
	}

	jsonBytes, err := json.Marshal(envVars)
	s.Require().NoError(err)

	jsonString := string(jsonBytes)
	s.T().Logf("JSON: %v", jsonString)

	instances := make([]*instance.Instance, numberOfInstances)
	for i := 0; i < numberOfInstances; i++ {
		name := fmt.Sprintf("%s-web%d", namePrefix, i+1)

		ins := s.CreateNginxInstance(ctx, name)
		s.Require().NoError(ins.Build().Commit(ctx))
		s.Require().NoError(ins.Execution().Start(ctx))

		_, err = ins.Execution().ExecuteCommand(ctx, "mkdir", "-p", e2e.NginxHTMLPath)
		s.Require().NoError(err)

		s.T().Logf("Writing JSON to instance '%v': %v", name, jsonString)
		_, err = ins.Execution().ExecuteCommand(ctx, "mkdir", "-p", "/opt/env")
		s.Require().NoError(err)

		// write the json file to the instance
		_, err = ins.Execution().ExecuteCommand(ctx, "echo", fmt.Sprintf("'%s'", jsonString), ">", "/opt/env/env.json")
		s.Require().NoError(err)

		// for testing it, we also add it as index.html to the nginx server
		_, err = ins.Execution().ExecuteCommand(ctx, "echo", fmt.Sprintf("'%s'", jsonString), ">", e2e.NginxHTMLPath+"/index.html")
		s.Require().NoError(err, "writing JSON to instance '%v': %v", name, err)

		instances[i] = ins
	}

	// Test logic
	for _, i := range instances {
		wget, err := executor.Execution().ExecuteCommand(ctx, "wget", "-q", "-O", "-", i.Network().HostName())
		s.Require().NoError(err)

		expectedBytes, err := json.Marshal(envVars)
		s.Require().NoError(err)
		s.Assert().Equal(string(expectedBytes), strings.TrimSpace(wget))
	}
}
