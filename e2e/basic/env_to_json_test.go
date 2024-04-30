package basic

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

func TestEnvToJSON(t *testing.T) {
	t.Parallel()
	// Setup

	executor, err := knuu.NewExecutor()
	if err != nil {
		t.Fatalf("Error creating executor: %v", err)
	}

	const numberOfInstances = 2
	instances := make([]*knuu.Instance, numberOfInstances)

	// get the values from the .env
	envVars, err := godotenv.Read()
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}
	jsonBytes, err := json.Marshal(envVars)
	if err != nil {
		t.Fatalf("Error converting env vars to JSON: %v", err)
	}
	jsonString := string(jsonBytes)
	logrus.Debugf("JSON: %v", jsonString)

	for i := 0; i < numberOfInstances; i++ {
		instanceName := fmt.Sprintf("web%d", i+1)
		instance, err := knuu.NewInstance(instanceName)
		if err != nil {
			t.Fatalf("Error creating instance '%v': %v", instanceName, err)
		}
		err = instance.SetImage("docker.io/nginx:latest")
		if err != nil {
			t.Fatalf("Error setting image for '%v': %v", instanceName, err)
		}
		instance.AddPortTCP(80)
		_, err = instance.ExecuteCommand("mkdir", "-p", "/usr/share/nginx/html")
		if err != nil {
			t.Fatalf("Error executing command for '%v': %v", instanceName, err)
		}

		logrus.Debugf("Writing JSON to instance '%v': %v", instanceName, jsonString)
		_, err = instance.ExecuteCommand("mkdir", "-p", "/opt/env")
		if err != nil {
			t.Fatalf("Error writing JSON to instance '%v': %v", instanceName, err)
		}
		// write the json file to the instance
		_, err = instance.ExecuteCommand("echo", fmt.Sprintf("'%s'", jsonString), ">", "/opt/env/env.json")
		if err != nil {
			t.Fatalf("Error writing JSON to instance '%v': %v", instanceName, err)
		}
		// for testing it, we also add it as index.html to the nginx server
		_, err = instance.ExecuteCommand("echo", fmt.Sprintf("'%s'", jsonString), ">", "/usr/share/nginx/html/index.html")
		if err != nil {
			t.Fatalf("Error writing JSON to instance '%v': %v", instanceName, err)
		}

		err = instance.Commit()
		if err != nil {
			t.Fatalf("Error committing instance '%v': %v", instanceName, err)
		}

		instances[i] = instance
	}

	t.Cleanup(func() {
		// Cleanup
		if os.Getenv("KNUU_SKIP_CLEANUP") != "true" {
			err := executor.Destroy()
			if err != nil {
				t.Fatalf("Error destroying executor: %v", err)
			}

			for _, instance := range instances {
				if instance != nil {
					err := instance.Destroy()
					if err != nil {
						t.Fatalf("Error destroying instance: %v", err)
					}
				}
			}
		}
	})

	// Test logic
	for _, instance := range instances {
		err = instance.StartAsync()
		if err != nil {
			t.Fatalf("Error waiting for instance to be running: %v", err)
		}
	}

	for _, instance := range instances {
		webIP, err := instance.GetIP()
		if err != nil {
			t.Fatalf("Error getting IP: %v", err)
		}

		err = instance.WaitInstanceIsRunning()
		if err != nil {
			t.Fatalf("Error waiting for instance to be running: %v", err)
		}

		wget, err := executor.ExecuteCommand("wget", "-q", "-O", "-", webIP)
		if err != nil {
			t.Fatalf("Error executing command: %v", err)
		}

		assert.Equal(t, "{\"TEST\":\"test\",\"TEST_2\":\"test2\",\"TEST_3\":\"test3\"}\n", wget)
	}
}
