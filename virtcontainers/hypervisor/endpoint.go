// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	"encoding/json"
	"fmt"
	"net"
	"sort"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"

	"github.com/kata-containers/runtime/virtcontainers/types"
)

// Endpoint represents a physical or virtual network interface.
type Endpoint interface {
	Properties() types.NetworkInfo
	Name() string
	HardwareAddr() string
	Type() EndpointType
	PciAddr() string
	NetworkPair() *types.NetworkInterfacePair

	SetProperties(types.NetworkInfo)
	SetPciAddr(string)
	Attach(Hypervisor) error
	Detach(netNsCreated bool, netNsPath string) error
	HotAttach(Hypervisor) error
	HotDetach(h Hypervisor, netNsCreated bool, netNsPath string) error
}

// EndpointType identifies the type of the network endpoint.
type EndpointType string

const (
	// PhysicalEndpointType is the physical network interface.
	PhysicalEndpointType EndpointType = "physical"

	// VethEndpointType is the virtual network interface.
	VethEndpointType EndpointType = "virtual"

	// VhostUserEndpointType is the vhostuser network interface.
	VhostUserEndpointType EndpointType = "vhost-user"

	// BridgedMacvlanEndpointType is macvlan network interface.
	BridgedMacvlanEndpointType EndpointType = "macvlan"

	// MacvtapEndpointType is macvtap network interface.
	MacvtapEndpointType EndpointType = "macvtap"

	// TapEndpointType is tap network interface.
	TapEndpointType EndpointType = "tap"

	// IPVlanEndpointType is ipvlan network interface.
	IPVlanEndpointType EndpointType = "ipvlan"
)

// Set sets an endpoint type based on the input string.
func (endpointType *EndpointType) Set(value string) error {
	switch value {
	case "physical":
		*endpointType = PhysicalEndpointType
		return nil
	case "virtual":
		*endpointType = VethEndpointType
		return nil
	case "vhost-user":
		*endpointType = VhostUserEndpointType
		return nil
	case "macvlan":
		*endpointType = BridgedMacvlanEndpointType
		return nil
	case "macvtap":
		*endpointType = MacvtapEndpointType
		return nil
	case "tap":
		*endpointType = TapEndpointType
		return nil
	case "ipvlan":
		*endpointType = IPVlanEndpointType
		return nil
	default:
		return fmt.Errorf("Unknown endpoint type %s", value)
	}
}

// String converts an endpoint type to a string.
func (endpointType *EndpointType) String() string {
	switch *endpointType {
	case PhysicalEndpointType:
		return string(PhysicalEndpointType)
	case VethEndpointType:
		return string(VethEndpointType)
	case VhostUserEndpointType:
		return string(VhostUserEndpointType)
	case BridgedMacvlanEndpointType:
		return string(BridgedMacvlanEndpointType)
	case MacvtapEndpointType:
		return string(MacvtapEndpointType)
	case TapEndpointType:
		return string(TapEndpointType)
	case IPVlanEndpointType:
		return string(IPVlanEndpointType)
	default:
		return ""
	}
}

func getLinkForEndpoint(endpoint Endpoint, netHandle *netlink.Handle) (netlink.Link, error) {
	var link netlink.Link

	switch ep := endpoint.(type) {
	case *VethEndpoint:
		link = &netlink.Veth{}
	case *BridgedMacvlanEndpoint:
		link = &netlink.Macvlan{}
	case *IPVlanEndpoint:
		link = &netlink.IPVlan{}
	default:
		return nil, fmt.Errorf("Unexpected endpointType %s", ep.Type())
	}

	return getLinkByName(netHandle, endpoint.NetworkPair().VirtIface.Name, link)
}

