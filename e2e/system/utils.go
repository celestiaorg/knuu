package system

import (
	"context"
	"fmt"
	"time"

	"github.com/celestiaorg/knuu/pkg/instance"
)

func retryOperation(operation func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, err)
}

func executeWget(ctx context.Context, executor *instance.Instance, url string) (string, error) {
	var output string
	err := retryOperation(func() error {
		var err error
		output, err = executor.ExecuteCommand(ctx, "wget", "-q", "-O", "-", url)
		return err
	}, 5)
	return output, err
}
