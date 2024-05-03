package basic

import (
	"os"
	"testing"

	"github.com/celestiaorg/knuu/pkg/knuu"
)

// assertCleanupInstance is a helper function that cleans up a single instance.
func assertCleanupInstance(t *testing.T, instance *knuu.Instance) error {
	if instance != nil {
		err := instance.Destroy()
		if err != nil {
			t.Fatalf("Error destroying instance: %v", err)
			return err
		}
	}
	return nil
}

// assertCleanupInstances is a helper function that cleans up a list of instances.
func assertCleanupInstances(t *testing.T, executor *knuu.Executor, instances []*knuu.Instance) error {
	if os.Getenv("KNUU_SKIP_CLEANUP") != "true" {
		err := executor.Destroy()
		if err != nil {
			t.Fatalf("Error destroying executor: %v", err)
			return err
		}

		for _, instance := range instances {
			if instance != nil {
				err := instance.Destroy()
				if err != nil {
					t.Fatalf("Error destroying instance: %v", err)
					return err
				}
			}
		}
	}
	return nil
}
