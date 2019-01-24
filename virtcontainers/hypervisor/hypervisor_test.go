// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testKernel = "kernel"
const testInitrd = "initrd"
const testImage = "image"
const testHypervisor = "hypervisor"

var testDir = ""

const testDisabledAsNonRoot = "Test disabled as requires root privileges"

func testSetType(t *testing.T, value string, expected Type) {
	var hypervisorType Type

	err := (&hypervisorType).Set(value)
	if err != nil {
		t.Fatal(err)
	}

	if hypervisorType != expected {
		t.Fatal()
	}
}

func TestSetQemuType(t *testing.T) {
	testSetType(t, "qemu", Qemu)
}

func TestSetMockType(t *testing.T) {
	testSetType(t, "mock", Mock)
}

func TestSetUnknownType(t *testing.T) {
	var hypervisorType Type

	err := (&hypervisorType).Set("unknown")
	if err == nil {
		t.Fatal()
	}

	if hypervisorType == Qemu ||
		hypervisorType == Mock ||
		hypervisorType == Firecracker {
		t.Fatal()
	}
}

func testStringFromType(t *testing.T, hypervisorType Type, expected string) {
	hypervisorTypeStr := (&hypervisorType).String()
	if hypervisorTypeStr != expected {
		t.Fatal()
	}
}

func TestStringFromQemuType(t *testing.T) {
	hypervisorType := Qemu
	testStringFromType(t, hypervisorType, "qemu")
}

func TestStringFromMockType(t *testing.T) {
	hypervisorType := Mock
	testStringFromType(t, hypervisorType, "mock")
}

func TestStringFromUnknownType(t *testing.T) {
	var hypervisorType Type
	testStringFromType(t, hypervisorType, "")
}

func testNewHypervisorFromType(t *testing.T, hypervisorType Type, expected Hypervisor) {
	hy, err := New(hypervisorType)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(hy, expected) == false {
		t.Fatal()
	}
}

// func TestNewHypervisorFromQemuType(t *testing.T) {
// 	hypervisorType := Qemu
// 	expectedHypervisor := &qemu{}
// 	testNewHypervisorFromType(t, hypervisorType, expectedHypervisor)
// }

// func TestNewHypervisorFromMockType(t *testing.T) {
// 	hypervisorType := Mock
// 	expectedHypervisor := &mockHypervisor{}
// 	testNewHypervisorFromType(t, hypervisorType, expectedHypervisor)
// }

func TestNewHypervisorFromUnknownType(t *testing.T) {
	var hypervisorType Type

	hy, err := New(hypervisorType)
	if err == nil {
		t.Fatal()
	}

	if hy != nil {
		t.Fatal()
	}
}

func testConfigValid(t *testing.T, hypervisorConfig *Config, success bool) {
	err := hypervisorConfig.Valid()
	if success && err != nil {
		t.Fatal()
	}
	if !success && err == nil {
		t.Fatal()
	}
}

