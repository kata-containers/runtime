package api

import (
	"context"
	"syscall"

	"github.com/hyperhq/runv/api"
	ocispecs "github.com/opencontainers/runtime-spec/specs-go"
)

type ContainerConfig struct {
	Id string

	UGI    *UserGroupInfo
	Mounts map[string]Mount

	OciSpec ocispecs.Spec
}

type Mount struct {
	StorageID string
	Path      string
	ReadOnly  bool
}

type UsersAndGroups struct {
	User             string
	Group            string
	AdditionalGroups []string
}

type ProcessConfig struct {
	Container      string
	Id             string
	UsersAndGroups *UsersAndGroups

	OciProcess ocispecs.Process
}

//example implementation
//type Sandbox struct {
//	VM    VM
//	Agent Agent
//	Persistence Persistence // save it when every major operation
//
//	Devices map[string]AddedDevice
//
//	Containers map[string]Container
//}

type SandboxBuilder struct {
	VMBuilder
	AgentBuider
	Persistence
	netowrks map[string]Network
}

func SynthesizeSandbox(VM, Agent, Persistence, map[string]Network) *Sandbox {
}

func CreateSandbox(SandboxBuider, VMConfig, AgentConfig, SandboxNetworkConfig) *Sandbox {

}

// most sandbox api will call into vm api, and them agent api
type Sandbox interface {
	Start(b *BootConfig) (err error)
	Dump() json.RawMessage
	Release() error
	Shutdown() error
	Destroy() error
	// get notified when vm is shutdown/killed/destroyed.
	SandboxDone() context.Context

	UpdateRoute( /*todo*/ ) error
	AddNic( /*todo*/ ) error
	AllNics() []*InterfaceConfig
	DeleteNic(id string) error
	UpdateNic( /*todo*/ ) error
	SetCpus(cpus int) error
	SetMem(totalMem int) error
	AddStorage(s Storage) error
	RemoveStorage(name string) error

	AddContainer(c *ContainerConfig) error
	RemoveContainer(id string) error
	StartContainer(id string) error
	AddProcess(process *ProcessConfig) error
	WaitProcess(container, process string, ctx context.Context) (int, error)
	SignalProcess(container, process string, signal syscall.Signal) error

	TerminalResize(containerId, execId string, row, column int) error
	// called when non-shim(fratki/hyperd)
	StdioPipe(container, process string) (stdinPipe io.WriteCloser, stdout io.Reader, stderr io.Reader, err error)

	Stats() *SandboxStats
	ContainerList() []string
	Pause() error
	Resume() error
}
