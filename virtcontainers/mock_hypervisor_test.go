// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"context"
	"fmt"
	"testing"

	"github.com/kata-containers/runtime/virtcontainers/hypervisor"
)

func TestMockHypervisorCreateSandbox(t *testing.T) {
	var m *mockHypervisor

	sandbox := &Sandbox{
		config: &SandboxConfig{
			ID: "mock_sandbox",
			HypervisorConfig: hypervisor.Config{
				KernelPath:     "",
				ImagePath:      "",
				HypervisorPath: "",
			},
		},
	}

	ctx := context.Background()

	// wrong config
	if err := m.CreateSandbox(ctx, sandbox.config.ID, &sandbox.config.HypervisorConfig, nil); err == nil {
		t.Fatal()
	}

	sandbox.config.HypervisorConfig = hypervisor.Config{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	if err := m.CreateSandbox(ctx, sandbox.config.ID, &sandbox.config.HypervisorConfig, nil); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorStartSandbox(t *testing.T) {
	var m *mockHypervisor

	if err := m.StartSandbox(vmStartTimeout); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorStopSandbox(t *testing.T) {
	var m *mockHypervisor

	if err := m.StopSandbox(); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorAddDevice(t *testing.T) {
	var m *mockHypervisor

	if err := m.AddDevice(nil, hypervisor.ImgDev); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorGetSandboxConsole(t *testing.T) {
	var m *mockHypervisor

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
	var m *mockHypervisor

	if err := m.SaveSandbox(); err != nil {
		t.Fatal(err)
	}
}

func TestMockHypervisorDisconnect(t *testing.T) {
	var m *mockHypervisor

	m.Disconnect()
}
