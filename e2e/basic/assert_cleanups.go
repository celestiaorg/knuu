package basic

import (
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

// assertCleanupInstance is a helper function that cleans up a single instance.
func assertCleanupInstance(t *testing.T, instance *knuu.Instance) error {
	if instance == nil {
		t.Fatal("Instance is nil")
	}

	if err := instance.Destroy(); err != nil {
		t.Fatalf("Error destroying instance: %v", err)
	}
	return nil
}

// assertCleanupInstances is a helper function that cleans up a list of instances.
func assertCleanupInstances(t *testing.T, executor *knuu.Executor, instances []*knuu.Instance) error {
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
