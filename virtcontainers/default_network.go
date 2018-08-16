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

// add adds all needed interfaces inside the network namespace, and save them
// in memory, into the sandbox structure.
func (n *defNetwork) add(s *Sandbox) error {
	endpoints, err := createEndpointsFromScan(s.networkNS.NetNsPath, s.config.NetworkConfig)
	if err != nil {
		return err
	}

	s.networkNS.Endpoints = endpoints

	return doNetNS(s.networkNS.NetNsPath, func(_ ns.NetNS) error {
		for _, endpoint := range s.networkNS.Endpoints {
			if err := endpoint.HotAttach(s.hypervisor); err != nil {
				return err
			}
		}

		defNetworkLogger().Debug("Network added")

		return nil
	})
}

// remove network endpoints in the network namespace. It also deletes the network
// namespace in case the namespace has been created by us.
func (n *defNetwork) remove(s *Sandbox) error {
	for _, endpoint := range s.networkNS.Endpoints {
		// Detach for an endpoint should enter the network namespace
		// if required.
		if err := endpoint.Detach(s.networkNS.NetNsCreated, s.networkNS.NetNsPath); err != nil {
			return err
		}
	}

	defNetworkLogger().Debug("Network removed")

	if s.networkNS.NetNsCreated {
		defNetworkLogger().Infof("Network namespace %q deleted", s.networkNS.NetNsPath)
		return deleteNetNS(s.networkNS.NetNsPath)
	}

	return nil
}
