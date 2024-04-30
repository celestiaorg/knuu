package basic

import (
	"github.com/celestiaorg/knuu/pkg/knuu"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/rbac/v1"
	"os"
	"testing"
)

func TestRBAC(t *testing.T) {
	t.Parallel()
	// Setup

	instance, err := knuu.NewInstance("kubectl")
	if err != nil {
		t.Fatalf("Error creating instance '%v':", err)
	}
	err = instance.SetImage("docker.io/bitnami/kubectl:latest")
	if err != nil {
		t.Fatalf("Error setting image: %v", err)
	}
	err = instance.SetCommand("sleep", "infinity")
	if err != nil {
		t.Fatalf("Error setting command: %v", err)
	}
	err = instance.Commit()
	if err != nil {
		t.Fatalf("Error committing instance: %v", err)
	}
	policyRule := v1.PolicyRule{
		Verbs:     []string{"get", "list", "watch"},
		APIGroups: []string{""},
		Resources: []string{"pods"},
	}
	err = instance.AddPolicyRule(policyRule)
	if err != nil {
		t.Fatalf("Error adding policy rule: %v", err)
	}

	t.Cleanup(func() {
		// Cleanup
		if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
			t.Log("Skipping cleanup")
			return
		}

		err = instance.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
		}
	})

	// Test logic

	err = instance.Start()
	if err != nil {
		t.Fatalf("Error starting instance: %v", err)
	}
	err = instance.WaitInstanceIsRunning()
	if err != nil {
		t.Fatalf("Error waiting for instance to be running: %v", err)
	}
	_, err = instance.ExecuteCommand("kubectl", "get", "pods")
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}
	exitCode, err := instance.ExecuteCommand("echo", "$?")
	if err != nil {
		t.Fatalf("Error executing command '%v':", err)
	}

	assert.Equal(t, "0\n", exitCode)
}
