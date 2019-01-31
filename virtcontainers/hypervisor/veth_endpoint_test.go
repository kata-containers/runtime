// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"net"
	"reflect"
	"testing"

	"github.com/kata-containers/runtime/virtcontainers/types"
)

func TestCreateVethNetworkEndpoint(t *testing.T) {
	macAddr := net.HardwareAddr{0x02, 0x00, 0xCA, 0xFE, 0x00, 0x04}

	expected := &VethEndpoint{
		NetPair: types.NetworkInterfacePair{
			TapInterface: types.TapInterface{
				ID:   "uniqueTestID-4",
				Name: "br4_kata",
				TAPIface: types.NetworkInterface{
					Name: "tap4_kata",
				},
			},
			VirtIface: types.NetworkInterface{
				Name:     "eth4",
				HardAddr: macAddr.String(),
			},
			NetInterworkingModel: types.DefaultNetInterworkingModel,
		},
		EndpointType: VethEndpointType,
	}

	result, err := createVethNetworkEndpoint(4, "", types.DefaultNetInterworkingModel)
	if err != nil {
		t.Fatal(err)
	}

	// the resulting ID  will be random - so let's overwrite to test the rest of the flow
	result.NetPair.ID = "uniqueTestID-4"

	// the resulting mac address will be random - so lets overwrite it
	result.NetPair.VirtIface.HardAddr = macAddr.String()

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("\nGot: %+v, \n\nExpected: %+v", result, expected)
	}
}

func TestCreateVethNetworkEndpointChooseIfaceName(t *testing.T) {
	macAddr := net.HardwareAddr{0x02, 0x00, 0xCA, 0xFE, 0x00, 0x04}

	expected := &VethEndpoint{
		NetPair: types.NetworkInterfacePair{
			TapInterface: types.TapInterface{
				ID:   "uniqueTestID-4",
				Name: "br4_kata",
				TAPIface: types.NetworkInterface{
					Name: "tap4_kata",
				},
			},
			VirtIface: types.NetworkInterface{
				Name:     "eth1",
				HardAddr: macAddr.String(),
			},
			NetInterworkingModel: types.DefaultNetInterworkingModel,
		},
		EndpointType: VethEndpointType,
	}

	result, err := createVethNetworkEndpoint(4, "eth1", types.DefaultNetInterworkingModel)
	if err != nil {
		t.Fatal(err)
	}

	// the resulting ID will be random - so let's overwrite to test the rest of the flow
	result.NetPair.ID = "uniqueTestID-4"

	// the resulting mac address will be random - so lets overwrite it
	result.NetPair.VirtIface.HardAddr = macAddr.String()

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("\nGot: %+v, \n\nExpected: %+v", result, expected)
	}
}

func TestCreateVethNetworkEndpointInvalidArgs(t *testing.T) {
	type endpointValues struct {
		idx    int
		ifName string
	}

	// all elements are expected to result in failure
	failingValues := []endpointValues{
		{-1, "bar"},
		{-1, ""},
	}

	for _, d := range failingValues {
		result, err := createVethNetworkEndpoint(d.idx, d.ifName, types.DefaultNetInterworkingModel)
		if err == nil {
			t.Fatalf("expected invalid endpoint for %v, got %v", d, result)
		}
	}
}
