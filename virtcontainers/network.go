// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/containernetworking/plugins/pkg/ns"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"

	"github.com/kata-containers/runtime/virtcontainers/hypervisor"
	vcTypes "github.com/kata-containers/runtime/virtcontainers/pkg/types"
	"github.com/kata-containers/runtime/virtcontainers/types"
)

func networkLogger() *logrus.Entry {
	return virtLog.WithField("subsystem", "network")
}

func createNetNS() (string, error) {
	n, err := ns.NewNS()
	if err != nil {
		return "", err
	}

	return n.Path(), nil
}

func deleteNetNS(netNSPath string) error {
	n, err := ns.GetNS(netNSPath)
	if err != nil {
		return err
	}

	err = n.Close()
	if err != nil {
		return err
	}

	if err = unix.Unmount(netNSPath, unix.MNT_DETACH); err != nil {
		return fmt.Errorf("Failed to unmount namespace %s: %v", netNSPath, err)
	}
	if err := os.RemoveAll(netNSPath); err != nil {
		return fmt.Errorf("Failed to clean up namespace %s: %v", netNSPath, err)
	}

	return nil
}

// NetworkNamespace contains all data related to its network namespace.
type NetworkNamespace struct {
	NetNsPath    string
	NetNsCreated bool
	Endpoints    []hypervisor.Endpoint
	NetmonPID    int
}

// MarshalJSON is the custom NetworkNamespace JSON marshalling routine.
// This is needed to properly marshall Endpoints array.
func (n NetworkNamespace) MarshalJSON() ([]byte, error) {
	// We need a shadow structure in order to prevent json from
	// entering a recursive loop when only calling json.Marshal().
	type shadow struct {
		NetNsPath    string
		NetNsCreated bool
		Endpoints    []hypervisor.TypedJSONEndpoint
	}

	s := &shadow{
		NetNsPath:    n.NetNsPath,
		NetNsCreated: n.NetNsCreated,
	}

	var typedEndpoints []hypervisor.TypedJSONEndpoint
	for _, ep := range n.Endpoints {
		tempJSON, _ := json.Marshal(ep)

		t := hypervisor.TypedJSONEndpoint{
			Type: ep.Type(),
			Data: tempJSON,
		}

		typedEndpoints = append(typedEndpoints, t)
	}

	s.Endpoints = typedEndpoints

	b, err := json.Marshal(s)
	return b, err
}

// UnmarshalJSON is the custom NetworkNamespace unmarshalling routine.
// This is needed for unmarshalling the Endpoints interfaces array.
func (n *NetworkNamespace) UnmarshalJSON(b []byte) error {
	var s struct {
		NetNsPath    string
		NetNsCreated bool
		Endpoints    json.RawMessage
	}

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	(*n).NetNsPath = s.NetNsPath
	(*n).NetNsCreated = s.NetNsCreated

	var typedEndpoints []hypervisor.TypedJSONEndpoint
	if err := json.Unmarshal([]byte(string(s.Endpoints)), &typedEndpoints); err != nil {
		return err
	}
	endpoints, err := hypervisor.UnmarshalEndpoints(typedEndpoints)
	if err != nil {
		return err
	}

	(*n).Endpoints = endpoints
	return nil
}