func TestConfigNoKernelPath(t *testing.T) {
	hypervisorConfig := &Config{
		KernelPath:     "",
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	testConfigValid(t, hypervisorConfig, false)
}

func TestConfigNoImagePath(t *testing.T) {
	hypervisorConfig := &Config{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      "",
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	testConfigValid(t, hypervisorConfig, false)
}

func TestConfigNoHypervisorPath(t *testing.T) {
	hypervisorConfig := &Config{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: "",
	}

	testConfigValid(t, hypervisorConfig, true)
}

func TestConfigIsValid(t *testing.T) {
	hypervisorConfig := &Config{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: fmt.Sprintf("%s/%s", testDir, testHypervisor),
	}

	testConfigValid(t, hypervisorConfig, true)
}

func TestConfigValidTemplateConfig(t *testing.T) {
	hypervisorConfig := &Config{
		KernelPath:       fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:        fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath:   fmt.Sprintf("%s/%s", testDir, testHypervisor),
		BootToBeTemplate: true,
		BootFromTemplate: true,
	}
	testConfigValid(t, hypervisorConfig, false)

	hypervisorConfig.BootToBeTemplate = false
	testConfigValid(t, hypervisorConfig, false)
	hypervisorConfig.MemoryPath = "foobar"
	testConfigValid(t, hypervisorConfig, false)
	hypervisorConfig.DevicesStatePath = "foobar"
	testConfigValid(t, hypervisorConfig, true)

	hypervisorConfig.BootFromTemplate = false
	hypervisorConfig.BootToBeTemplate = true
	testConfigValid(t, hypervisorConfig, true)
	hypervisorConfig.MemoryPath = ""
	testConfigValid(t, hypervisorConfig, false)
}

func TestConfigDefaults(t *testing.T) {
	hypervisorConfig := &Config{
		KernelPath:     fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:      fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath: "",
	}
	testConfigValid(t, hypervisorConfig, true)

	hypervisorConfigDefaultsExpected := &Config{
		KernelPath:        fmt.Sprintf("%s/%s", testDir, testKernel),
		ImagePath:         fmt.Sprintf("%s/%s", testDir, testImage),
		HypervisorPath:    "",
		NumVCPUs:          DefaultVCPUs,
		MemorySize:        DefaultMemSzMiB,
		DefaultBridges:    DefaultBridges,
		BlockDeviceDriver: DefaultBlockDriver,
		DefaultMaxVCPUs:   DefaultMaxQemuVCPUs,
		Msize9p:           DefaultMsize9p,
	}

	if reflect.DeepEqual(hypervisorConfig, hypervisorConfigDefaultsExpected) == false {
		t.Fatal()
	}
}

func TestAppendParams(t *testing.T) {
	paramList := []Param{
		{
			Key:   "param1",
			Value: "value1",
		},
	}

	expectedParams := []Param{
		{
			Key:   "param1",
			Value: "value1",
		},
		{
			Key:   "param2",
			Value: "value2",
		},
	}

	paramList = appendParam(paramList, "param2", "value2")
	if reflect.DeepEqual(paramList, expectedParams) == false {
		t.Fatal()
	}
}

func testSerializeParams(t *testing.T, params []Param, delim string, expected []string) {
	result := SerializeParams(params, delim)
	if reflect.DeepEqual(result, expected) == false {
		t.Fatal()
	}
}

func TestSerializeParamsNoParamNoValue(t *testing.T) {
	params := []Param{
		{
			Key:   "",
			Value: "",
		},
	}
	var expected []string

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParamsNoParam(t *testing.T) {
	params := []Param{
		{
			Value: "value1",
		},
	}

	expected := []string{"value1"}

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParamsNoValue(t *testing.T) {
	params := []Param{
		{
			Key: "param1",
		},
	}

	expected := []string{"param1"}

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParamsNoDelim(t *testing.T) {
	params := []Param{
		{
			Key:   "param1",
			Value: "value1",
		},
	}

	expected := []string{"param1", "value1"}

	testSerializeParams(t, params, "", expected)
}

func TestSerializeParams(t *testing.T) {
	params := []Param{
		{
			Key:   "param1",
			Value: "value1",
		},
	}

	expected := []string{"param1=value1"}

	testSerializeParams(t, params, "=", expected)
}

func testDeserializeParams(t *testing.T, parameters []string, expected []Param) {
	result := DeserializeParams(parameters)
	if reflect.DeepEqual(result, expected) == false {
		t.Fatal()
	}
}

func TestDeserializeParamsNil(t *testing.T) {
	var parameters []string
	var expected []Param

	testDeserializeParams(t, parameters, expected)
}

func TestDeserializeParamsNoParamNoValue(t *testing.T) {
	parameters := []string{
		"",
	}

	var expected []Param

	testDeserializeParams(t, parameters, expected)
}

func TestDeserializeParamsNoValue(t *testing.T) {
	parameters := []string{
		"param1",
	}
	expected := []Param{
		{
			Key: "param1",
		},
	}

	testDeserializeParams(t, parameters, expected)
}

func TestDeserializeParams(t *testing.T) {
	parameters := []string{
		"param1=value1",
	}

	expected := []Param{
		{
			Key:   "param1",
			Value: "value1",
		},
	}

	testDeserializeParams(t, parameters, expected)
}

func TestAddKernelParamValid(t *testing.T) {
	var config Config

	expected := []Param{
		{"foo", "bar"},
	}

	err := config.AddKernelParam(expected[0])
	if err != nil || reflect.DeepEqual(config.KernelParams, expected) == false {
		t.Fatal()
	}
}

func TestAddKernelParamInvalid(t *testing.T) {
	var config Config

	invalid := []Param{
		{"", "bar"},
	}

	err := config.AddKernelParam(invalid[0])
	if err == nil {
		t.Fatal()
	}
}

func TestGetHostMemorySizeKb(t *testing.T) {

	type testData struct {
		contents       string
		expectedResult int
		expectError    bool
	}

	data := []testData{
		{
			`
			MemTotal:      1 kB
			MemFree:       2 kB
			SwapTotal:     3 kB
			SwapFree:      4 kB
			`,
			1024,
			false,
		},
		{
			`
			MemFree:       2 kB
			SwapTotal:     3 kB
			SwapFree:      4 kB
			`,
			0,
			true,
		},
	}

	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	file := filepath.Join(dir, "meminfo")
	if _, err := GetHostMemorySizeKb(file); err == nil {
		t.Fatalf("expected failure as file %q does not exist", file)
	}

	for _, d := range data {
		if err := ioutil.WriteFile(file, []byte(d.contents), os.FileMode(0640)); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file)

		hostMemKb, err := GetHostMemorySizeKb(file)

		if (d.expectError && err == nil) || (!d.expectError && err != nil) {
			t.Fatalf("got %d, input %v", hostMemKb, d)
		}

		if reflect.DeepEqual(hostMemKb, d.expectedResult) {
			t.Fatalf("got %d, input %v", hostMemKb, d)
		}
	}
}

var dataFlagsFieldWithoutHypervisor = []byte(`
fpu_exception   : yes
cpuid level     : 20
wp              : yes
flags           : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ss ht syscall nx pdpe1gb rdtscp lm constant_tsc rep_good nopl xtopology eagerfpu pni pclmulqdq vmx ssse3 fma cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx f16c rdrand lahf_lm abm 3dnowprefetch tpr_shadow vnmi ept vpid fsgsbase bmi1 hle avx2 smep bmi2 erms rtm rdseed adx smap xsaveopt
bugs            :
bogomips        : 4589.35
`)

var dataFlagsFieldWithHypervisor = []byte(`
fpu_exception   : yes
cpuid level     : 20
wp              : yes
flags           : fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ss ht syscall nx pdpe1gb rdtscp lm constant_tsc rep_good nopl xtopology eagerfpu pni pclmulqdq vmx ssse3 fma cx16 sse4_1 sse4_2 movbe popcnt aes xsave avx f16c rdrand hypervisor lahf_lm abm 3dnowprefetch tpr_shadow vnmi ept vpid fsgsbase bmi1 hle avx2 smep bmi2 erms rtm rdseed adx smap xsaveopt
bugs            :
bogomips        : 4589.35
`)

var dataWithoutFlagsField = []byte(`
fpu_exception   : yes
cpuid level     : 20
wp              : yes
bugs            :
bogomips        : 4589.35
`)

func testRunningOnVMMSuccessful(t *testing.T, cpuInfoContent []byte, expectedErr bool, expected bool) {
	f, err := ioutil.TempFile("", "cpuinfo")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	n, err := f.Write(cpuInfoContent)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(cpuInfoContent) {
		t.Fatalf("Only %d bytes written out of %d expected", n, len(cpuInfoContent))
	}

	running, err := RunningOnVMM(f.Name())
	if !expectedErr && err != nil {
		t.Fatalf("This test should succeed: %v", err)
	} else if expectedErr && err == nil {
		t.Fatalf("This test should fail")
	}

	if running != expected {
		t.Fatalf("Expecting running on VMM = %t, Got %t", expected, running)
	}
}

func TestRunningOnVMMFalseSuccessful(t *testing.T) {
	testRunningOnVMMSuccessful(t, dataFlagsFieldWithoutHypervisor, false, false)
}

func TestRunningOnVMMTrueSuccessful(t *testing.T) {
	testRunningOnVMMSuccessful(t, dataFlagsFieldWithHypervisor, false, true)
}

func TestRunningOnVMMNoFlagsFieldFailure(t *testing.T) {
	testRunningOnVMMSuccessful(t, dataWithoutFlagsField, true, false)
}

func TestRunningOnVMMNotExistingCPUInfoPathFailure(t *testing.T) {
	f, err := ioutil.TempFile("", "cpuinfo")
	if err != nil {
		t.Fatal(err)
	}

	filePath := f.Name()

	f.Close()
	os.Remove(filePath)

	if _, err := RunningOnVMM(filePath); err == nil {
		t.Fatalf("Should fail because %q file path does not exist", filePath)
	}
}
