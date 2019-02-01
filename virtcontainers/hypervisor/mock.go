// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"context"
	"os"

	"github.com/kata-containers/runtime/virtcontainers/store"
	"github.com/kata-containers/runtime/virtcontainers/types"
)

type mock struct {
}

// NewMock returns a new mock hypervisor instance.
func NewMock() Hypervisor {
	return &mock{}
}

func (m *mock) Capabilities() types.Capabilities {
	return types.Capabilities{}
}

func (m *mock) Config() Config {
	return Config{}
}

func (m *mock) CreateSandbox(ctx context.Context, id string, hypervisorConfig *Config, store *store.VCStore) error {
	err := hypervisorConfig.Valid()
	if err != nil {
		return err
	}

	return nil
}

func (m *mock) StartSandbox(timeout int) error {
	return nil
}

func (m *mock) StopSandbox() error {
	return nil
}

func (m *mock) PauseSandbox() error {
	return nil
}

func (m *mock) ResumeSandbox() error {
	return nil
}

func (m *mock) SaveSandbox() error {
	return nil
}

func (m *mock) AddDevice(devInfo interface{}, devType Device) error {
	return nil
}

func (m *mock) HotplugAddDevice(devInfo interface{}, devType Device) (interface{}, error) {
	switch devType {
	case CPUDev:
		return devInfo.(uint32), nil
	case MemoryDev:
		memdev := devInfo.(*MemoryDevice)
		return memdev.SizeMB, nil
	}
	return nil, nil
}

func (m *mock) HotplugRemoveDevice(devInfo interface{}, devType Device) (interface{}, error) {
	switch devType {
	case CPUDev:
		return devInfo.(uint32), nil
	case MemoryDev:
		return 0, nil
	}
	return nil, nil
}

func (m *mock) GetSandboxConsole(sandboxID string) (string, error) {
	return "", nil
}

func (m *mock) ResizeMemory(memMB uint32, memorySectionSizeMB uint32) (uint32, error) {
	return 0, nil
}
func (m *mock) ResizeVCPUs(cpus uint32) (uint32, uint32, error) {
	return 0, 0, nil
}

func (m *mock) Disconnect() {
}

func (m *mock) GetThreadIDs() (*ThreadIDs, error) {
	vcpus := []int{os.Getpid()}
	return &ThreadIDs{vcpus}, nil
}

func (m *mock) Cleanup() error {
	return nil
}
