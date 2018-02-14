package api

type InterfaceConfig struct {
	Id      string
	Name    string
	Lo      bool
	Bridge  string
	IP      string
	MAC     string
	MTU     uint64
	Gateway string
	TapName string
	Options string
}

type InterfaceDevice struct {
}

type InterfaceDescription struct {
}

type PortDescription struct {
	HostPort      int32
	ContainerPort int32
	Protocol      string
}

type NeighborNetworks struct {
	InternalNetworks []string
	ExternalNetworks []string
}

type SandboxNetworkConfig struct {
	Hostname   string
	Dns        []string
	Neighbors  *NeighborNetworks
	DnsOptions []string
	DnsSearch  []string
}

type Network interface {
	ID() string
	Add(InterfaceConfig) (InterfaceDvice, error)
	Remove(InterfaceDeive) error
}
