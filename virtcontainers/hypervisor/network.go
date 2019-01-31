// Copyright (c) 2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package hypervisor

import (
	cryptoRand "crypto/rand"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/kata-containers/runtime/virtcontainers/pkg/uuid"
	"github.com/kata-containers/runtime/virtcontainers/types"
	"github.com/kata-containers/runtime/virtcontainers/utils"
)

func networkLogger() *logrus.Entry {
	return logrus.WithField("source", "virtcontainers/hypervisor").WithField("subsystem", "network")
}

// DoNetNS is free from any call to a go routine, and it calls
// into runtime.LockOSThread(), meaning it won't be executed in a
// different thread than the one expected by the caller.
func DoNetNS(netNSPath string, cb func(ns.NetNS) error) error {
	// if netNSPath is empty, the callback function will be run in the current network namespace.
	// So skip the whole function, just call cb(). cb() needs a NetNS as arg but ignored, give it a fake one.
	if netNSPath == "" {
		var netNs ns.NetNS
		return cb(netNs)
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	currentNS, err := ns.GetCurrentNS()
	if err != nil {
		return err
	}
	defer currentNS.Close()

	targetNS, err := ns.GetNS(netNSPath)
	if err != nil {
		return err
	}

	if err := targetNS.Set(); err != nil {
		return err
	}
	defer currentNS.Set()

	return cb(targetNS)
}

func createLink(netHandle *netlink.Handle, name string, expectedLink netlink.Link, queues int) (netlink.Link, []*os.File, error) {
	var newLink netlink.Link
	var fds []*os.File

	switch expectedLink.Type() {
	case (&netlink.Bridge{}).Type():
		newLink = &netlink.Bridge{
			LinkAttrs:         netlink.LinkAttrs{Name: name},
			MulticastSnooping: expectedLink.(*netlink.Bridge).MulticastSnooping,
		}
	case (&netlink.Tuntap{}).Type():
		flags := netlink.TUNTAP_VNET_HDR
		if queues > 0 {
			flags |= netlink.TUNTAP_MULTI_QUEUE_DEFAULTS
		}
		newLink = &netlink.Tuntap{
			LinkAttrs: netlink.LinkAttrs{Name: name},
			Mode:      netlink.TUNTAP_MODE_TAP,
			Queues:    queues,
			Flags:     flags,
		}
	case (&netlink.Macvtap{}).Type():
		qlen := expectedLink.Attrs().TxQLen
		if qlen <= 0 {
			qlen = types.DefaultQlen
		}
		newLink = &netlink.Macvtap{
			Macvlan: netlink.Macvlan{
				Mode: netlink.MACVLAN_MODE_BRIDGE,
				LinkAttrs: netlink.LinkAttrs{
					Index:       expectedLink.Attrs().Index,
					Name:        name,
					TxQLen:      qlen,
					ParentIndex: expectedLink.Attrs().ParentIndex,
				},
			},
		}
	default:
		return nil, fds, fmt.Errorf("Unsupported link type %s", expectedLink.Type())
	}

	if err := netHandle.LinkAdd(newLink); err != nil {
		return nil, fds, fmt.Errorf("LinkAdd() failed for %s name %s: %s", expectedLink.Type(), name, err)
	}

	tuntapLink, ok := newLink.(*netlink.Tuntap)
	if ok {
		fds = tuntapLink.Fds
	}

	newLink, err := getLinkByName(netHandle, name, expectedLink)
	return newLink, fds, err
}

const hostLinkOffset = 8192 // Host should not have more than 8k interfaces
const linkRange = 0xFFFF    // This will allow upto 2^16 containers
const linkRetries = 128     // The numbers of time we try to find a non conflicting index
const macvtapWorkaround = true

func createMacVtap(netHandle *netlink.Handle, name string, link netlink.Link, queues int) (taplink netlink.Link, err error) {
	if !macvtapWorkaround {
		taplink, _, err = createLink(netHandle, name, link, queues)
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < linkRetries; i++ {
		index := hostLinkOffset + (r.Int() & linkRange)
		link.Attrs().Index = index
		taplink, _, err = createLink(netHandle, name, link, queues)
		if err == nil {
			break
		}
	}

	return
}

func clearIPs(link netlink.Link, addrs []netlink.Addr) error {
	for _, addr := range addrs {
		if err := netlink.AddrDel(link, &addr); err != nil {
			return err
		}
	}
	return nil
}

func setIPs(link netlink.Link, addrs []netlink.Addr) error {
	for _, addr := range addrs {
		if err := netlink.AddrAdd(link, &addr); err != nil {
			return err
		}
	}
	return nil
}

func createFds(device string, numFds int) ([]*os.File, error) {
	fds := make([]*os.File, numFds)

	for i := 0; i < numFds; i++ {
		f, err := os.OpenFile(device, os.O_RDWR, types.DefaultFilePerms)
		if err != nil {
			utils.CleanupFds(fds, i)
			return nil, err
		}
		fds[i] = f
	}
	return fds, nil
}

func createMacvtapFds(linkIndex int, queues int) ([]*os.File, error) {
	tapDev := fmt.Sprintf("/dev/tap%d", linkIndex)
	return createFds(tapDev, queues)
}

func createVhostFds(numFds int) ([]*os.File, error) {
	vhostDev := "/dev/vhost-net"
	return createFds(vhostDev, numFds)
}

func getLinkByName(netHandle *netlink.Handle, name string, expectedLink netlink.Link) (netlink.Link, error) {
	link, err := netHandle.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("LinkByName() failed for %s name %s: %s", expectedLink.Type(), name, err)
	}

	switch expectedLink.Type() {
	case (&netlink.Bridge{}).Type():
		if l, ok := link.(*netlink.Bridge); ok {
			return l, nil
		}
	case (&netlink.Tuntap{}).Type():
		if l, ok := link.(*netlink.GenericLink); ok {
			return l, nil
		}
	case (&netlink.Veth{}).Type():
		if l, ok := link.(*netlink.Veth); ok {
			return l, nil
		}
	case (&netlink.Macvtap{}).Type():
		if l, ok := link.(*netlink.Macvtap); ok {
			return l, nil
		}
	case (&netlink.Macvlan{}).Type():
		if l, ok := link.(*netlink.Macvlan); ok {
			return l, nil
		}
	case (&netlink.IPVlan{}).Type():
		if l, ok := link.(*netlink.IPVlan); ok {
			return l, nil
		}
	default:
		return nil, fmt.Errorf("Unsupported link type %s", expectedLink.Type())
	}

	return nil, fmt.Errorf("Incorrect link type %s, expecting %s", link.Type(), expectedLink.Type())
}

// addQdiscIngress creates a new qdisc for nwtwork interface with the specified network index
// on "ingress". qdiscs normally don't work on ingress so this is really a special qdisc
// that you can consider an "alternate root" for inbound packets.
// Handle for ingress qdisc defaults to "ffff:"
//
// This is equivalent to calling `tc qdisc add dev eth0 ingress`
func addQdiscIngress(index int) error {
	qdisc := &netlink.Ingress{
		QdiscAttrs: netlink.QdiscAttrs{
			LinkIndex: index,
			Parent:    netlink.HANDLE_INGRESS,
		},
	}

	err := netlink.QdiscAdd(qdisc)
	if err != nil {
		return fmt.Errorf("Failed to add qdisc for network index %d : %s", index, err)
	}

	return nil
}

// addRedirectTCFilter adds a tc filter for device with index "sourceIndex".
// All traffic for interface with index "sourceIndex" is redirected to interface with
// index "destIndex"
//
// This is equivalent to calling:
// `tc filter add dev source parent ffff: protocol all u32 match u8 0 0 action mirred egress redirect dev dest`
func addRedirectTCFilter(sourceIndex, destIndex int) error {
	filter := &netlink.U32{
		FilterAttrs: netlink.FilterAttrs{
			LinkIndex: sourceIndex,
			Parent:    netlink.MakeHandle(0xffff, 0),
			Protocol:  unix.ETH_P_ALL,
		},
		Actions: []netlink.Action{
			&netlink.MirredAction{
				ActionAttrs: netlink.ActionAttrs{
					Action: netlink.TC_ACT_STOLEN,
				},
				MirredAction: netlink.TCA_EGRESS_REDIR,
				Ifindex:      destIndex,
			},
		},
	}

	if err := netlink.FilterAdd(filter); err != nil {
		return fmt.Errorf("Failed to add filter for index %d : %s", sourceIndex, err)
	}

	return nil
}

// removeRedirectTCFilter removes all tc u32 filters created on ingress qdisc for "link".
func removeRedirectTCFilter(link netlink.Link) error {
	if link == nil {
		return nil
	}

	// Handle 0xffff is used for ingress
	filters, err := netlink.FilterList(link, netlink.MakeHandle(0xffff, 0))
	if err != nil {
		return err
	}

	for _, f := range filters {
		u32, ok := f.(*netlink.U32)

		if !ok {
			continue
		}

		if err := netlink.FilterDel(u32); err != nil {
			return err
		}
	}
	return nil
}

// removeQdiscIngress removes the ingress qdisc previously created on "link".
func removeQdiscIngress(link netlink.Link) error {
	if link == nil {
		return nil
	}

	qdiscs, err := netlink.QdiscList(link)
	if err != nil {
		return err
	}

	for _, qdisc := range qdiscs {
		ingress, ok := qdisc.(*netlink.Ingress)
		if !ok {
			continue
		}

		if err := netlink.QdiscDel(ingress); err != nil {
			return err
		}
	}
	return nil
}

func generateRandomPrivateMacAddr() (string, error) {
	buf := make([]byte, 6)
	_, err := cryptoRand.Read(buf)
	if err != nil {
		return "", err
	}

	// Set the local bit for local addresses
	// Addresses in this range are local mac addresses:
	// x2-xx-xx-xx-xx-xx , x6-xx-xx-xx-xx-xx , xA-xx-xx-xx-xx-xx , xE-xx-xx-xx-xx-xx
	buf[0] = (buf[0] | 2) & 0xfe

	hardAddr := net.HardwareAddr(buf)
	return hardAddr.String(), nil
}

func createNetworkInterfacePair(idx int, ifName string, interworkingModel types.NetInterworkingModel) (types.NetworkInterfacePair, error) {
	uniqueID := uuid.Generate().String()

	randomMacAddr, err := generateRandomPrivateMacAddr()
	if err != nil {
		return types.NetworkInterfacePair{}, fmt.Errorf("Could not generate random mac address: %s", err)
	}

	netPair := types.NetworkInterfacePair{
		TapInterface: types.TapInterface{
			ID:   uniqueID,
			Name: fmt.Sprintf("br%d_kata", idx),
			TAPIface: types.NetworkInterface{
				Name: fmt.Sprintf("tap%d_kata", idx),
			},
		},
		VirtIface: types.NetworkInterface{
			Name:     fmt.Sprintf("eth%d", idx),
			HardAddr: randomMacAddr,
		},
		NetInterworkingModel: interworkingModel,
	}

	return netPair, nil
}

func networkInfoFromLink(handle *netlink.Handle, link netlink.Link) (types.NetworkInfo, error) {
	addrs, err := handle.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		return types.NetworkInfo{}, err
	}

	routes, err := handle.RouteList(link, netlink.FAMILY_ALL)
	if err != nil {
		return types.NetworkInfo{}, err
	}

	return types.NetworkInfo{
		Iface: types.NetlinkIface{
			LinkAttrs: *(link.Attrs()),
			Type:      link.Type(),
		},
		Addrs:  addrs,
		Routes: routes,
	}, nil
}
