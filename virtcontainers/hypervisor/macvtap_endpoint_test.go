// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"reflect"
	"testing"

	"github.com/kata-containers/runtime/virtcontainers/types"
)

func TestCreateMacvtapEndpoint(t *testing.T) {
	netInfo := types.NetworkInfo{
		Iface: types.NetlinkIface{
			Type: "macvtap",
		},
	}
	expected := &MacvtapEndpoint{
		EndpointType:       MacvtapEndpointType,
		EndpointProperties: netInfo,
	}

	result, err := createMacvtapNetworkEndpoint(netInfo)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(result, expected) == false {
		t.Fatalf("\nGot: %+v, \n\nExpected: %+v", result, expected)
	}
}
