//
// Copyright (c) 2018 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package virtcontainers

import (
	"fmt"
	"plugin"
)

var (
	getHypervisorPluginFuncName = "GetHypervisorPlugin"
)

type pluginHypervisor struct {
	HypervisorPlugin
}

func (h *pluginHypervisor) init(pod *Pod) error {
	plugIn, err := plugin.Open(pod.config.HypervisorConfig.PluginPath)
	if err != nil {
		return fmt.Errorf("Failed to open plugin path %q: %v",
			pod.config.HypervisorConfig.PluginPath, err)
	}

	getHypervisorPluginFunc, err := plugIn.Lookup(getHypervisorPluginFuncName)
	if err != nil {
		return fmt.Errorf("Failed to lookup function %q: %v",
			getHypervisorPluginFuncName, err)
	}

	h.HypervisorPlugin = getHypervisorPluginFunc.(func() HypervisorPlugin)()
	if err != nil {
		return fmt.Errorf("%s() failed: %v",
			getHypervisorPluginFuncName, err)
	}

	return h.HypervisorPlugin.Init(pod)
}

func (h *pluginHypervisor) createPod(podConfig PodConfig) error {
	return h.HypervisorPlugin.CreatePod(podConfig)
}

func (h *pluginHypervisor) startPod() error {
	return h.HypervisorPlugin.StartPod()
}

func (h *pluginHypervisor) waitPod(timeout int) error {
	return h.HypervisorPlugin.WaitPod(timeout)
}

func (h *pluginHypervisor) stopPod() error {
	return h.HypervisorPlugin.StopPod()
}

func (h *pluginHypervisor) pausePod() error {
	return h.HypervisorPlugin.PausePod()
}

func (h *pluginHypervisor) resumePod() error {
	return h.HypervisorPlugin.ResumePod()
}

func (h *pluginHypervisor) addDevice(devInfo interface{}, devType DeviceType) error {
	return h.HypervisorPlugin.AddDevice(devInfo, devType)
}

func (h *pluginHypervisor) hotplugAddDevice(devInfo interface{}, devType DeviceType) error {
	return h.HypervisorPlugin.HotplugAddDevice(devInfo, devType)
}

func (h *pluginHypervisor) hotplugRemoveDevice(devInfo interface{}, devType DeviceType) error {
	return h.HypervisorPlugin.HotplugRemoveDevice(devInfo, devType)
}

func (h *pluginHypervisor) getPodConsole(podID string) string {
	return h.HypervisorPlugin.GetPodConsole(podID)
}

func (h *pluginHypervisor) capabilities() Capabilities {
	return h.HypervisorPlugin.Capabilities()
}
