// Copyright (c) 2018 IBM
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"encoding/hex"
	"os"
	"runtime"

	govmmQemu "github.com/intel/govmm/qemu"
	deviceConfig "github.com/kata-containers/runtime/virtcontainers/device/config"
	"github.com/kata-containers/runtime/virtcontainers/utils"
	"github.com/sirupsen/logrus"
)

type qemuS390x struct {
	// inherit from qemuArchBase, overwrite methods if needed
	qemuArchBase
}

const defaultQemuPath = "/usr/bin/qemu-system-s390x"

const defaultQemuMachineType = QemuSseries

const defaultQemuMachineOptions = "accel=kvm,usb=off"

// Not used
const defaultPCBridgeBus = ""

const defaultMemMaxS390x = 33280 // need to check

var qemuPaths = map[string]string{
	QemuSseries: defaultQemuPath,
}

var kernelRootParams = []Param{}

var kernelParams = []Param{
	{"console", "ttysclp0"},
}

var supportedQemuMachines = []govmmQemu.Machine{
	{
		Type:    QemuSseries,
		Options: defaultQemuMachineOptions,
	},
}

// Logger returns a logrus logger appropriate for logging qemu messages
func (q *qemuS390x) Logger() *logrus.Entry {
	return virtLog.WithField("subsystem", "qemu")
}

// MaxQemuVCPUs returns the maximum number of vCPUs supported
func MaxQemuVCPUs() uint32 {
	return uint32(runtime.NumCPU())
}

func newQemuArch(config HypervisorConfig) qemuArch {
	machineType := config.HypervisorMachineType
	if machineType == "" {
		machineType = defaultQemuMachineType
	}

	q := &qemuS390x{
		qemuArchBase{
			machineType:           machineType,
			qemuPaths:             qemuPaths,
			supportedQemuMachines: supportedQemuMachines,
			kernelParamsNonDebug:  kernelParamsNonDebug,
			kernelParamsDebug:     kernelParamsDebug,
			kernelParams:          kernelParams,
		},
	}

	q.handleImagePath(config)
	return q
}

func (q *qemuS390x) capabilities() capabilities {
	var caps capabilities

	// pseries machine type supports hotplugging drives
	if q.machineType == QemuPseries {
		caps.setBlockDeviceHotplugSupport()
	}

	return caps
}

func (q *qemuS390x) bridges(number uint32) []Bridge {
	return genericBridges(number, q.machineType)
}

func (q *qemuS390x) cpuModel() string {
	cpuModel := defaultCPUModel
	if q.nestedRun {
		cpuModel += ",pmu=off"
	}
	return cpuModel
}

func (q *qemuS390x) memoryTopology(memoryMb, hostMemoryMb uint64) govmmQemu.Memory {
	return genericMemoryTopology(memoryMb, hostMemoryMb)
}

func (q *qemuS390x) appendImage(devices []govmmQemu.Device, path string) ([]govmmQemu.Device, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}

	randBytes, err := utils.GenerateRandomBytes(8)
	if err != nil {
		return nil, err
	}

	id := utils.MakeNameID("image", hex.EncodeToString(randBytes), maxDevIDSize)

	drive := deviceConfig.BlockDrive{
		File:   path,
		Format: "raw",
		ID:     id,
	}

	return q.appendBlockDevice(devices, drive), nil
}

func (q *qemuS390x) appendBlockDevice(devices []govmmQemu.Device, drive deviceConfig.BlockDrive) []govmmQemu.Device {
	if drive.File == "" || drive.ID == "" || drive.Format == "" {
		return devices
	}

	if len(drive.ID) > maxDevIDSize {
		drive.ID = drive.ID[:maxDevIDSize]
	}

	devices = append(devices,
		govmmQemu.BlockDevice{
			Driver:        govmmQemu.VirtioBlock,
			ID:            drive.ID,
			File:          drive.File,
			AIO:           govmmQemu.Threads,
			Format:        govmmQemu.BlockDeviceFormat(drive.Format),
			Interface:     "none",
			DisableModern: q.nestedRun,
		},
	)

	return devices
}

func (q *qemuS390x) appendBridges(devices []govmmQemu.Device, bridges []Bridge) []govmmQemu.Device {
	return genericAppendBridges(devices, bridges, q.machineType)
}
