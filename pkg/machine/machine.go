package machine

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/inlets/cloud-provision/provision"
)

// Machine represents a machine with its provisioned host details
type Machine struct {
	logger      *log.Logger
	provisioner provision.Provisioner
	host        *provision.ProvisionedHost
	region      string
	size        string
	name        string
}

// NewMachine creates a new machine
func NewMachine(logger *log.Logger, provisioner provision.Provisioner, region, size, name, machineOS string, machineUserData []string) (*Machine, error) {
	userData := strings.Join(machineUserData, "\n")
	userData = strings.ReplaceAll(userData, "%POOL_ID%", os.Getenv("POOL_ID"))
	userData = strings.ReplaceAll(userData, "%SCW_SECRET_KEY%", os.Getenv("SCW_SECRET_KEY"))
	res, err := provisioner.Provision(provision.BasicHost{
		Name:       name,
		OS:         machineOS,
		Plan:       size,
		Region:     string(region),
		UserData:   userData,
		Additional: map[string]string{},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to provision host: %w", err)
	}

	logger.Printf("Machine created: %s\n", res.ID)

	machine := &Machine{
		logger:      logger,
		provisioner: provisioner,
		host:        res,
		region:      region,
		size:        size,
		name:        name,
	}

	return machine, nil
}

// Remove removes a machine
func (machine *Machine) Remove(ctx context.Context) error {
	if machine.host == nil {
		return fmt.Errorf("host is not provisioned")
	}

	// Delete the hardware node via the provisioner
	err := machine.provisioner.Delete(provision.HostDeleteRequest{ID: machine.host.ID})
	if err != nil {
		return fmt.Errorf("failed to delete host: %w", err)
	}
	machine.logger.Printf("Machine deleted: %s\n", machine.host.ID)

	return nil
}

func (machine *Machine) Setup(ctx context.Context) error {
	return nil
}

// WaitForCreation blocks until the instance is created
func (machine *Machine) WaitForCreation() error {
	if machine.host == nil {
		return fmt.Errorf("host is not provisioned for machine %s", machine.name)
	}
	pollStatusAttempts := 250
	waitInterval := time.Second * 2
	for i := 0; i <= pollStatusAttempts; i++ {
		machine.logger.Printf("Machine %s: Polling status attempt %d of %d", machine.name, i+1, pollStatusAttempts)
		res, err := machine.provisioner.Status(machine.host.ID)

		if err != nil {
			return fmt.Errorf("failed to get status for Machine %s: %w", machine.name, err)
		}
		if res.Status == provision.ActiveStatus {
			machine.host = res
			machine.logger.Printf("Machine %s: Machine created with ID %s", machine.name, res.ID)
			return nil
		}
		time.Sleep(waitInterval)
	}

	return fmt.Errorf("timeout waiting for instance creation for Machine %s", machine.name)
}

// GetName returns the name of the machine
func (machine *Machine) GetName() string {
	return machine.name
}

// GetIP returns the IP address of the machine
func (machine *Machine) GetIP() string {
	if machine.host != nil {
		return machine.host.IP
	}
	return ""
}
