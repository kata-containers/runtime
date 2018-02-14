package api

import (
	"encoding/json"
	"syscall"

	ocispecs "github.com/opencontainers/runtime-spec/specs-go"
)

type AgentConfig struct {
	// ConnProto: the method to connect to agent(or proxy)
	// "unix": the paths are unix socket
	// "vsock": use cid and port pair to connect to the agent (kata-only)
	// "kvmtool": the paths are kvmtool driver created pty slaver with escaping character
	ConnProto string
	// AgentProto: the agent protocol
	// "hyperstart": orginal hyperstart protocol
	// "kata": kata agent protocol
	// "kata-yamux": kata agent protocol over yamux
	AgentProto string

	HyperstartCtlPath, HyperstartStreamPath string
	KataPath                                string
	KataVsockCid, KataVsockPort             uint32
}

type Agent interface {
	Config() AgentConfig
	Dump() json.RawMessage
	Release() error
	Destroy()

	PauseSync() error
	Unpause() error

	CreateContainer(container string, user *UserGroupInfo, storages []*Storage, c *ocispecs.Spec) error
	StartContainer(container string) error
	ExecProcess(container, process string, user *UserGroupInfo, p *ocispecs.Process) error
	SignalProcess(container, process string, signal syscall.Signal) error
	WaitProcess(container, process string) int

	StdioPipe(container, process string) (stdinPipe io.WriteCloser, stdout io.Reader, stderr io.Reader, err error)
	TtyWinResize(container, process string, row, col uint16) error

	StartSandbox(sb *SandboxNetworkConfig, storages []*Storage) error
	DestroySandbox() error
	UpdateRoute(r []Route) error
	UpdateInterface(ifs []Interface) error
	//OnlineCpuMem() error
}

// proxies, shim should be implemented via AgentBuilder (and Agent.CreateContainer() Agent.ExecProcess())
type AgentBuilder interface {
	Build(AgentConfig) Agent
}
