package api

import (
	"encoding/json"
)

type VMConfig struct {
	CPU    int
	Memory int
	Kernel string
	Initrd string // it can be nil, boot image should be added via AddDevice() in this case
	//Bios   string
	//Cbfs   string

	CPUMax        int
	MemSlot       int
	MemMax        int
	MemPath       string
	MemPathShared bool

	// to be filled
}

type QosConfig struct {
	CgroupPath string

	// For network QoS (kilobytes/s)
	InboundAverage  string
	InboundPeak     string
	OutboundAverage string
	OutboundPeak    string

	// to be filled
}

type Device interface {
}

type VM interface {
	// start to a fine state: example hotplug is workable, vm console is workable.
	Start()
	//Shutdown()
	Destroy()
	Snapshot() (Snapshot, error)
	Dump() json.RawMessage
	Release() error

	Pause() error
	Resume() error

	//ConsoleUrl() string
	//Console() io.Reader
	Stats() (*VMStats, error)
	// update qos config,cpu/memory,...
	Update( /*todo*/ ) error

	Capabilities() Capabilities

	// add cpu/memory/storage/network and other devices.
	// if the vm is not started, Device might be added on boot instead of hutpluging
	AddDevice(Device) (kata.Device, error)
	RemoveDevice(Device) error

	SetCpu(cpu int)
	SetMemory(cpu int)
	// add storage.
	// The returned storage is use for agent api, and it is
	// not the same content of the original one.
	// exmple1: source path is different between host and vm.
	// exmple2: when a storage with driver=qcow2 is added, the dirver is
	// handled on the runtime instead of the agent. And the agent can use
	// the resulted device as normal block device (without dirver field).
	AddStorage(Storage) (Storage, error)
	RemoveStorage(Storage) error
	AddNIC(NetowrkDevice) (Interface, error)
	RemoveNIC(NetowrkDevice) error
}

type VMBuilder interface {
	Build(VMConfig, AgentConfig, QosConfig) (VM, error)
	BuildFromReleased(json.RawMessage) (VM, error)
	BuildFromSnapshot(Snapshot, VMConfig, AgentConfig, QosConfig) (VM, error)
	Capabilities() BuilderCapabilities
}
