// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"net"
	"os"
	"reflect"
	"testing"

	"github.com/kata-containers/runtime/virtcontainers/hypervisor"
	vcTypes "github.com/kata-containers/runtime/virtcontainers/pkg/types"
	"github.com/kata-containers/runtime/virtcontainers/types"
	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"
)

func TestCreateDeleteNetNS(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	netNSPath, err := createNetNS()
	if err != nil {
		t.Fatal(err)
	}

	if netNSPath == "" {
		t.Fatal()
	}

	_, err = os.Stat(netNSPath)
	if err != nil {
		t.Fatal(err)
	}

	err = deleteNetNS(netNSPath)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGenerateInterfacesAndRoutes(t *testing.T) {
	//
	//Create a couple of addresses
	//
	address1 := &net.IPNet{IP: net.IPv4(172, 17, 0, 2), Mask: net.CIDRMask(16, 32)}
	address2 := &net.IPNet{IP: net.IPv4(182, 17, 0, 2), Mask: net.CIDRMask(16, 32)}

	addrs := []netlink.Addr{
		{IPNet: address1, Label: "phyaddr1"},
		{IPNet: address2, Label: "phyaddr2"},
	}

	// Create a couple of routes:
	dst2 := &net.IPNet{IP: net.IPv4(172, 17, 0, 0), Mask: net.CIDRMask(16, 32)}
	src2 := net.IPv4(172, 17, 0, 2)
	gw2 := net.IPv4(172, 17, 0, 1)

	routes := []netlink.Route{
		{LinkIndex: 329, Dst: nil, Src: nil, Gw: net.IPv4(172, 17, 0, 1), Scope: netlink.Scope(254)},
		{LinkIndex: 329, Dst: dst2, Src: src2, Gw: gw2},
	}

	networkInfo := types.NetworkInfo{
		Iface: types.NetlinkIface{
			LinkAttrs: netlink.LinkAttrs{MTU: 1500},
			Type:      "",
		},
		Addrs:  addrs,
		Routes: routes,
	}

	ep0 := &hypervisor.PhysicalEndpoint{
		IfaceName:          "eth0",
		HardAddr:           net.HardwareAddr{0x02, 0x00, 0xca, 0xfe, 0x00, 0x04}.String(),
		EndpointProperties: networkInfo,
	}

	endpoints := []hypervisor.Endpoint{ep0}

	nns := NetworkNamespace{NetNsPath: "foobar", NetNsCreated: true, Endpoints: endpoints}

	resInterfaces, resRoutes, err := nns.interfacesAndRoutes(nns)

	//
	// Build expected results:
	//
	expectedAddresses := []*vcTypes.IPAddress{
		{Family: netlink.FAMILY_V4, Address: "172.17.0.2", Mask: "16"},
		{Family: netlink.FAMILY_V4, Address: "182.17.0.2", Mask: "16"},
	}

	expectedInterfaces := []*vcTypes.Interface{
		{Device: "eth0", Name: "eth0", IPAddresses: expectedAddresses, Mtu: 1500, HwAddr: "02:00:ca:fe:00:04"},
	}

	expectedRoutes := []*vcTypes.Route{
		{Dest: "", Gateway: "172.17.0.1", Device: "eth0", Source: "", Scope: uint32(254)},
		{Dest: "172.17.0.0/16", Gateway: "172.17.0.1", Device: "eth0", Source: "172.17.0.2"},
	}

	assert.Nil(t, err, "unexpected failure when calling generateKataInterfacesAndRoutes")
	assert.True(t, reflect.DeepEqual(resInterfaces, expectedInterfaces),
		"Interfaces returned didn't match: got %+v, expecting %+v", resInterfaces, expectedInterfaces)
	assert.True(t, reflect.DeepEqual(resRoutes, expectedRoutes),
		"Routes returned didn't match: got %+v, expecting %+v", resRoutes, expectedRoutes)

}
