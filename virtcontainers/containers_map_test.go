// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package virtcontainers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindSliceIndexStrNilSlice(t *testing.T) {
	assert := assert.New(t)

	idx := findSliceIndexStr(nil, "")
	assert.Equal(idx, -1)
}

func TestFindSliceIndexStr(t *testing.T) {
	assert := assert.New(t)

	idx := findSliceIndexStr([]string{"val1", "val2"}, "val2")
	assert.Equal(idx, 1)
}

func TestFetchContainersMapNotExisting(t *testing.T) {
	assert := assert.New(t)
	expected := make(map[string][]string)

	ctrsMap, err := fetchContainersMap()
	assert.Nil(err)
	assert.True(reflect.DeepEqual(ctrsMap, expected), "Got %v\nExpecting %v", ctrsMap, expected)
}

func TestFetchContainersMap(t *testing.T) {
	assert := assert.New(t)

	ctrID := "12345"
	sandboxID := "67890"
	ctrsMap := make(map[string][]string)
	sandboxList := []string{sandboxID}
	ctrsMap[ctrID] = sandboxList

	path := filepath.Join(runStoragePath, containersMapFile)
	err := os.RemoveAll(path)
	assert.Nil(err)

	f, err := os.Create(path)
	assert.Nil(err)
	defer os.RemoveAll(path)
	defer f.Close()

	jsonOut, err := json.Marshal(ctrsMap)
	assert.Nil(err)

	_, err = f.Write(jsonOut)
	assert.Nil(err)

	result, err := fetchContainersMap()
	assert.Nil(err)
	assert.True(reflect.DeepEqual(result, ctrsMap), "Got %v\nExpecting %v", result, ctrsMap)
}

func TestAddToContainersMapNoCtrIDFailure(t *testing.T) {
	assert := assert.New(t)

	err := addToContainersMap("", "sandboxID")
	assert.Error(err)
	assert.Equal(err, errNeedContainerID)
}

func TestAddToContainersMapNoSandboxIDFailure(t *testing.T) {
	assert := assert.New(t)

	err := addToContainersMap("ctrID", "")
	assert.Error(err)
	assert.Equal(err, errNeedSandboxID)
}

func TestAddToContainersMapSuccessful(t *testing.T) {
	assert := assert.New(t)

	ctrID := "ctr1"
	sandboxID := "sb1"
	ctrsMap := make(map[string][]string)
	sandboxList := []string{sandboxID}
	ctrsMap[ctrID] = sandboxList

	additionalCtrID := "ctr2"
	additionalSandboxID := "sb2"

	expected := map[string][]string{
		ctrID:           {sandboxID},
		additionalCtrID: {additionalSandboxID},
	}

	path := filepath.Join(runStoragePath, containersMapFile)
	err := os.RemoveAll(path)
	assert.Nil(err)

	f, err := os.Create(path)
	assert.Nil(err)
	defer os.RemoveAll(path)
	defer f.Close()

	jsonOut, err := json.Marshal(ctrsMap)
	assert.Nil(err)

	_, err = f.Write(jsonOut)
	assert.Nil(err)

	err = addToContainersMap(additionalCtrID, additionalSandboxID)
	assert.Nil(err)
	result, err := fetchContainersMap()
	assert.Nil(err)
	assert.True(reflect.DeepEqual(result, expected), "Got %v\nExpecting %v", result, expected)
}

func TestDelFromContainersMapNoCtrIDFailure(t *testing.T) {
	assert := assert.New(t)

	err := delFromContainersMap("", "sandboxID")
	assert.Error(err)
	assert.Equal(err, errNeedContainerID)
}

func TestDelFromContainersMapNoSandboxIDFailure(t *testing.T) {
	assert := assert.New(t)

	err := delFromContainersMap("ctrID", "")
	assert.Error(err)
	assert.Equal(err, errNeedSandboxID)
}

func TestDelFromContainersMapSuccessful(t *testing.T) {
	assert := assert.New(t)

	ctrID := "ctr1"
	sandboxID := "sb1"
	additionalCtrID := "ctr2"
	additionalSandboxID := "sb2"

	ctrsMap := map[string][]string{
		ctrID:           {sandboxID},
		additionalCtrID: {additionalSandboxID},
	}

	expected := map[string][]string{
		ctrID: {sandboxID},
	}

	path := filepath.Join(runStoragePath, containersMapFile)
	err := os.RemoveAll(path)
	assert.Nil(err)

	f, err := os.Create(path)
	assert.Nil(err)
	defer os.RemoveAll(path)
	defer f.Close()

	jsonOut, err := json.Marshal(ctrsMap)
	assert.Nil(err)

	_, err = f.Write(jsonOut)
	assert.Nil(err)

	err = delFromContainersMap(additionalCtrID, additionalSandboxID)
	assert.Nil(err)
	result, err := fetchContainersMap()
	assert.Nil(err)
	assert.True(reflect.DeepEqual(result, expected), "Got %v\nExpecting %v", result, expected)
}
