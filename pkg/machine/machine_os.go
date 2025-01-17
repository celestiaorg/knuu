package machine

// CAUTION: With the current cloud-init, only Ubuntu is supported

// DigitalOceanMachineOS represents the available OS in DigitalOcean
type DigitalOceanMachineOS struct {
	Ubuntu2404 string
}

// ScalewayMachineOS represents the available OS in Scaleway
type ScalewayMachineOS struct {
	Ubuntu2404 string
}

// CloudMachineOS represents the available OS in various cloud providers
type CloudMachineOS struct {
	DigitalOcean DigitalOceanMachineOS
	Scaleway     ScalewayMachineOS
}

var OS = CloudMachineOS{
	DigitalOcean: DigitalOceanMachineOS{
		Ubuntu2404: "ubuntu-24-04-x64",
	},
	Scaleway: ScalewayMachineOS{
		Ubuntu2404: "ubuntu_noble",
	},
}