func tapNetworkPair(endpoint Endpoint, queues int, disableVhostNet bool) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	netPair := endpoint.NetworkPair()

	link, err := getLinkForEndpoint(endpoint, netHandle)
	if err != nil {
		return err
	}

	attrs := link.Attrs()

	// Attach the macvtap interface to the underlying container
	// interface. Also picks relevant attributes from the parent
	tapLink, err := createMacVtap(netHandle, netPair.TAPIface.Name,
		&netlink.Macvtap{
			Macvlan: netlink.Macvlan{
				LinkAttrs: netlink.LinkAttrs{
					TxQLen:      attrs.TxQLen,
					ParentIndex: attrs.Index,
				},
			},
		}, queues)

	if err != nil {
		return fmt.Errorf("Could not create TAP interface: %s", err)
	}

	// Save the veth MAC address to the TAP so that it can later be used
	// to build the hypervisor command line. This MAC address has to be
	// the one inside the VM in order to avoid any firewall issues. The
	// bridge created by the network plugin on the host actually expects
	// to see traffic from this MAC address and not another one.
	tapHardAddr := attrs.HardwareAddr
	netPair.TAPIface.HardAddr = attrs.HardwareAddr.String()

	if err := netHandle.LinkSetMTU(tapLink, attrs.MTU); err != nil {
		return fmt.Errorf("Could not set TAP MTU %d: %s", attrs.MTU, err)
	}

	hardAddr, err := net.ParseMAC(netPair.VirtIface.HardAddr)
	if err != nil {
		return err
	}
	if err := netHandle.LinkSetHardwareAddr(link, hardAddr); err != nil {
		return fmt.Errorf("Could not set MAC address %s for veth interface %s: %s",
			netPair.VirtIface.HardAddr, netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetHardwareAddr(tapLink, tapHardAddr); err != nil {
		return fmt.Errorf("Could not set MAC address %s for veth interface %s: %s",
			netPair.VirtIface.HardAddr, netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetUp(tapLink); err != nil {
		return fmt.Errorf("Could not enable TAP %s: %s", netPair.TAPIface.Name, err)
	}

	// Clear the IP addresses from the veth interface to prevent ARP conflict
	netPair.VirtIface.Addrs, err = netlink.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("Unable to obtain veth IP addresses: %s", err)
	}

	if err := clearIPs(link, netPair.VirtIface.Addrs); err != nil {
		return fmt.Errorf("Unable to clear veth IP addresses: %s", err)
	}

	if err := netHandle.LinkSetUp(link); err != nil {
		return fmt.Errorf("Could not enable veth %s: %s", netPair.VirtIface.Name, err)
	}

	// Note: The underlying interfaces need to be up prior to fd creation.

	netPair.VMFds, err = createMacvtapFds(tapLink.Attrs().Index, queues)
	if err != nil {
		return fmt.Errorf("Could not setup macvtap fds %s: %s", netPair.TAPIface, err)
	}

	if !disableVhostNet {
		vhostFds, err := createVhostFds(queues)
		if err != nil {
			return fmt.Errorf("Could not setup vhost fds %s : %s", netPair.VirtIface.Name, err)
		}
		netPair.VhostFds = vhostFds
	}

	return nil
}

func bridgeNetworkPair(endpoint Endpoint, queues int, disableVhostNet bool) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	netPair := endpoint.NetworkPair()

	tapLink, fds, err := createLink(netHandle, netPair.TAPIface.Name, &netlink.Tuntap{}, queues)
	if err != nil {
		return fmt.Errorf("Could not create TAP interface: %s", err)
	}
	netPair.VMFds = fds

	if !disableVhostNet {
		vhostFds, err := createVhostFds(queues)
		if err != nil {
			return fmt.Errorf("Could not setup vhost fds %s : %s", netPair.VirtIface.Name, err)
		}
		netPair.VhostFds = vhostFds
	}

	var attrs *netlink.LinkAttrs
	var link netlink.Link

	link, err = getLinkForEndpoint(endpoint, netHandle)
	if err != nil {
		return err
	}

	attrs = link.Attrs()

	// Save the veth MAC address to the TAP so that it can later be used
	// to build the hypervisor command line. This MAC address has to be
	// the one inside the VM in order to avoid any firewall issues. The
	// bridge created by the network plugin on the host actually expects
	// to see traffic from this MAC address and not another one.
	netPair.TAPIface.HardAddr = attrs.HardwareAddr.String()

	if err := netHandle.LinkSetMTU(tapLink, attrs.MTU); err != nil {
		return fmt.Errorf("Could not set TAP MTU %d: %s", attrs.MTU, err)
	}

	hardAddr, err := net.ParseMAC(netPair.VirtIface.HardAddr)
	if err != nil {
		return err
	}
	if err := netHandle.LinkSetHardwareAddr(link, hardAddr); err != nil {
		return fmt.Errorf("Could not set MAC address %s for veth interface %s: %s",
			netPair.VirtIface.HardAddr, netPair.VirtIface.Name, err)
	}

	mcastSnoop := false
	bridgeLink, _, err := createLink(netHandle, netPair.Name, &netlink.Bridge{MulticastSnooping: &mcastSnoop}, queues)
	if err != nil {
		return fmt.Errorf("Could not create bridge: %s", err)
	}

	if err := netHandle.LinkSetMaster(tapLink, bridgeLink.(*netlink.Bridge)); err != nil {
		return fmt.Errorf("Could not attach TAP %s to the bridge %s: %s",
			netPair.TAPIface.Name, netPair.Name, err)
	}

	if err := netHandle.LinkSetUp(tapLink); err != nil {
		return fmt.Errorf("Could not enable TAP %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkSetMaster(link, bridgeLink.(*netlink.Bridge)); err != nil {
		return fmt.Errorf("Could not attach veth %s to the bridge %s: %s",
			netPair.VirtIface.Name, netPair.Name, err)
	}

	if err := netHandle.LinkSetUp(link); err != nil {
		return fmt.Errorf("Could not enable veth %s: %s", netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetUp(bridgeLink); err != nil {
		return fmt.Errorf("Could not enable bridge %s: %s", netPair.Name, err)
	}

	return nil
}

func setupTCFiltering(endpoint Endpoint, queues int, disableVhostNet bool) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	netPair := endpoint.NetworkPair()

	tapLink, fds, err := createLink(netHandle, netPair.TAPIface.Name, &netlink.Tuntap{}, queues)
	if err != nil {
		return fmt.Errorf("Could not create TAP interface: %s", err)
	}
	netPair.VMFds = fds

	if !disableVhostNet {
		vhostFds, err := createVhostFds(queues)
		if err != nil {
			return fmt.Errorf("Could not setup vhost fds %s : %s", netPair.VirtIface.Name, err)
		}
		netPair.VhostFds = vhostFds
	}

	var attrs *netlink.LinkAttrs
	var link netlink.Link

	link, err = getLinkForEndpoint(endpoint, netHandle)
	if err != nil {
		return err
	}

	attrs = link.Attrs()

	// Save the veth MAC address to the TAP so that it can later be used
	// to build the hypervisor command line. This MAC address has to be
	// the one inside the VM in order to avoid any firewall issues. The
	// bridge created by the network plugin on the host actually expects
	// to see traffic from this MAC address and not another one.
	netPair.TAPIface.HardAddr = attrs.HardwareAddr.String()

	if err := netHandle.LinkSetMTU(tapLink, attrs.MTU); err != nil {
		return fmt.Errorf("Could not set TAP MTU %d: %s", attrs.MTU, err)
	}

	if err := netHandle.LinkSetUp(tapLink); err != nil {
		return fmt.Errorf("Could not enable TAP %s: %s", netPair.TAPIface.Name, err)
	}

	tapAttrs := tapLink.Attrs()

	if err := addQdiscIngress(tapAttrs.Index); err != nil {
		return err
	}

	if err := addQdiscIngress(attrs.Index); err != nil {
		return err
	}

	if err := addRedirectTCFilter(attrs.Index, tapAttrs.Index); err != nil {
		return err
	}

	if err := addRedirectTCFilter(tapAttrs.Index, attrs.Index); err != nil {
		return err
	}

	return nil
}

func untapNetworkPair(endpoint Endpoint) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	netPair := endpoint.NetworkPair()

	tapLink, err := getLinkByName(netHandle, netPair.TAPIface.Name, &netlink.Macvtap{})
	if err != nil {
		return fmt.Errorf("Could not get TAP interface %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkDel(tapLink); err != nil {
		return fmt.Errorf("Could not remove TAP %s: %s", netPair.TAPIface.Name, err)
	}

	link, err := getLinkForEndpoint(endpoint, netHandle)
	if err != nil {
		return err
	}

	hardAddr, err := net.ParseMAC(netPair.TAPIface.HardAddr)
	if err != nil {
		return err
	}
	if err := netHandle.LinkSetHardwareAddr(link, hardAddr); err != nil {
		return fmt.Errorf("Could not set MAC address %s for veth interface %s: %s",
			netPair.VirtIface.HardAddr, netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetDown(link); err != nil {
		return fmt.Errorf("Could not disable veth %s: %s", netPair.VirtIface.Name, err)
	}

	// Restore the IPs that were cleared
	err = setIPs(link, netPair.VirtIface.Addrs)
	return err
}

func unBridgeNetworkPair(endpoint Endpoint) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	netPair := endpoint.NetworkPair()

	tapLink, err := getLinkByName(netHandle, netPair.TAPIface.Name, &netlink.Tuntap{})
	if err != nil {
		return fmt.Errorf("Could not get TAP interface: %s", err)
	}

	bridgeLink, err := getLinkByName(netHandle, netPair.Name, &netlink.Bridge{})
	if err != nil {
		return fmt.Errorf("Could not get bridge interface: %s", err)
	}

	if err := netHandle.LinkSetDown(bridgeLink); err != nil {
		return fmt.Errorf("Could not disable bridge %s: %s", netPair.Name, err)
	}

	if err := netHandle.LinkSetDown(tapLink); err != nil {
		return fmt.Errorf("Could not disable TAP %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkSetNoMaster(tapLink); err != nil {
		return fmt.Errorf("Could not detach TAP %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkDel(bridgeLink); err != nil {
		return fmt.Errorf("Could not remove bridge %s: %s", netPair.Name, err)
	}

	if err := netHandle.LinkDel(tapLink); err != nil {
		return fmt.Errorf("Could not remove TAP %s: %s", netPair.TAPIface.Name, err)
	}

	link, err := getLinkForEndpoint(endpoint, netHandle)
	if err != nil {
		return err
	}

	hardAddr, err := net.ParseMAC(netPair.TAPIface.HardAddr)
	if err != nil {
		return err
	}
	if err := netHandle.LinkSetHardwareAddr(link, hardAddr); err != nil {
		return fmt.Errorf("Could not set MAC address %s for veth interface %s: %s",
			netPair.VirtIface.HardAddr, netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetDown(link); err != nil {
		return fmt.Errorf("Could not disable veth %s: %s", netPair.VirtIface.Name, err)
	}

	if err := netHandle.LinkSetNoMaster(link); err != nil {
		return fmt.Errorf("Could not detach veth %s: %s", netPair.VirtIface.Name, err)
	}

	return nil
}

func removeTCFiltering(endpoint Endpoint) error {
	netHandle, err := netlink.NewHandle()
	if err != nil {
		return err
	}
	defer netHandle.Delete()

	netPair := endpoint.NetworkPair()

	tapLink, err := getLinkByName(netHandle, netPair.TAPIface.Name, &netlink.Tuntap{})
	if err != nil {
		return fmt.Errorf("Could not get TAP interface: %s", err)
	}

	if err := netHandle.LinkSetDown(tapLink); err != nil {
		return fmt.Errorf("Could not disable TAP %s: %s", netPair.TAPIface.Name, err)
	}

	if err := netHandle.LinkDel(tapLink); err != nil {
		return fmt.Errorf("Could not remove TAP %s: %s", netPair.TAPIface.Name, err)
	}

	link, err := getLinkForEndpoint(endpoint, netHandle)
	if err != nil {
		return err
	}

	if err := removeRedirectTCFilter(link); err != nil {
		return err
	}

	if err := removeQdiscIngress(link); err != nil {
		return err
	}

	if err := netHandle.LinkSetDown(link); err != nil {
		return fmt.Errorf("Could not disable veth %s: %s", netPair.VirtIface.Name, err)
	}

	return nil
}

// The endpoint type should dictate how the connection needs to happen.
func xConnectVMNetwork(endpoint Endpoint, h Hypervisor) error {
	netPair := endpoint.NetworkPair()

	queues := 0
	caps := h.Capabilities()
	if caps.IsMultiQueueSupported() {
		queues = int(h.Config().NumVCPUs)
	}

	disableVhostNet := h.Config().DisableVhostNet

	if netPair.NetInterworkingModel == types.NetXConnectDefaultModel {
		netPair.NetInterworkingModel = types.DefaultNetInterworkingModel
	}

	switch netPair.NetInterworkingModel {
	case types.NetXConnectBridgedModel:
		return bridgeNetworkPair(endpoint, queues, disableVhostNet)
	case types.NetXConnectMacVtapModel:
		return tapNetworkPair(endpoint, queues, disableVhostNet)
	case types.NetXConnectTCFilterModel:
		return setupTCFiltering(endpoint, queues, disableVhostNet)
	case types.NetXConnectEnlightenedModel:
		return fmt.Errorf("Unsupported networking model")
	default:
		return fmt.Errorf("Invalid internetworking model")
	}
}

// The endpoint type should dictate how the disconnection needs to happen.
func xDisconnectVMNetwork(endpoint Endpoint) error {
	netPair := endpoint.NetworkPair()

	if netPair.NetInterworkingModel == types.NetXConnectDefaultModel {
		netPair.NetInterworkingModel = types.DefaultNetInterworkingModel
	}

	switch netPair.NetInterworkingModel {
	case types.NetXConnectBridgedModel:
		return unBridgeNetworkPair(endpoint)
	case types.NetXConnectMacVtapModel:
		return untapNetworkPair(endpoint)
	case types.NetXConnectTCFilterModel:
		return removeTCFiltering(endpoint)
	case types.NetXConnectEnlightenedModel:
		return fmt.Errorf("Unsupported networking model")
	default:
		return fmt.Errorf("Invalid internetworking model")
	}
}

// CreateEndpoint creates an Endpoint from a network info and interworking model.
func CreateEndpoint(netInfo types.NetworkInfo, idx int, model types.NetInterworkingModel) (Endpoint, error) {
	var endpoint Endpoint
	// TODO: This is the incoming interface
	// based on the incoming interface we should create
	// an appropriate EndPoint based on interface type
	// This should be a switch

	// Check if interface is a physical interface. Do not create
	// tap interface/bridge if it is.
	isPhysical, err := isPhysicalIface(netInfo.Iface.Name)
	if err != nil {
		return nil, err
	}

	if isPhysical {
		networkLogger().WithField("interface", netInfo.Iface.Name).Info("Physical network interface found")
		endpoint, err = createPhysicalEndpoint(netInfo)
	} else {
		var socketPath string

		// Check if this is a dummy interface which has a vhost-user socket associated with it
		socketPath, err = vhostUserSocketPath(netInfo)
		if err != nil {
			return nil, err
		}

		if socketPath != "" {
			networkLogger().WithField("interface", netInfo.Iface.Name).Info("VhostUser network interface found")
			endpoint, err = createVhostUserEndpoint(netInfo, socketPath)
		} else if netInfo.Iface.Type == "macvlan" {
			networkLogger().Infof("macvlan interface found")
			endpoint, err = createBridgedMacvlanNetworkEndpoint(idx, netInfo.Iface.Name, model)
		} else if netInfo.Iface.Type == "macvtap" {
			networkLogger().Infof("macvtap interface found")
			endpoint, err = createMacvtapNetworkEndpoint(netInfo)
		} else if netInfo.Iface.Type == "tap" {
			networkLogger().Info("tap interface found")
			endpoint, err = createTapNetworkEndpoint(idx, netInfo.Iface.Name)
		} else if netInfo.Iface.Type == "veth" {
			endpoint, err = createVethNetworkEndpoint(idx, netInfo.Iface.Name, model)
		} else if netInfo.Iface.Type == "ipvlan" {
			endpoint, err = createIPVlanNetworkEndpoint(idx, netInfo.Iface.Name)
		} else {
			return nil, fmt.Errorf("Unsupported network interface")
		}
	}

	return endpoint, err
}

// CreateEndpointsFromScan creates a slice of Endpoints from a network namespace scan.
func CreateEndpointsFromScan(networkNSPath string, config *types.NetworkConfig) ([]Endpoint, error) {
	var endpoints []Endpoint

	netnsHandle, err := netns.GetFromPath(networkNSPath)
	if err != nil {
		return []Endpoint{}, err
	}
	defer netnsHandle.Close()

	netlinkHandle, err := netlink.NewHandleAt(netnsHandle)
	if err != nil {
		return []Endpoint{}, err
	}
	defer netlinkHandle.Delete()

	linkList, err := netlinkHandle.LinkList()
	if err != nil {
		return []Endpoint{}, err
	}

	idx := 0
	for _, link := range linkList {
		var (
			endpoint  Endpoint
			errCreate error
		)

		netInfo, err := networkInfoFromLink(netlinkHandle, link)
		if err != nil {
			return []Endpoint{}, err
		}

		// Ignore unconfigured network interfaces. These are
		// either base tunnel devices that are not namespaced
		// like gre0, gretap0, sit0, ipip0, tunl0 or incorrectly
		// setup interfaces.
		if len(netInfo.Addrs) == 0 {
			continue
		}

		// Skip any loopback interfaces:
		if (netInfo.Iface.Flags & net.FlagLoopback) != 0 {
			continue
		}

		if err := DoNetNS(networkNSPath, func(_ ns.NetNS) error {
			endpoint, errCreate = CreateEndpoint(netInfo, idx, config.InterworkingModel)
			return errCreate
		}); err != nil {
			return []Endpoint{}, err
		}

		endpoint.SetProperties(netInfo)
		endpoints = append(endpoints, endpoint)

		idx++
	}

	sort.Slice(endpoints, func(i, j int) bool {
		return endpoints[i].Name() < endpoints[j].Name()
	})

	networkLogger().WithField("endpoints", endpoints).Info("Endpoints found after scan")

	return endpoints, nil
}

// TypedJSONEndpoint is used as an intermediate representation for
// marshalling and unmarshalling Endpoint objects.
type TypedJSONEndpoint struct {
	Type EndpointType
	Data json.RawMessage
}

// UnmarshalEndpoints unmarshalls a slice of typed Endpoints into a slice of regular ones.
func UnmarshalEndpoints(typedEndpoints []TypedJSONEndpoint) ([]Endpoint, error) {
	var endpoints []Endpoint

	for _, e := range typedEndpoints {
		switch e.Type {
		case PhysicalEndpointType:
			var endpoint PhysicalEndpoint
			err := json.Unmarshal(e.Data, &endpoint)
			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, &endpoint)
			networkLogger().WithFields(logrus.Fields{
				"endpoint":      endpoint,
				"endpoint-type": "physical",
			}).Info("endpoint unmarshalled")

		case VethEndpointType:
			var endpoint VethEndpoint
			err := json.Unmarshal(e.Data, &endpoint)
			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, &endpoint)
			networkLogger().WithFields(logrus.Fields{
				"endpoint":      endpoint,
				"endpoint-type": "virtual",
			}).Info("endpoint unmarshalled")

		case VhostUserEndpointType:
			var endpoint VhostUserEndpoint
			err := json.Unmarshal(e.Data, &endpoint)
			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, &endpoint)
			networkLogger().WithFields(logrus.Fields{
				"endpoint":      endpoint,
				"endpoint-type": "vhostuser",
			}).Info("endpoint unmarshalled")

		case BridgedMacvlanEndpointType:
			var endpoint BridgedMacvlanEndpoint
			err := json.Unmarshal(e.Data, &endpoint)
			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, &endpoint)
			networkLogger().WithFields(logrus.Fields{
				"endpoint":      endpoint,
				"endpoint-type": "macvlan",
			}).Info("endpoint unmarshalled")

		case MacvtapEndpointType:
			var endpoint MacvtapEndpoint
			err := json.Unmarshal(e.Data, &endpoint)
			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, &endpoint)
			networkLogger().WithFields(logrus.Fields{
				"endpoint":      endpoint,
				"endpoint-type": "macvtap",
			}).Info("endpoint unmarshalled")

		case TapEndpointType:
			var endpoint TapEndpoint
			err := json.Unmarshal(e.Data, &endpoint)
			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, &endpoint)
			networkLogger().WithFields(logrus.Fields{
				"endpoint":      endpoint,
				"endpoint-type": "tap",
			}).Info("endpoint unmarshalled")

		default:
			networkLogger().WithField("endpoint-type", e.Type).Error("Ignoring unknown endpoint type")
		}
	}
	return endpoints, nil
}
