// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	spec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli"
)

func TestNewPersistentNamespaces(t *testing.T) {
	assert := assert.New(t)
	var err error

	orgNamespacesDirMode := namespacesDirMode
	namespacesDirMode = os.FileMode(0777)
	orgNewPersistentNamespacesFunc := newPersistentNamespacesFunc
	newPersistentNamespacesFunc = func(sandboxNsPath string) int { return 0 }
	orgNamespacesPath := namespacesPath
	namespacesPath, err = ioutil.TempDir("", "ns")
	assert.NoError(err)
	mntNs := filepath.Join(namespacesPath, "mnt")
	defer func() {
		os.RemoveAll(namespacesPath)
		namespacesDirMode = orgNamespacesDirMode
		newPersistentNamespacesFunc = orgNewPersistentNamespacesFunc
		namespacesPath = orgNamespacesPath
	}()

	err = newPersistentNamespaces("", "", nil)
	assert.Error(err)

	newPersistentNamespacesFunc = func(sandboxNsPath string) int { return 1 }
	err = newPersistentNamespaces(testContainerID, testContainerID, nil)
	assert.NoError(err)
	os.RemoveAll(filepath.Join(namespacesPath, testContainerID))

	err = newPersistentNamespaces(testSandboxID, testContainerID, nil)
	assert.NoError(err)
	os.RemoveAll(filepath.Join(namespacesPath, testContainerID))
	os.RemoveAll(filepath.Join(namespacesPath, testSandboxID))

	namespaces := []spec.LinuxNamespace{}
	err = newPersistentNamespaces(testSandboxID, "", namespaces)
	assert.NoError(err)

	namespaces = []spec.LinuxNamespace{
		{spec.NetworkNamespace, ""},
		{spec.MountNamespace, mntNs},
	}
	err = newPersistentNamespaces(testSandboxID, "", namespaces)
	assert.NoError(err)

	newPersistentNamespacesFunc = func(sandboxNsPath string) int { return -1 }
	err = newPersistentNamespaces(testSandboxID, "", namespaces)
	assert.Error(err)
}

func TestJoinNamespaces(t *testing.T) {
	assert := assert.New(t)

	joinedNs, err := joinNamespaces("")
	assert.Error(err)
	assert.False(joinedNs)

	joinedNs, err = joinNamespaces(testContainerID)
	assert.NoError(err)
	assert.False(joinedNs)

	orgJoinNamespacesFunc := joinNamespacesFunc
	orgNamespacesPath := namespacesPath
	defer func() {
		joinNamespacesFunc = orgJoinNamespacesFunc
		namespacesPath = orgNamespacesPath
	}()

	namespacesPath, err = ioutil.TempDir("", "ns")
	assert.NoError(err)
	defer func() {
		os.RemoveAll(namespacesPath)
	}()
	err = os.MkdirAll(filepath.Join(namespacesPath, testContainerID), 0755)
	assert.NoError(err)

	joinNamespacesFunc = func(containerNsPath string) int { return 0 }
	joinedNs, err = joinNamespaces(testContainerID)
	e, ok := err.(*cli.ExitError)
	assert.IsType(*e, cli.ExitError{})
	assert.True(ok)
	assert.False(joinedNs)

	joinNamespacesFunc = func(containerNsPath string) int { return -1 }
	joinedNs, err = joinNamespaces(testContainerID)
	assert.Error(err)
	assert.False(joinedNs)

	joinNamespacesFunc = func(containerNsPath string) int { return 1 }
	joinedNs, err = joinNamespaces(testContainerID)
	assert.NoError(err)
	assert.True(joinedNs)
}

func TestRemovePersistentNamespaces(t *testing.T) {
	assert := assert.New(t)

	err := removePersistentNamespaces("", "")
	assert.Error(err)

	orgNamespacesPath := namespacesPath
	namespacesPath, err = ioutil.TempDir("", "ns")
	assert.NoError(err)
	defer func() {
		os.RemoveAll(namespacesPath)
		namespacesPath = orgNamespacesPath
	}()

	err = removePersistentNamespaces(testSandboxID, "")
	assert.NoError(err)

	orgRemovePersistentNamespacesFunc := removePersistentNamespacesFunc
	defer func() {
		removePersistentNamespacesFunc = orgRemovePersistentNamespacesFunc
	}()
	removePersistentNamespacesFunc = func(path string) int { return -1 }
	err = removePersistentNamespaces(testSandboxID, "")
	assert.NoError(err)
}
