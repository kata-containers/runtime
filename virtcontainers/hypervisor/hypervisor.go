// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/kata-containers/runtime/virtcontainers/device/config"
	"github.com/kata-containers/runtime/virtcontainers/store"
	"github.com/kata-containers/runtime/virtcontainers/types"
	"github.com/sirupsen/logrus"
)

var hLog = logrus.WithField("source", "virtcontainers/hypervisor")

// Type describes an hypervisor type.
type Type string

const (
	// Firecracker is the Firecracker hypervisor.
	Firecracker Type = "firecracker"

	// Qemu is the QEMU hypervisor.
	Qemu Type = "qemu"

	// Mock is a mock hypervisor for testing purposes
	Mock Type = "mock"
)

// Operation represents a hypervisor device operation
type Operation int

const (
	// AddDevice adds a device to a guest.
	AddDevice Operation = iota

	// RemoveDevice removes a device from a guest.
	RemoveDevice
)

const (
	// ProcMemInfo is the /proc/meminfo path
	ProcMemInfo = "/proc/meminfo"

	// ProcCPUInfo is the /proc/cpuinfo path
	ProcCPUInfo = "/proc/cpuinfo"
)

const (
	// DefaultVCPUs is the number of vCPUs a virtcontainers VM will run with by default.
	DefaultVCPUs = 1

	// DefaultMemSzMiB is the amount of memory a virtcontainers VM will run with by default.
	DefaultMemSzMiB = 2048

	// DefaultBridges is the number of PCI bridges a virtcontainers VM will run with by default.
	DefaultBridges = 1

	// DefaultBlockDriver is the default virtio block based driver.
	DefaultBlockDriver = config.VirtioSCSI

	// DefaultMsize9p is the default 9pfs msize value.
	DefaultMsize9p = 8192
)

// Device describes a virtualized device.
type Device int

const (
	// ImgDev is the image device type.
	ImgDev Device = iota

	// FsDev is the filesystem device type.
	FsDev

	// NetDev is the network device type.
	NetDev

	// SerialDev is the serial device type.
	SerialDev // nolint: varcheck,unused

	// BlockDev is the block device type.
	BlockDev

	// ConsoleDev is the console device type.
	ConsoleDev // nolint: varcheck,unused

	// SerialPortDev is the serial port device type.
	SerialPortDev

	// VSockPCIDev is the vhost vsock PCI device type.
	VSockPCIDev

	// VfioDev is VFIO device type
	VfioDev

	// VhostuserDev is a Vhost-user device type
	VhostuserDev

	// CPUDev is CPU device type
	CPUDev

	// MemoryDev is memory device type
	MemoryDev
)

// MemoryDevice represents a memory slot
type MemoryDevice struct {
	// Slot is the memory slot ID
	Slot int

	// SizeMB is the memory slot size in MBytes.
	SizeMB int
}

// Set sets an hypervisor type based on the input string.
func (t *Type) Set(value string) error {
	switch value {
	case "qemu":
		*t = Qemu
		return nil
	case "firecracker":
		*t = Firecracker
		return nil
	case "mock":
		*t = Mock
		return nil
	default:
		return fmt.Errorf("Unknown hypervisor type %s", value)
	}
}

// String converts an hypervisor type to a string.
func (t *Type) String() string {
	switch *t {
	case Qemu:
		return string(Qemu)
	case Firecracker:
		return string(Firecracker)
	case Mock:
		return string(Mock)
	default:
		return ""
	}
}

// New returns an hypervisor from and hypervisor type.
func New(t Type) (Hypervisor, error) {
	switch t {
	case Qemu:
		return &qemu{}, nil
	case Mock:
		return &mock{}, nil
	default:
		return nil, fmt.Errorf("Unknown hypervisor type %s", t)
	}
}

// Param is a key/value representation for hypervisor and kernel parameters.
type Param struct {
	Key   string
	Value string
}

