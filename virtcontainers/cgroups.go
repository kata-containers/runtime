// +build linux
// Copyright (c) 2018 Huawei Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"encoding/json"
	"fmt"

	"github.com/kata-containers/runtime/virtcontainers/pkg/annotations"
	"github.com/opencontainers/runc/libcontainer/cgroups"
	"github.com/opencontainers/runc/libcontainer/cgroups/fs"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/specconv"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// cgroupsManager maintains cgroups implementations and config
type cgroupsManager struct {
	libcontainerManager cgroups.Manager
	libcontainerConfig  *configs.Config
}

// newManager setup cgroup manager for sandbox
func (cgm *cgroupsManager) newManager(s *Sandbox) error {
	ociConfigStr, err := s.Annotations(annotations.ConfigJSONKey)
	if err != nil {
		return err
	}

	var ociSpec specs.Spec
	if err := json.Unmarshal([]byte(ociConfigStr), &ociSpec); err != nil {
		return err
	}

	cgm.libcontainerConfig, err = specconv.CreateLibcontainerConfig(&specconv.CreateOpts{
		CgroupName:       s.id,
		UseSystemdCgroup: false,
		NoPivotRoot:      false,
		NoNewKeyring:     false,
		Spec:             &ociSpec,
	})

	if err != nil {
		return fmt.Errorf("create container config for %s failed with %s", s.id, err)
	}

	state, _ := s.storage.fetchSandboxState(s.id)
	if state.CgroupPaths == nil {
		cgm.libcontainerManager = &fs.Manager{
			Cgroups: cgm.libcontainerConfig.Cgroups,
			Paths:   nil,
		}
	} else {
		cgm.libcontainerManager = &fs.Manager{
			Cgroups: cgm.libcontainerConfig.Cgroups,
			Paths:   state.CgroupPaths,
		}
	}

	s.cgroups = *cgm

	return nil
}

// addSandbox adding shim pid to host cgroups and set the resource limitation with cgroups
func (cgm *cgroupsManager) addSandbox(s *Sandbox) error {
	shimPid := s.state.Pid
	if err := cgm.libcontainerManager.Apply(shimPid); err != nil {
		return fmt.Errorf("apply %d to host cgroups of sandbox %s failed with %s", shimPid, s.id, err)
	}

	if cgm.libcontainerConfig == nil {
		return fmt.Errorf("setup host cgroup of sandbox %s failed with libcontainerConfig is nil", s.id)
	}

	if err := cgm.libcontainerManager.Set(cgm.libcontainerConfig); err != nil {
		return fmt.Errorf("setup host cgroups of sandbox %s failed with %s", s.id, err)
	}

	s.state.CgroupPaths = cgm.libcontainerManager.GetPaths()
	s.storage.storeSandboxResource(s.id, stateFileType, s.state)

	s.Logger().Infof("set host cgroup for %v successful", s.id)
	return nil
}

// addContainer adding shim pid of container to sandbox's host cgroups
func (cgm *cgroupsManager) addContainer(c *Container) error {
	shimPid := c.process.Pid
	if err := cgm.libcontainerManager.Apply(shimPid); err != nil {
		return fmt.Errorf("apply %d of container %s to host cgroups of sandbox %s failed with %s", shimPid, c.id, c.sandboxID, err)
	}
	return nil
}

// deleteSandbox cleanup cgroup folders
func (cgm *cgroupsManager) deleteSandbox(s *Sandbox) {
	if cgm.libcontainerManager == nil {
		s.Logger().Warningf("destroying host cgroups for %s failed: libcontainerManager is nil!", s.id)
	}

	if err := cgm.libcontainerManager.Destroy(); err != nil {
		s.Logger().Warningf("destroy host cgroups for %s failed: %v", s.id, err)
	}

	return
}
