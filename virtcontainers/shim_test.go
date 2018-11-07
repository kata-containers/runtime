// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"os/exec"
	"reflect"
	"syscall"
	"testing"

	vshim "github.com/kata-containers/runtime/virtcontainers/shim"
)

const (
	testRunningProcess = "sleep"
)

func testSetShimType(t *testing.T, value string, expected vshim.Type) {
	var shimType vshim.Type

	err := (&shimType).Set(value)
	if err != nil {
		t.Fatal(err)
	}

	if shimType != expected {
		t.Fatalf("Got %s\nExpecting %s", shimType, expected)
	}
}

func TestSetCCShimType(t *testing.T) {
	testSetShimType(t, "ccShim", vshim.CCShimType)
}

func TestSetKataShimType(t *testing.T) {
	testSetShimType(t, "kataShim", vshim.KataShimType)
}

func TestSetNoopShimType(t *testing.T) {
	testSetShimType(t, "noopShim", vshim.NoopShimType)
}

func TestSetUnknownShimType(t *testing.T) {
	var shimType vshim.Type

	unknownType := "unknown"

	err := (&shimType).Set(unknownType)
	if err == nil {
		t.Fatalf("Should fail because %s type used", unknownType)
	}

	if shimType == vshim.CCShimType || shimType == vshim.NoopShimType {
		t.Fatalf("%s shim type was not expected", shimType)
	}
}

func testStringFromShimType(t *testing.T, shimType vshim.Type, expected string) {
	shimTypeStr := (&shimType).String()
	if shimTypeStr != expected {
		t.Fatalf("Got %s\nExpecting %s", shimTypeStr, expected)
	}
}

func TestStringFromCCShimType(t *testing.T) {
	shimType := vshim.CCShimType
	testStringFromShimType(t, shimType, "ccShim")
}

func TestStringFromKataShimType(t *testing.T) {
	shimType := vshim.KataShimType
	testStringFromShimType(t, shimType, "kataShim")
}

func TestStringFromNoopShimType(t *testing.T) {
	shimType := vshim.NoopShimType
	testStringFromShimType(t, shimType, "noopShim")
}

func TestStringFromKataBuiltInShimType(t *testing.T) {
	shimType := vshim.KataBuiltInShimType
	testStringFromShimType(t, shimType, "kataBuiltInShim")
}

func TestStringFromUnknownShimType(t *testing.T) {
	var shimType vshim.Type
	testStringFromShimType(t, shimType, "")
}

func testNewShimFromShimType(t *testing.T, shimType vshim.Type, expected vshim.Shim) {
	result, err := vshim.NewShim(shimType)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("Got %+v\nExpecting %+v", result, expected)
	}
}

func TestNewShimFromCCShimType(t *testing.T) {
	shimType := vshim.CCShimType
	expectedShim := &vshim.CcShim{}
	testNewShimFromShimType(t, shimType, expectedShim)
}

func TestNewShimFromKataShimType(t *testing.T) {
	shimType := vshim.KataShimType
	expectedShim := &vshim.KataShim{}
	testNewShimFromShimType(t, shimType, expectedShim)
}

func TestNewShimFromNoopShimType(t *testing.T) {
	shimType := vshim.NoopShimType
	expectedShim := &vshim.NoopShim{}
	testNewShimFromShimType(t, shimType, expectedShim)
}

func TestNewShimFromKataBuiltInShimType(t *testing.T) {
	shimType := vshim.KataBuiltInShimType
	expectedShim := &vshim.KataBuiltInShim{}
	testNewShimFromShimType(t, shimType, expectedShim)
}

func TestNewShimFromUnknownShimType(t *testing.T) {
	var shimType vshim.Type

	_, err := vshim.NewShim(shimType)
	if err != nil {
		t.Fatal(err)
	}
}

func testNewShimConfigFromSandboxConfig(t *testing.T, sandboxConfig SandboxConfig, expected interface{}) {
	result := vshim.NewShimConfig(sandboxConfig.ShimType, sandboxConfig.ShimConfig)

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("Got %+v\nExpecting %+v", result, expected)
	}
}

