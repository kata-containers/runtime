// Copyright (c) 2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package types

// DeviceType describes a virtualized device type.
type DeviceType int

const (
	// ImgDev is the image device type.
	ImgDev DeviceType = iota

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

	// VFIODev is VFIO device type
	VFIODev

	// vhostuserDev is a Vhost-user device type
	VhostuserDev

	// CPUDevice is CPU device type
	CPUDev

	// MemoryDevice is memory device type
	MemoryDev
)

// Device represents a virtcontainers Device
type Device struct {
	// Info is the device specific data
	Info interface{}

	// Type is the device type
	Type DeviceType
}

// MemoryDevice represents a memory slot.
type MemoryDevice struct {
	// Slot is the menory slot index
	Slot int

	// SizeMB is the slot size in MegaBytes.
	SizeMB int
}
