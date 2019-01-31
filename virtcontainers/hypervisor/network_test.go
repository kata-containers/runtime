// Copyright (c) 2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vishvananda/netlink"

	"github.com/kata-containers/runtime/virtcontainers/types"
)

func TestGenerateRandomPrivateMacAdd(t *testing.T) {
	assert := assert.New(t)

	addr1, err := generateRandomPrivateMacAddr()
	assert.NoError(err)

	_, err = net.ParseMAC(addr1)
	assert.NoError(err)

	addr2, err := generateRandomPrivateMacAddr()
	assert.NoError(err)

	_, err = net.ParseMAC(addr2)
	assert.NoError(err)

	assert.NotEqual(addr1, addr2)
}

func TestCreateGetBridgeLink(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	assert := assert.New(t)

	netHandle, err := netlink.NewHandle()
	defer netHandle.Delete()

	assert.NoError(err)

	brName := "testbr0"
	brLink, _, err := createLink(netHandle, brName, &netlink.Bridge{}, 1)
	assert.NoError(err)
	assert.NotNil(brLink)

	brLink, err = getLinkByName(netHandle, brName, &netlink.Bridge{})
	assert.NoError(err)

	err = netHandle.LinkDel(brLink)
	assert.NoError(err)
}

func TestCreateGetTunTapLink(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	assert := assert.New(t)

	netHandle, err := netlink.NewHandle()
	defer netHandle.Delete()

	assert.NoError(err)

	tapName := "testtap0"
	tapLink, fds, err := createLink(netHandle, tapName, &netlink.Tuntap{}, 1)
	assert.NoError(err)
	assert.NotNil(tapLink)
	assert.NotZero(len(fds))

	tapLink, err = getLinkByName(netHandle, tapName, &netlink.Tuntap{})
	assert.NoError(err)

	err = netHandle.LinkDel(tapLink)
	assert.NoError(err)
}

func TestCreateMacVtap(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	assert := assert.New(t)

	netHandle, err := netlink.NewHandle()
	defer netHandle.Delete()

	assert.NoError(err)

	brName := "testbr0"
	brLink, _, err := createLink(netHandle, brName, &netlink.Bridge{}, 1)
	assert.NoError(err)

	attrs := brLink.Attrs()

	mcLink := &netlink.Macvtap{
		Macvlan: netlink.Macvlan{
			LinkAttrs: netlink.LinkAttrs{
				TxQLen:      attrs.TxQLen,
				ParentIndex: attrs.Index,
			},
		},
	}

	macvtapName := "testmc0"
	_, err = createMacVtap(netHandle, macvtapName, mcLink, 1)
	assert.NoError(err)

	macvtapLink, err := getLinkByName(netHandle, macvtapName, &netlink.Macvtap{})
	assert.NoError(err)

	err = netHandle.LinkDel(macvtapLink)
	assert.NoError(err)

	brLink, err = getLinkByName(netHandle, brName, &netlink.Bridge{})
	assert.NoError(err)

	err = netHandle.LinkDel(brLink)
	assert.NoError(err)
}

func TestTcRedirectNetwork(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip(testDisabledAsNonRoot)
	}

	assert := assert.New(t)

	netHandle, err := netlink.NewHandle()
	assert.NoError(err)
	defer netHandle.Delete()

	// Create a test veth interface.
	vethName := "foo"
	veth := &netlink.Veth{LinkAttrs: netlink.LinkAttrs{Name: vethName, TxQLen: 200, MTU: 1400}, PeerName: "bar"}

	err = netlink.LinkAdd(veth)
	assert.NoError(err)

	endpoint, err := createVethNetworkEndpoint(1, vethName, types.NetXConnectTCFilterModel)
	assert.NoError(err)

	link, err := netlink.LinkByName(vethName)
	assert.NoError(err)

	err = netHandle.LinkSetUp(link)
	assert.NoError(err)

	err = setupTCFiltering(endpoint, 1, true)
	assert.NoError(err)

	err = removeTCFiltering(endpoint)
	assert.NoError(err)

	// Remove the veth created for testing.
	err = netHandle.LinkDel(link)
	assert.NoError(err)
}