// Config is the hypervisor configuration.
type Config struct {
	// NumVCPUs specifies default number of vCPUs for the VM.
	NumVCPUs uint32

	// DefaultMaxVCPUs specifies the maximum number of vCPUs for the VM.
	DefaultMaxVCPUs uint32

	// DefaultMem specifies default memory size in MiB for the VM.
	MemorySize uint32

	// DefaultBridges specifies default number of bridges for the VM.
	// Bridges can be used to hot plug devices
	DefaultBridges uint32

	// Msize9p is used as the msize for 9p shares
	Msize9p uint32

	// MemSlots specifies default memory slots the VM.
	MemSlots uint32

	// MemOffset specifies memory space for nvdimm device
	MemOffset uint32

	// KernelParams are additional guest kernel parameters.
	KernelParams []Param

	// HypervisorParams are additional hypervisor parameters.
	HypervisorParams []Param

	// KernelPath is the guest kernel host path.
	KernelPath string

	// ImagePath is the guest image host path.
	ImagePath string

	// InitrdPath is the guest initrd image host path.
	// ImagePath and InitrdPath cannot be set at the same time.
	InitrdPath string

	// FirmwarePath is the bios host path
	FirmwarePath string

	// MachineAccelerators are machine specific accelerators
	MachineAccelerators string

	// HypervisorPath is the hypervisor executable host path.
	HypervisorPath string

	// BlockDeviceDriver specifies the driver to be used for block device
	// either VirtioSCSI or VirtioBlock with the default driver being defaultBlockDriver
	BlockDeviceDriver string

	// HypervisorMachineType specifies the type of machine being
	// emulated.
	HypervisorMachineType string

	// MemoryPath is the memory file path of VM memory. Used when either BootToBeTemplate or
	// BootFromTemplate is true.
	MemoryPath string

	// DevicesStatePath is the VM device state file path. Used when either BootToBeTemplate or
	// BootFromTemplate is true.
	DevicesStatePath string

	// EntropySource is the path to a host source of
	// entropy (/dev/random, /dev/urandom or real hardware RNG device)
	EntropySource string

	// customAssets is a map of assets.
	// Each value in that map takes precedence over the configured assets.
	// For example, if there is a value for the "kernel" key in this map,
	// it will be used for the sandbox's kernel path instead of KernelPath.
	customAssets map[types.AssetType]*types.Asset

	// BlockDeviceCacheSet specifies cache-related options will be set to block devices or not.
	BlockDeviceCacheSet bool

	// BlockDeviceCacheDirect specifies cache-related options for block devices.
	// Denotes whether use of O_DIRECT (bypass the host page cache) is enabled.
	BlockDeviceCacheDirect bool

	// BlockDeviceCacheNoflush specifies cache-related options for block devices.
	// Denotes whether flush requests for the device are ignored.
	BlockDeviceCacheNoflush bool

	// DisableBlockDeviceUse disallows a block device from being used.
	DisableBlockDeviceUse bool

	// EnableIOThreads enables IO to be processed in a separate thread.
	// Supported currently for virtio-scsi driver.
	EnableIOThreads bool

	// Debug changes the default hypervisor and kernel parameters to
	// enable debug output where available.
	Debug bool

	// MemPrealloc specifies if the memory should be pre-allocated
	MemPrealloc bool

	// HugePages specifies if the memory should be pre-allocated from huge pages
	HugePages bool

	// Realtime Used to enable/disable realtime
	Realtime bool

	// Mlock is used to control memory locking when Realtime is enabled
	// Realtime=true and Mlock=false, allows for swapping out of VM memory
	// enabling higher density
	Mlock bool

	// DisableNestingChecks is used to override customizations performed
	// when running on top of another VMM.
	DisableNestingChecks bool

	// UseVSock use a vsock for agent communication
	UseVSock bool

	// HotplugVFIOOnRootBus is used to indicate if devices need to be hotplugged on the
	// root bus instead of a bridge.
	HotplugVFIOOnRootBus bool

	// BootToBeTemplate used to indicate if the VM is created to be a template VM
	BootToBeTemplate bool

	// BootFromTemplate used to indicate if the VM should be created from a template VM
	BootFromTemplate bool

	// DisableVhostNet is used to indicate if host supports vhost_net
	DisableVhostNet bool

	// GuestHookPath is the path within the VM that will be used for 'drop-in' hooks
	GuestHookPath string
}

// ThreadIDs represent a set of threads vCPU IDs.
type ThreadIDs struct {
	// VCPUs is a slice of vCPU IDs.
	VCPUs []int
}

func (conf *Config) checkTemplateConfig() error {
	if conf.BootToBeTemplate && conf.BootFromTemplate {
		return fmt.Errorf("Cannot set both 'to be' and 'from' vm tempate")
	}

	if conf.BootToBeTemplate || conf.BootFromTemplate {
		if conf.MemoryPath == "" {
			return fmt.Errorf("Missing MemoryPath for vm template")
		}

		if conf.BootFromTemplate && conf.DevicesStatePath == "" {
			return fmt.Errorf("Missing DevicesStatePath to load from vm template")
		}
	}

	return nil
}

