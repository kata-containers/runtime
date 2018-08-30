// Copyright (c) 2018 IBM
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"fmt"
	"testing"

	govmmQemu "github.com/intel/govmm/qemu"
	"github.com/stretchr/testify/assert"
)

func newTestQemu(machineType string) qemuArch {
	config := HypervisorConfig{
		HypervisorMachineType: machineType,
	}
	return newQemuArch(config)
}

func TestQemuS390xCPUModel(t *testing.T) {
	assert := assert.New(t)
	s390x := newTestQemu(QemuSseries)

	expectedOut := defaultCPUModel
	model := s390x.cpuModel()
	assert.Equal(expectedOut, model)

	s390x.enableNestingChecks()
	expectedOut = defaultCPUModel + ",pmu=off"
	model = s390x.cpuModel()
	assert.Equal(expectedOut, model)
}

func TestQemuS390xMemoryTopology(t *testing.T) {
	assert := assert.New(t)
	s390x := newTestQemu(QemuSseries)
	memoryOffset := 1024

	hostMem := uint64(1024)
	mem := uint64(120)
	expectedMemory := govmmQemu.Memory{
		Size:   fmt.Sprintf("%dM", mem),
		Slots:  defaultMemSlots,
		MaxMem: fmt.Sprintf("%dM", hostMem+uint64(memoryOffset)),
	}

	m := s390x.memoryTopology(mem, hostMem)
	assert.Equal(expectedMemory, m)
}

