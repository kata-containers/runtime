// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package main

/*
#cgo CFLAGS: -Wall
#include <stdlib.h>
#include <limits.h>
extern int new_namespaces(const char* namespaces_path, unsigned int len);
extern int remove_namespaces(const char* namespaces_path, unsigned int len);
extern int join_namespaces(const char* namespaces_path, unsigned int len);
int get_fs_info(char* fs, char* device, char* mount_point, char* type, char* data);
extern void init(void);
extern int close_channels(void);
void __attribute__((constructor)) c_main(void) {
       init();
}
*/
import "C"

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	spec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var ociToNs = map[spec.LinuxNamespaceType]string{
	spec.PIDNamespace:     "pid",
	spec.NetworkNamespace: "net",
	spec.MountNamespace:   "mnt",
	spec.IPCNamespace:     "ipc",
	spec.UTSNamespace:     "uts",
	spec.UserNamespace:    "user",
	spec.CgroupNamespace:  "cgroup",
}

var namespacesDirMode = os.FileMode(0600)

var namespacesPath = "/var/run/kata-containers/ns"

var newPersistentNamespacesFunc = cgoNewPersistentNamespaces
var joinNamespacesFunc = cgoJoinNamespaces
var removePersistentNamespacesFunc = cgoRemovePersistentNamespaces
var getFsInfoFunc = cgoGetFsInfo

func cgoNewPersistentNamespaces(sandboxNsPath string) int {
	nsPath := C.CString(sandboxNsPath)
	ret := C.new_namespaces(nsPath, C.uint(len(sandboxNsPath)))
	C.free(unsafe.Pointer(nsPath))
	return int(ret)
}

func cgoJoinNamespaces(containerNsPath string) int {
	nsPath := C.CString(containerNsPath)
	ret := C.join_namespaces(nsPath, C.uint(len(containerNsPath)))
	C.free(unsafe.Pointer(nsPath))
	return int(ret)
}

func cgoRemovePersistentNamespaces(sandboxNsPath string) int {
	nsPath := C.CString(sandboxNsPath)
	ret := C.remove_namespaces(nsPath, C.uint(len(sandboxNsPath)))
	C.free(unsafe.Pointer(nsPath))
	return int(ret)
}

func cgoGetFsInfo(filesystem string) (fsInfo, error) {
	var info fsInfo

	fs := C.CString(filesystem)
	device := (*C.char)(C.calloc(C.PATH_MAX, C.sizeof_char))
	mountPoint := (*C.char)(C.calloc(C.PATH_MAX, C.sizeof_char))
	fsType := (*C.char)(C.calloc(C.NAME_MAX, C.sizeof_char))
	data := (*C.char)(C.calloc(C.PATH_MAX, C.sizeof_char))

	if ret := C.get_fs_info(fs, device, mountPoint, fsType, data); ret == -1 {
		return info, fmt.Errorf("Could not get filesystem information")
	}

	info.device = C.GoString(device)
	info.mountPoint = C.GoString(mountPoint)
	info.fsType = C.GoString(fsType)
	for _, f := range strings.Split(C.GoString(data), ",") {
		switch f {
		case "relatime":
			info.flags |= syscall.MS_RELATIME
		case "noatime":
			info.flags |= syscall.MS_NOATIME
		case "rw", "r", "w":
			// syscall.Mount not support this
		default:
			info.data += f + ","
		}
	}
	info.data = strings.TrimSuffix(info.data, ",")

	C.free(unsafe.Pointer(fs))
	C.free(unsafe.Pointer(device))
	C.free(unsafe.Pointer(mountPoint))
	C.free(unsafe.Pointer(fsType))
	C.free(unsafe.Pointer(data))

	return info, nil
}

type fsInfo struct {
	device     string
	mountPoint string
	fsType     string
	flags      int
	data       string
}

func closeChannels() {
	C.close_channels()
}

func getFsInfo(filesystem string) (fsInfo, error) {
	return getFsInfoFunc(filesystem)
}