// Valid checks if a hypervisor configuration is valid.
func (conf *Config) Valid() error {
	if conf.KernelPath == "" {
		return fmt.Errorf("Missing kernel path")
	}

	if conf.ImagePath == "" && conf.InitrdPath == "" {
		return fmt.Errorf("Missing image and initrd path")
	}

	if err := conf.checkTemplateConfig(); err != nil {
		return err
	}

	if conf.NumVCPUs == 0 {
		conf.NumVCPUs = DefaultVCPUs
	}

	if conf.MemorySize == 0 {
		conf.MemorySize = DefaultMemSzMiB
	}

	if conf.DefaultBridges == 0 {
		conf.DefaultBridges = DefaultBridges
	}

	if conf.BlockDeviceDriver == "" {
		conf.BlockDeviceDriver = DefaultBlockDriver
	}

	if conf.Msize9p == 0 {
		conf.Msize9p = DefaultMsize9p
	}

	return nil
}

// AddKernelParam allows the addition of new kernel parameters to an existing
// hypervisor configuration.
func (conf *Config) AddKernelParam(p Param) error {
	if p.Key == "" {
		return fmt.Errorf("Empty kernel parameter")
	}

	conf.KernelParams = append(conf.KernelParams, p)

	return nil
}

// AddCustomAsset adds a custom asset to a hypervisor configuration
func (conf *Config) AddCustomAsset(a *types.Asset) error {
	if a == nil || a.Path() == "" {
		// We did not get a custom asset, we will use the default one.
		return nil
	}

	if !a.Valid() {
		return fmt.Errorf("Invalid %s at %s", a.Type(), a.Path())
	}

	hLog.Debugf("Using custom %v asset %s", a.Type(), a.Path())

	if conf.customAssets == nil {
		conf.customAssets = make(map[types.AssetType]*types.Asset)
	}

	conf.customAssets[a.Type()] = a

	return nil
}

func (conf *Config) assetPath(t types.AssetType) (string, error) {
	// Custom assets take precedence over the configured ones
	a, ok := conf.customAssets[t]
	if ok {
		return a.Path(), nil
	}

	// We could not find a custom asset for the given type, let's
	// fall back to the configured ones.
	switch t {
	case types.KernelAsset:
		return conf.KernelPath, nil
	case types.ImageAsset:
		return conf.ImagePath, nil
	case types.InitrdAsset:
		return conf.InitrdPath, nil
	case types.HypervisorAsset:
		return conf.HypervisorPath, nil
	case types.FirmwareAsset:
		return conf.FirmwarePath, nil
	default:
		return "", fmt.Errorf("Unknown asset type %v", t)
	}
}

func (conf *Config) isCustomAsset(t types.AssetType) bool {
	_, ok := conf.customAssets[t]
	if ok {
		return true
	}

	return false
}

// KernelAssetPath returns the guest kernel path
func (conf *Config) KernelAssetPath() (string, error) {
	return conf.assetPath(types.KernelAsset)
}

// CustomKernelAsset returns true if the kernel asset is a custom one, false otherwise.
func (conf *Config) CustomKernelAsset() bool {
	return conf.isCustomAsset(types.KernelAsset)
}

// ImageAssetPath returns the guest image path
func (conf *Config) ImageAssetPath() (string, error) {
	return conf.assetPath(types.ImageAsset)
}

// CustomImageAsset returns true if the image asset is a custom one, false otherwise.
func (conf *Config) CustomImageAsset() bool {
	return conf.isCustomAsset(types.ImageAsset)
}

// InitrdAssetPath returns the guest initrd path
func (conf *Config) InitrdAssetPath() (string, error) {
	return conf.assetPath(types.InitrdAsset)
}

// CustomInitrdAsset returns true if the initrd asset is a custom one, false otherwise.
func (conf *Config) CustomInitrdAsset() bool {
	return conf.isCustomAsset(types.InitrdAsset)
}

// HypervisorAssetPath returns the VM hypervisor path
func (conf *Config) HypervisorAssetPath() (string, error) {
	return conf.assetPath(types.HypervisorAsset)
}

// CustomHypervisorAsset returns true if the hypervisor asset is a custom one, false otherwise.
func (conf *Config) CustomHypervisorAsset() bool {
	return conf.isCustomAsset(types.HypervisorAsset)
}

