// Copyright (c) 2016 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"fmt"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/sirupsen/logrus"
)

type defNetwork struct {
}

func defNetworkLogger() *logrus.Entry {
	return virtLog.WithField("subsystem", "default-network")
}

// init initializes the network, setting a new network namespace for the default network.
func (n *defNetwork) init(config NetworkConfig) (string, bool, error) {
	if !config.InterworkingModel.IsValid() || config.InterworkingModel == NetXConnectDefaultModel {
		config.InterworkingModel = DefaultNetInterworkingModel
	}

	if config.NetNSPath == "" {
		path, err := createNetNS()
		if err != nil {
			return "", false, err
		}

		return path, true, nil
	}

	isHostNs, err := hostNetworkingRequested(config.NetNSPath)
	if err != nil {
		return "", false, err
	}

	if isHostNs {
		return "", false, fmt.Errorf("Host networking requested, not supported by runtime")
	}

	return config.NetNSPath, false, nil
}

// run runs a callback in the specified network namespace.
func (n *defNetwork) run(networkNSPath string, cb func() error) error {
	if networkNSPath == "" {
		return fmt.Errorf("networkNSPath cannot be empty")
	}

	return doNetNS(networkNSPath, func(_ ns.NetNS) error {
		return cb()
	})
}

// add adds all needed interfaces inside the network namespace for the CNM network.
func (n *defNetwork) add(sandbox *Sandbox, config NetworkConfig, netNsPath string, netNsCreated bool) (NetworkNamespace, error) {
	endpoints, err := createEndpointsFromScan(netNsPath, config)
	if err != nil {
		return NetworkNamespace{}, err
	}

	networkNS := NetworkNamespace{
		NetNsPath:    netNsPath,
		NetNsCreated: netNsCreated,
		Endpoints:    endpoints,
	}

	err = doNetNS(networkNS.NetNsPath, func(_ ns.NetNS) error {
		for _, endpoint := range networkNS.Endpoints {
			if err := endpoint.Attach(sandbox.hypervisor); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return NetworkNamespace{}, err
	}

	defNetworkLogger().Debug("Network added")

	return networkNS, nil
}

// remove network endpoints in the network namespace. It also deletes the network
// namespace in case the namespace has been created by us.
func (n *defNetwork) remove(sandbox *Sandbox, networkNS NetworkNamespace, netNsCreated bool) error {
	for _, endpoint := range networkNS.Endpoints {
		// Detach for an endpoint should enter the network namespace
		// if required.
		if err := endpoint.Detach(netNsCreated, networkNS.NetNsPath); err != nil {
			return err
		}
	}

	defNetworkLogger().Debug("Network removed")

	if netNsCreated {
		defNetworkLogger().Infof("Network namespace %q deleted", networkNS.NetNsPath)
		return deleteNetNS(networkNS.NetNsPath)
	}

	return nil
}
