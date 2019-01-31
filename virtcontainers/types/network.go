// Copyright (c) 2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package types

import (
	"fmt"
	"os"

	"github.com/vishvananda/netlink"
)

// NetInterworkingModel defines the network model connecting
// the network interface to the virtual machine.
type NetInterworkingModel int

const (
	// NetXConnectDefaultModel Ask to use DefaultNetInterworkingModel
	NetXConnectDefaultModel NetInterworkingModel = iota

	// NetXConnectBridgedModel uses a linux bridge to interconnect
	// the container interface to the VM. This is the
	// safe default that works for most cases except
	// macvlan and ipvlan
	NetXConnectBridgedModel

	// NetXConnectMacVtapModel can be used when the Container network
	// interface can be bridged using macvtap
	NetXConnectMacVtapModel

	// NetXConnectEnlightenedModel can be used when the Network plugins
	// are enlightened to create VM native interfaces
	// when requested by the runtime
	// This will be used for vethtap, macvtap, ipvtap
	NetXConnectEnlightenedModel

	// NetXConnectTCFilterModel redirects traffic from the network interface
	// provided by the network plugin to a tap interface.
	// This works for ipvlan and macvlan as well.
	NetXConnectTCFilterModel

	// NetXConnectNoneModel can be used when the VM is in the host network namespace
	NetXConnectNoneModel

	// NetXConnectInvalidModel is the last item to check valid values by IsValid()
	NetXConnectInvalidModel
)

// DefaultNetInterworkingModel is a package level default
// that determines how the VM should be connected to the
// the container network interface
var DefaultNetInterworkingModel = NetXConnectMacVtapModel

//IsValid checks if a model is valid
func (n NetInterworkingModel) IsValid() bool {
	return 0 <= int(n) && int(n) < int(NetXConnectInvalidModel)
}

const (
	// DefaultNetModelStr is the default networking model.
	DefaultNetModelStr = "default"

	// BridgedNetModelStr is the bridged networking model.
	BridgedNetModelStr = "bridged"

	// MacvtapNetModelStr is the MacVTap networking model.
	MacvtapNetModelStr = "macvtap"

	// EnlightenedNetModelStr is the VM enlightened networking model.
	EnlightenedNetModelStr = "enlightened"

	// TcFilterNetModelStr is the Traffic Control filtering networking model.
	TcFilterNetModelStr = "tcfilter"

	// NoneNetModelStr is the no network networking model.
	NoneNetModelStr = "none"
)

//SetModel change the model string value
func (n *NetInterworkingModel) SetModel(modelName string) error {
	switch modelName {
	case DefaultNetModelStr:
		*n = DefaultNetInterworkingModel
		return nil
	case BridgedNetModelStr:
		*n = NetXConnectBridgedModel
		return nil
	case MacvtapNetModelStr:
		*n = NetXConnectMacVtapModel
		return nil
	case EnlightenedNetModelStr:
		*n = NetXConnectEnlightenedModel
		return nil
	case TcFilterNetModelStr:
		*n = NetXConnectTCFilterModel
		return nil
	case NoneNetModelStr:
		*n = NetXConnectNoneModel
		return nil
	}
	return fmt.Errorf("Unknown type %s", modelName)
}

// Introduces constants related to networking
const (
	DefaultRouteDest  = "0.0.0.0/0"
	DefaultRouteLabel = "default"
	DefaultFilePerms  = 0600
	DefaultQlen       = 1500
)

// DNSInfo describes the DNS setup related to a network interface.
type DNSInfo struct {
	Servers  []string
	Domain   string
	Searches []string
	Options  []string
}

// NetlinkIface describes fully a network interface.
type NetlinkIface struct {
	netlink.LinkAttrs
	Type string
}

// NetworkInfo gathers all information related to a network interface.
// It can be used to store the description of the underlying network.
type NetworkInfo struct {
	Iface  NetlinkIface
	Addrs  []netlink.Addr
	Routes []netlink.Route
	DNS    DNSInfo
}

// NetworkInterface defines a network interface.
type NetworkInterface struct {
	Name     string
	HardAddr string
	Addrs    []netlink.Addr
}

// TapInterface defines a tap interface
type TapInterface struct {
	ID       string
	Name     string
	TAPIface NetworkInterface
	VMFds    []*os.File
	VhostFds []*os.File
}

// NetworkInterfacePair defines a pair between VM and virtual network interfaces.
type NetworkInterfacePair struct {
	TapInterface
	VirtIface NetworkInterface
	NetInterworkingModel
}

// NetworkConfig is the network configuration related to a network.
type NetworkConfig struct {
	NetNSPath         string
	NetNsCreated      bool
	DisableNewNetNs   bool
	NetmonConfig      NetmonConfig
	InterworkingModel NetInterworkingModel
}

// NetmonConfig is the structure providing specific configuration
// for the network monitor.
type NetmonConfig struct {
	Path   string
	Debug  bool
	Enable bool
}
