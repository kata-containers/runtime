// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"context"
	"os"

	"github.com/kata-containers/runtime/virtcontainers/hypervisor"
	"github.com/kata-containers/runtime/virtcontainers/store"
	"github.com/kata-containers/runtime/virtcontainers/types"
)

type mockHypervisor struct {
}

func (m *mockHypervisor) Capabilities() types.Capabilities {
	return types.Capabilities{}
}

func (m *mockHypervisor) Config() hypervisor.Config {
	return hypervisor.Config{}
}

func (m *mockHypervisor) CreateSandbox(ctx context.Context, id string, hypervisorConfig *hypervisor.Config, store *store.VCStore) error {
	err := hypervisorConfig.Valid()
	if err != nil {
		return err
	}

	return nil
}

func (m *mockHypervisor) StartSandbox(timeout int) error {
	return nil
}

func (m *mockHypervisor) StopSandbox() error {
	return nil
}

func (m *mockHypervisor) PauseSandbox() error {
	return nil
}

func (m *mockHypervisor) ResumeSandbox() error {
	return nil
}

func (m *mockHypervisor) SaveSandbox() error {
	return nil
}

func (m *mockHypervisor) AddDevice(devInfo interface{}, devType hypervisor.Device) error {
	return nil
}

func (m *mockHypervisor) HotplugAddDevice(devInfo interface{}, devType hypervisor.Device) (interface{}, error) {
	switch devType {
	case hypervisor.CPUDev:
		return devInfo.(uint32), nil
	case hypervisor.MemoryDev:
		memdev := devInfo.(*hypervisor.MemoryDevice)
		return memdev.SizeMB, nil
	}
	return nil, nil
}

func (m *mockHypervisor) HotplugRemoveDevice(devInfo interface{}, devType hypervisor.Device) (interface{}, error) {
	switch devType {
	case hypervisor.CPUDev:
		return devInfo.(uint32), nil
	case hypervisor.MemoryDev:
		return 0, nil
	}
	return nil, nil
}

func (m *mockHypervisor) GetSandboxConsole(sandboxID string) (string, error) {
	return "", nil
}

func (m *mockHypervisor) ResizeMemory(memMB uint32, memorySectionSizeMB uint32) (uint32, error) {
	return 0, nil
}
func (m *mockHypervisor) ResizeVCPUs(cpus uint32) (uint32, uint32, error) {
	return 0, 0, nil
}

func (m *mockHypervisor) Disconnect() {
}

func (m *mockHypervisor) GetThreadIDs() (*hypervisor.ThreadIDs, error) {
	vcpus := []int{os.Getpid()}
	return &hypervisor.ThreadIDs{vcpus}, nil
}

func (m *mockHypervisor) Cleanup() error {
	return nil
}