func newPersistentNamespaces(sandboxID, containerID string, namespaces []spec.LinuxNamespace) error {
	if sandboxID == "" {
		return fmt.Errorf("Missing sandbox ID")
	}

	sandboxNsPath := filepath.Join(namespacesPath, sandboxID)

	if containerID != "" && sandboxID != containerID {
		// container is part of a POD, create a symlink to the sandbox ns dir
		containerNsPath := filepath.Join(namespacesPath, containerID)
		if _, err := os.Stat(containerNsPath); err == nil {
			// container's namespaces dir already exist
			return nil
		}

		kataLog.WithFields(logrus.Fields{
			"src":  sandboxNsPath,
			"dest": containerNsPath,
		}).Debug("Creating symlink")

		if err := os.Symlink(sandboxNsPath, containerNsPath); err != nil {
			return fmt.Errorf("Could not create symlink: %v", err)
		}

		_, err := joinNamespaces(containerID)
		return err
	}

	if err := os.MkdirAll(sandboxNsPath, namespacesDirMode); err != nil {
		return fmt.Errorf("Could not create sandbox namespaces directory: %v", err)
	}

	for _, ns := range namespaces {
		if ns.Path == "" {
			continue
		}

		persistentNs := filepath.Join(sandboxNsPath, ociToNs[ns.Type])
		if _, err := os.Stat(persistentNs); err == nil {
			continue
		}

		kataLog.WithFields(
			logrus.Fields{
				"src":  ns.Path,
				"dest": persistentNs,
			}).Debugln("Creating symlink to namespace")

		if err := os.Symlink(ns.Path, persistentNs); err != nil {
			return fmt.Errorf("Could not create namespace symlink: %v", err)
		}
	}

	ret := newPersistentNamespacesFunc(sandboxNsPath)
	if ret == -1 {
		return fmt.Errorf("Could not create persistent namespaces in path: %s", sandboxNsPath)
	}

	if ret == 0 {
		// finish the process to allow the execution of
		// the process that joined the namespaces
		return cli.NewExitError("", 0)
	}

	return nil
}

// join namespaces, returns true if current process already joined namespaces,
// otherwise false.
func joinNamespaces(containerID string) (bool, error) {
	if containerID == "" {
		return false, fmt.Errorf("Missing container ID")
	}

	containerNsPath := filepath.Join(namespacesPath, containerID)
	if _, err := os.Stat(containerNsPath); err != nil {
		// ns path not exist
		return false, nil
	}

	log := kataLog.WithField("ns-path", containerNsPath)

	log.Debugln("Joining namespaces")

	ret := joinNamespacesFunc(containerNsPath)
	if ret == -1 {
		return false, fmt.Errorf("Could not join namespaces in the path: %s", containerNsPath)
	}

	if ret == 0 {
		// finish the process to allow the execution of
		// the process that joined the namespaces
		return false, cli.NewExitError("", 0)
	}

	return true, nil
}

func removePersistentNamespaces(sandboxID, containerID string) error {
	if sandboxID == "" {
		return fmt.Errorf("Missing sandbox ID")
	}

	// remove the symlink if the container is not the sandbox
	if containerID != "" && sandboxID != containerID {
		containerNsPath := filepath.Join(namespacesPath, containerID)
		if _, err := os.Stat(containerNsPath); err != nil {
			// ns path not exist
			return nil
		}

		kataLog.WithField("path", containerNsPath).Debug("Removing container namespace symlink")
		return os.Remove(containerNsPath)
	}

	sandboxNsPath := filepath.Join(namespacesPath, sandboxID)
	if _, err := os.Stat(sandboxNsPath); err != nil {
		// ns path not exist
		return nil
	}

	nsFiles, err := ioutil.ReadDir(sandboxNsPath)
	if err != nil {
		return err
	}

	for _, ns := range nsFiles {
		if ns.Mode()&os.ModeSymlink == 0 {
			// it's not a symlink
			continue
		}

		nsPath := filepath.Join(sandboxNsPath, ns.Name())
		kataLog.WithFields(
			logrus.Fields{
				"path": nsPath,
			}).Debugln("Removing namespace symlink")

		if err := os.Remove(nsPath); err != nil {
			return fmt.Errorf("Could not remove namespace symlink: %v", err)
		}
	}

	kataLog.WithField("ns-path", sandboxNsPath).Debug("Removing persistent namespaces")
	if ret := removePersistentNamespacesFunc(sandboxNsPath); ret == -1 {
		return fmt.Errorf("Could not remove persistent namespaces in the path: %s", sandboxNsPath)
	}

	// At this point ns directory MUST BE empty, otherwise fail
	return os.Remove(sandboxNsPath)
}