func TestNewShimConfigFromCCShimSandboxConfig(t *testing.T) {
	shimConfig := vshim.Config{}

	sandboxConfig := SandboxConfig{
		ShimType:   vshim.CCShimType,
		ShimConfig: shimConfig,
	}

	testNewShimConfigFromSandboxConfig(t, sandboxConfig, shimConfig)
}

func TestNewShimConfigFromKataShimSandboxConfig(t *testing.T) {
	shimConfig := vshim.Config{}

	sandboxConfig := SandboxConfig{
		ShimType:   vshim.KataShimType,
		ShimConfig: shimConfig,
	}

	testNewShimConfigFromSandboxConfig(t, sandboxConfig, shimConfig)
}

func TestNewShimConfigFromNoopShimSandboxConfig(t *testing.T) {
	sandboxConfig := SandboxConfig{
		ShimType: vshim.NoopShimType,
	}

	testNewShimConfigFromSandboxConfig(t, sandboxConfig, nil)
}

func TestNewShimConfigFromKataBuiltInShimSandboxConfig(t *testing.T) {
	sandboxConfig := SandboxConfig{
		ShimType: vshim.KataBuiltInShimType,
	}

	testNewShimConfigFromSandboxConfig(t, sandboxConfig, nil)
}

func TestNewShimConfigFromUnknownShimSandboxConfig(t *testing.T) {
	var shimType vshim.Type

	sandboxConfig := SandboxConfig{
		ShimType: shimType,
	}

	testNewShimConfigFromSandboxConfig(t, sandboxConfig, nil)
}

func testRunSleep0AndGetPid(t *testing.T) int {
	binPath, err := exec.LookPath(testRunningProcess)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "0")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	pid := cmd.Process.Pid

	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	return pid
}

func testRunSleep999AndGetCmd(t *testing.T) *exec.Cmd {
	binPath, err := exec.LookPath(testRunningProcess)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(binPath, "999")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	return cmd
}

func TestStopShimSuccessfulProcessNotRunning(t *testing.T) {
	pid := testRunSleep0AndGetPid(t)

	if err := vshim.StopShim(pid); err != nil {
		t.Fatal(err)
	}
}

func TestStopShimSuccessfulProcessRunning(t *testing.T) {
	cmd := testRunSleep999AndGetCmd(t)

	if err := vshim.StopShim(cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
}

func testIsShimRunning(t *testing.T, pid int, expected bool) {
	running, err := vshim.IsShimRunning(pid)
	if err != nil {
		t.Fatal(err)
	}

	if running != expected {
		t.Fatalf("Expecting running=%t, Got running=%t", expected, running)
	}
}

func TestIsShimRunningFalse(t *testing.T) {
	pid := testRunSleep0AndGetPid(t)

	testIsShimRunning(t, pid, false)
}

func TestIsShimRunningTrue(t *testing.T) {
	cmd := testRunSleep999AndGetCmd(t)

	testIsShimRunning(t, cmd.Process.Pid, true)

	if err := syscall.Kill(cmd.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForShimInvalidPidSuccessful(t *testing.T) {
	wrongValuesList := []int{0, -1, -100}

	for _, val := range wrongValuesList {
		if err := vshim.WaitForShim(val); err != nil {
			t.Fatal(err)
		}
	}
}

func TestWaitForShimNotRunningSuccessful(t *testing.T) {
	pid := testRunSleep0AndGetPid(t)

	if err := vshim.WaitForShim(pid); err != nil {
		t.Fatal(err)
	}
}

func TestWaitForShimRunningForTooLongFailure(t *testing.T) {
	cmd := testRunSleep999AndGetCmd(t)

	vshim.SetWaitForShimTimeout(0.1)
	if err := vshim.WaitForShim(cmd.Process.Pid); err == nil {
		t.Fatal(err)
	}

	if err := syscall.Kill(cmd.Process.Pid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
}
