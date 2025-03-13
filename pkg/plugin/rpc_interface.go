package plugin

type VMDriver interface {
	// Start starts the VM.
	Start(args StartArgs) (string, error)
}

type StartArgs struct {
	InstanceName string
	ConfigData   []byte
	InstanceDir  string
	SSHLocalPort int
	SSHAddress   string
}

type StopArgs struct {
	InstanceName string
}
