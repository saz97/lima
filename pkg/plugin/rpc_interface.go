package plugin

type VMDriver interface {
	// Start starts the VM.
	Start(instanceName string, Config []byte) error
}

type StartArgs struct {
	InstanceName string
	Config       []byte
}

type StopArgs struct {
	InstanceName string
}
