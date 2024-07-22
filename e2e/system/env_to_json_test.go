package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/celestiaorg/knuu/pkg/instance"
)

func (s *Suite) TestEnvToJSON() {
	const namePrefix = "env-to-json"

	s.T().Parallel()

	// Setup
	executor, err := s.Knuu.NewInstance(namePrefix + "-executor")
	s.Require().NoError(err)

	const numberOfInstances = 2
	instances := make([]*instance.Instance, numberOfInstances)

	// Define the env vars
	testEnvVarKey1 := "TESTKEY1"
	testEnvVarKey2 := "TESTKEY2"
	testEnvVarKey3 := "TESTKEY3"
	testEnvVarValue1 := "testvalue1"
	testEnvVarValue2 := "testvalue2"
	testEnvVarValue3 := "testvalue3"

	// Set the OS env vars
	s.Require().NoError(os.Setenv(testEnvVarKey1, testEnvVarValue1))
	s.Require().NoError(os.Setenv(testEnvVarKey2, testEnvVarValue2))
	s.Require().NoError(os.Setenv(testEnvVarKey3, testEnvVarValue3))

	// Define helper function to get env vars
	getEnv := func(key string) string {
		value := os.Getenv(key)
		s.Require().NotEmpty(value, "getting env var '%v'", key)
		return value
	}
	jsonBytes, err := json.Marshal(map[string]string{
		testEnvVarKey1: getEnv(testEnvVarKey1),
		testEnvVarKey2: getEnv(testEnvVarKey2),
		testEnvVarKey3: getEnv(testEnvVarKey3),
	})
	s.Require().NoError(err)

	jsonString := string(jsonBytes)
	s.T().Logf("JSON: %v", jsonString)
	ctx := context.Background()

	for i := 0; i < numberOfInstances; i++ {
		instanceName := fmt.Sprintf("%s-web%d", namePrefix, i+1)
		ins, err := s.Knuu.NewInstance(instanceName)
		s.Require().NoError(err)

		s.Require().NoError(ins.SetImage(ctx, nginxImage))
		ins.AddPortTCP(80)
		_, err = ins.ExecuteCommand(ctx, "mkdir", "-p", nginxPath)
		s.Require().NoError(err)

		s.T().Logf("Writing JSON to instance '%v': %v", instanceName, jsonString)
		_, err = ins.ExecuteCommand(ctx, "mkdir", "-p", "/opt/env")
		s.Require().NoError(err)

		// write the json file to the instance
		_, err = ins.ExecuteCommand(ctx, "echo", fmt.Sprintf("'%s'", jsonString), ">", "/opt/env/env.json")
		s.Require().NoError(err)

		// for testing it, we also add it as index.html to the nginx server
		_, err = ins.ExecuteCommand(ctx, "echo", fmt.Sprintf("'%s'", jsonString), ">", "/usr/share/nginx/html/index.html")
		s.Require().NoError(err, "writing JSON to instance '%v': %v", instanceName, err)

		s.Require().NoError(ins.Commit())
		instances[i] = ins
	}

	s.T().Cleanup(func() {
		all := append(instances, executor)
		err := instance.BatchDestroy(ctx, all...)
		if err != nil {
			s.T().Logf("error destroying instances: %v", err)
		}
	})

	// Test logic
	for _, i := range instances {
		s.Require().NoError(i.StartAsync(ctx))
	}

	for _, i := range instances {
		webIP, err := i.GetIP(ctx)
		s.Require().NoError(err)

		err = i.WaitInstanceIsRunning(ctx)
		s.Require().NoError(err)

		wget, err := executor.ExecuteCommand(ctx, "wget", "-q", "-O", "-", webIP)
		s.Require().NoError(err)

		expected := fmt.Sprintf("{\"%s\":\"%s\",\"%s\":\"%s\",\"%s\":\"%s\"}\n", testEnvVarKey1, testEnvVarValue1, testEnvVarKey2, testEnvVarValue2, testEnvVarKey3, testEnvVarValue3)
		s.Assert().Equal(expected, wget)
	}
}