// FirmwareAssetPath returns the guest firmware path
func (conf *Config) FirmwareAssetPath() (string, error) {
	return conf.assetPath(types.FirmwareAsset)
}

// CustomFirmwareAsset returns true if the firmware asset is a custom one, false otherwise.
func (conf *Config) CustomFirmwareAsset() bool {
	return conf.isCustomAsset(types.FirmwareAsset)
}

func appendParam(params []Param, parameter string, value string) []Param {
	return append(params, Param{parameter, value})
}

// SerializeParams converts []Param to []string
func SerializeParams(params []Param, delim string) []string {
	var parameters []string

	for _, p := range params {
		if p.Key == "" && p.Value == "" {
			continue
		} else if p.Key == "" {
			parameters = append(parameters, fmt.Sprintf("%s", p.Value))
		} else if p.Value == "" {
			parameters = append(parameters, fmt.Sprintf("%s", p.Key))
		} else if delim == "" {
			parameters = append(parameters, fmt.Sprintf("%s", p.Key))
			parameters = append(parameters, fmt.Sprintf("%s", p.Value))
		} else {
			parameters = append(parameters, fmt.Sprintf("%s%s%s", p.Key, delim, p.Value))
		}
	}

	return parameters
}

// DeserializeParams converts []string to []Param
func DeserializeParams(parameters []string) []Param {
	var params []Param

	for _, param := range parameters {
		if param == "" {
			continue
		}
		p := strings.SplitN(param, "=", 2)
		if len(p) == 2 {
			params = append(params, Param{Key: p[0], Value: p[1]})
		} else {
			params = append(params, Param{Key: p[0], Value: ""})
		}
	}

	return params
}

// GetHostMemorySizeKb return the host memory size in KBytes.
func GetHostMemorySizeKb(memInfoPath string) (uint64, error) {
	f, err := os.Open(memInfoPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Expected format: ["MemTotal:", "1234", "kB"]
		parts := strings.Fields(scanner.Text())

		// Sanity checks: Skip malformed entries.
		if len(parts) < 3 || parts[0] != "MemTotal:" || parts[2] != "kB" {
			continue
		}

		sizeKb, err := strconv.ParseUint(parts[1], 0, 64)
		if err != nil {
			continue
		}

		return sizeKb, nil
	}

	// Handle errors that may have occurred during the reading of the file.
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return 0, fmt.Errorf("unable get MemTotal from %s", memInfoPath)
}

// RunningOnVMM checks if the system is running inside a VM.
func RunningOnVMM(cpuInfoPath string) (bool, error) {
	if runtime.GOARCH == "arm64" || runtime.GOARCH == "ppc64le" || runtime.GOARCH == "s390x" {
		hLog.Info("Unable to know if the system is running inside a VM")
		return false, nil
	}

	flagsField := "flags"

	f, err := os.Open(cpuInfoPath)
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Expected format: ["flags", ":", ...] or ["flags:", ...]
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}

		if !strings.HasPrefix(fields[0], flagsField) {
			continue
		}

		for _, field := range fields[1:] {
			if field == "hypervisor" {
				return true, nil
			}
		}

		// As long as we have been able to analyze the fields from
		// "flags", there is no reason to check what comes next from
		// /proc/cpuinfo, because we already know we are not running
		// on a VMM.
		return false, nil
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, fmt.Errorf("Couldn't find %q from %q output", flagsField, cpuInfoPath)
}

// Hypervisor is the virtcontainers hypervisor interface.
// The default hypervisor implementation is Qemu.
type Hypervisor interface {
	CreateSandbox(ctx context.Context, id string, hypervisorConfig *Config, store *store.VCStore) error
	StartSandbox(timeout int) error
	StopSandbox() error
	PauseSandbox() error
	SaveSandbox() error
	ResumeSandbox() error
	AddDevice(devInfo interface{}, dev Device) error
	HotplugAddDevice(devInfo interface{}, dev Device) (interface{}, error)
	HotplugRemoveDevice(devInfo interface{}, dev Device) (interface{}, error)
	ResizeMemory(memMB uint32, memoryBlockSizeMB uint32) (uint32, error)
	ResizeVCPUs(vcpus uint32) (uint32, uint32, error)
	GetSandboxConsole(sandboxID string) (string, error)
	Disconnect()
	Capabilities() types.Capabilities
	Config() Config
	GetThreadIDs() (*ThreadIDs, error)
	Cleanup() error
}
