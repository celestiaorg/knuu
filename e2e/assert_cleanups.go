package e2e

import (
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

// AssertCleanupInstance is a helper function that cleans up a single instance.
func AssertCleanupInstance(t *testing.T, instance *knuu.Instance) error {
	if instance == nil {
		t.Fatal("Instance is nil")
	}

	if err := instance.Destroy(); err != nil {
		t.Fatalf("Error destroying instance: %v", err)
	}
	return nil
}

// AssertCleanupInstances is a helper function that cleans up a list of instances.
func AssertCleanupInstances(t *testing.T, executor *knuu.Executor, instances []*knuu.Instance) error {
	if os.Getenv("KNUU_SKIP_CLEANUP") == "true" {
		t.Log("Skipping cleanup")
		return nil
	}

	if executor == nil {
		t.Fatal("Executor is nil")
	}

	if err := executor.Destroy(); err != nil {
		t.Fatalf("Error destroying executor: %v", err)
	}

	err := knuu.BatchDestroy(instances...)
	if err != nil {
		t.Fatalf("Error destroying instances: %v", err)
	}

	return nil
}
