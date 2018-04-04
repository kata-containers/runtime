package api

import (
	"encoding/json"
	"syscall"
)

// Save changes that needs to be persistent. (save the config of inserted volume and used scsi id for example)
// Rename it to ResourceStorage?
type Persistence interface {
	// new resources [and lock the filelock]
	CreateSandbox(id string) error
	// store the sandbox (call sandbox.Dump() and save)
	StoreSandbox(*Sandbox) error
	// TODO more finegrain store operations

	// [lock the filelock and] build sandbox from existing running sandbox
	ObtainSandbox(id string, builder SandboxBuilder) (*Sandbox, error)
	// release sandbox [and unlock the filelock]
	ReleaseSandbox(*Sandbox) error

	// queries
	ListSandboxs() []SandboxStatus
	ListContainers() []ContainerStatus
	SandboxOfContainer(string) string
	// TODO more finegrain query/fetch operations
}
