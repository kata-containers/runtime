// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"context"
	"fmt"
	"testing"
)

func TestMockHypervisorCreateSandbox(t *testing.T) {
	var m *mock
	sandboxID := "mock_sandbox"

	hypervisorConfig := Config{
		KernelPath:     "",
		ImagePath:      "",
		HypervisorPath: "",
	}

	ctx := context.Background()

	// wrong config
	if err := m.CreateSandbox(ctx, sandboxID, &hypervisorConfig, nil); err == nil {
		t.Fatal()
	}

	validHypervisorConfig := Config{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	if err := m.CreateSandbox(ctx, sandboxID, &validHypervisorConfig, nil); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorStartSandbox(t *testing.T) {
	var m *mock

	if err := m.StartSandbox(10); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorStopSandbox(t *testing.T) {
	var m *mock

	if err := m.StopSandbox(); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorAddDevice(t *testing.T) {
	var m *mock

	if err := m.AddDevice(nil, ImgDev); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorGetSandboxConsole(t *testing.T) {
	var m *mock

	expected := ""

	result, err := m.GetSandboxConsole("testSandboxID")
	if err != nil {
		t.Fatal(err)
	}

	if result != expected {
		t.Fatalf("Got %s\nExpecting %s", result, expected)
	}
}

func TestMockHypervisorSaveSandbox(t *testing.T) {
	var m *mock

	if err := m.SaveSandbox(); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorDisconnect(t *testing.T) {
	var m *mock

	m.Disconnect()
}
