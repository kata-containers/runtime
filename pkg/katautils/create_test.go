// Copyright (c) 2017 Intel Corporation
// Copyright (c) 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package katautils

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"

	vc "github.com/kata-containers/runtime/virtcontainers"
	"github.com/kata-containers/runtime/virtcontainers/pkg/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// return the value of the *last* param with the specified key
func findLastParam(key string, params []vc.Param) (string, error) {
	if key == "" {
		return "", errors.New("ERROR: need non-nil key")
	}

	l := len(params)
	if l == 0 {
		return "", errors.New("ERROR: no params")
	}

	for i := l - 1; i >= 0; i-- {
		p := params[i]

		if key == p.Key {
			return p.Value, nil
		}
	}

	return "", fmt.Errorf("no param called %q found", name)
}

func TestSetEphemeralStorageType(t *testing.T) {
	assert := assert.New(t)

	ociSpec := oci.CompatOCISpec{}
	var ociMounts []specs.Mount
	mount := specs.Mount{
		Source: "/var/lib/kubelet/pods/366c3a77-4869-11e8-b479-507b9ddd5ce4/volumes/kubernetes.io~empty-dir/cache-volume",
	}

	ociMounts = append(ociMounts, mount)
	ociSpec.Mounts = ociMounts
	ociSpec = SetEphemeralStorageType(ociSpec)

	mountType := ociSpec.Mounts[0].Type
	assert.Equal(mountType, "ephemeral",
		"Unexpected mount type, got %s expected ephemeral", mountType)
}

func TestSetKernelParams(t *testing.T) {
	assert := assert.New(t)

	config := oci.RuntimeConfig{}

	assert.Empty(config.HypervisorConfig.KernelParams)

	err := SetKernelParams(testContainerID, &config)
	assert.NoError(err)

	if needSystemd(config.HypervisorConfig) {
		assert.NotEmpty(config.HypervisorConfig.KernelParams)
	}
}

func TestSetKernelParamsUserOptionTakesPriority(t *testing.T) {
	assert := assert.New(t)

	initName := "init"
	initValue := "/sbin/myinit"

	ipName := "ip"
	ipValue := "127.0.0.1"

	params := []vc.Param{
		{Key: initName, Value: initValue},
		{Key: ipName, Value: ipValue},
	}

	hypervisorConfig := vc.HypervisorConfig{
		KernelParams: params,
	}

	// Config containing user-specified kernel parameters
	config := oci.RuntimeConfig{
		HypervisorConfig: hypervisorConfig,
	}

	assert.NotEmpty(config.HypervisorConfig.KernelParams)

	err := SetKernelParams(testContainerID, &config)
	assert.NoError(err)

	kernelParams := config.HypervisorConfig.KernelParams

	init, err := findLastParam(initName, kernelParams)
	assert.NoError(err)
	assert.Equal(initValue, init)

	ip, err := findLastParam(ipName, kernelParams)
	assert.NoError(err)
	assert.Equal(ipValue, ip)

}