func (n NetworkNamespace) interfacesAndRoutes(networkNS NetworkNamespace) ([]*vcTypes.Interface, []*vcTypes.Route, error) {

	if n.NetNsPath == "" {
		return nil, nil, nil
	}

	var routes []*vcTypes.Route
	var ifaces []*vcTypes.Interface

	for _, endpoint := range n.Endpoints {

		var ipAddresses []*vcTypes.IPAddress
		for _, addr := range endpoint.Properties().Addrs {
			// Skip IPv6 because not supported
			if addr.IP.To4() == nil {
				// Skip IPv6 because not supported
				networkLogger().WithFields(logrus.Fields{
					"unsupported-address-type": "ipv6",
					"address":                  addr,
				}).Warn("unsupported address")
				continue
			}
			// Skip localhost interface
			if addr.IP.IsLoopback() {
				continue
			}
			netMask, _ := addr.Mask.Size()
			ipAddress := vcTypes.IPAddress{
				Family:  netlink.FAMILY_V4,
				Address: addr.IP.String(),
				Mask:    fmt.Sprintf("%d", netMask),
			}
			ipAddresses = append(ipAddresses, &ipAddress)
		}
		ifc := vcTypes.Interface{
			IPAddresses: ipAddresses,
			Device:      endpoint.Name(),
			Name:        endpoint.Name(),
			Mtu:         uint64(endpoint.Properties().Iface.MTU),
			HwAddr:      endpoint.HardwareAddr(),
			PciAddr:     endpoint.PciAddr(),
		}

		ifaces = append(ifaces, &ifc)

		for _, route := range endpoint.Properties().Routes {
			var r vcTypes.Route

			if route.Dst != nil {
				r.Dest = route.Dst.String()

				if route.Dst.IP.To4() == nil {
					// Skip IPv6 because not supported
					networkLogger().WithFields(logrus.Fields{
						"unsupported-route-type": "ipv6",
						"destination":            r.Dest,
					}).Warn("unsupported route")
					continue
				}
			}

			if route.Gw != nil {
				gateway := route.Gw.String()

				if route.Gw.To4() == nil {
					// Skip IPv6 because is is not supported
					networkLogger().WithFields(logrus.Fields{
						"unsupported-route-type": "ipv6",
						"gateway":                gateway,
					}).Warn("unsupported route")
					continue
				}
				r.Gateway = gateway
			}

			if route.Src != nil {
				r.Source = route.Src.String()
			}

			r.Device = endpoint.Name()
			r.Scope = uint32(route.Scope)
			routes = append(routes, &r)

		}
	}
	return ifaces, routes, nil
}

// Network is the virtcontainer network structure
type Network struct {
}

func (n *Network) trace(ctx context.Context, name string) (opentracing.Span, context.Context) {
	span, ct := opentracing.StartSpanFromContext(ctx, name)

	span.SetTag("subsystem", "network")
	span.SetTag("type", "default")

	return span, ct
}

// Run runs a callback in the specified network namespace.
func (n *Network) Run(networkNSPath string, cb func() error) error {
	span, _ := n.trace(context.Background(), "run")
	defer span.Finish()

	return hypervisor.DoNetNS(networkNSPath, func(_ ns.NetNS) error {
		return cb()
	})
}

// Add adds all needed interfaces inside the network namespace.
func (n *Network) Add(ctx context.Context, config *types.NetworkConfig, h hypervisor.Hypervisor, hotplug bool) ([]hypervisor.Endpoint, error) {
	span, _ := n.trace(ctx, "add")
	defer span.Finish()

	endpoints, err := hypervisor.CreateEndpointsFromScan(config.NetNSPath, config)
	if err != nil {
		return endpoints, err
	}

	err = hypervisor.DoNetNS(config.NetNSPath, func(_ ns.NetNS) error {
		for _, endpoint := range endpoints {
			networkLogger().WithField("endpoint-type", endpoint.Type()).WithField("hotplug", hotplug).Info("Attaching endpoint")
			if hotplug {
				if err := endpoint.HotAttach(h); err != nil {
					return err
				}
			} else {
				if err := endpoint.Attach(h); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if err != nil {
		return []hypervisor.Endpoint{}, err
	}

	networkLogger().Debug("Network added")

	return endpoints, nil
}

// Remove network endpoints in the network namespace. It also deletes the network
// namespace in case the namespace has been created by us.
func (n *Network) Remove(ctx context.Context, ns *NetworkNamespace, h hypervisor.Hypervisor, hotunplug bool) error {
	span, _ := n.trace(ctx, "remove")
	defer span.Finish()

	for _, endpoint := range ns.Endpoints {
		// Detach for an endpoint should enter the network namespace
		// if required.
		networkLogger().WithField("endpoint-type", endpoint.Type()).WithField("hotunplug", hotunplug).Info("Detaching endpoint")
		if hotunplug {
			if err := endpoint.HotDetach(h, ns.NetNsCreated, ns.NetNsPath); err != nil {
				return err
			}
		} else {
			if err := endpoint.Detach(ns.NetNsCreated, ns.NetNsPath); err != nil {
				return err
			}
		}
	}

	networkLogger().Debug("Network removed")

	if ns.NetNsCreated {
		networkLogger().Infof("Network namespace %q deleted", ns.NetNsPath)
		return deleteNetNS(ns.NetNsPath)
	}

	return nil
}
