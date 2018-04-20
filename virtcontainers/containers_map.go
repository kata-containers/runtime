// Copyright (c) 2018 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package virtcontainers

import (
	"fmt"
	"os"
	"syscall"
)

// rLockSandboxes locks sandboxes with a shared lock.
func rLockSandboxes() (*os.File, error) {
	return lockSandboxes(sharedLock)
}

// rwLockSandboxes locks sandboxes with an exclusive lock.
func rwLockSandboxes() (*os.File, error) {
	return lockSandboxes(exclusiveLock)
}

// lock takes a lock across all sandboxes.
func lockSandboxes(lockType int) (*os.File, error) {
	fs := filesystem{}
	sandboxesLockFile, parentDir, err := fs.globalSandboxURI(lockFileType)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		if err := os.MkdirAll(parentDir, 0750); err != nil {
			return nil, err
		}
	}

	lockFile, err := os.OpenFile(sandboxesLockFile, os.O_RDONLY|os.O_CREATE, 0640)
	if err != nil {
		return nil, err
	}

	if err := syscall.Flock(int(lockFile.Fd()), lockType); err != nil {
		return nil, err
	}

	return lockFile, nil
}

// unlock releases the lock taken across all sandboxes.
func unlockSandboxes(lockFile *os.File) error {
	if lockFile == nil {
		return fmt.Errorf("lockFile cannot be empty")
	}

	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
		return err
	}

	lockFile.Close()

	return nil
}

// findSliceIndexStr returns the slice index where the value matches
// the provided string.
// Returns -1 if it could not find the string.
func findSliceIndexStr(slice []string, val string) int {
	for idx, elem := range slice {
		if elem == val {
			return idx
		}
	}

	return -1
}

// fetchContainersMap retrieves the containers map from the storage.
func fetchContainersMap() (map[string][]string, error) {
	fs := &filesystem{}
	ctrsMap := make(map[string][]string)

	lockFile, err := rLockSandboxes()
	if err != nil {
		return nil, err
	}
	defer unlockSandboxes(lockFile)

	if err := fs.fetchContainersMap(&ctrsMap); err != nil {
		return nil, err
	}

	return ctrsMap, nil
}

// addToContainersMap adds a pair containerID/sandboxID to the map and stores
// the result.
func addToContainersMap(containerID, sandboxID string) error {
	if containerID == "" {
		return errNeedContainerID
	}
	if sandboxID == "" {
		return errNeedSandboxID
	}

	fs := &filesystem{}
	ctrsMap := make(map[string][]string)

	lockFile, err := rwLockSandboxes()
	if err != nil {
		return err
	}
	defer unlockSandboxes(lockFile)

	if err := fs.fetchContainersMap(&ctrsMap); err != nil {
		return err
	}

	idx := findSliceIndexStr(ctrsMap[containerID], sandboxID)
	if idx != -1 {
		return nil
	}

	ctrsMap[containerID] = append(ctrsMap[containerID], sandboxID)

	return fs.storeContainersMap(ctrsMap)
}

// delFromContainersMap removes a pair containerID/sandboxID from the map and
// stores the result.
func delFromContainersMap(containerID, sandboxID string) error {
	if containerID == "" {
		return errNeedContainerID
	}
	if sandboxID == "" {
		return errNeedSandboxID
	}

	fs := &filesystem{}
	ctrsMap := make(map[string][]string)

	lockFile, err := rwLockSandboxes()
	if err != nil {
		return err
	}
	defer unlockSandboxes(lockFile)

	if err := fs.fetchContainersMap(&ctrsMap); err != nil {
		return err
	}

	idx := findSliceIndexStr(ctrsMap[containerID], sandboxID)
	if idx == -1 {
		return nil
	}

	ctrsMap[containerID] = append(ctrsMap[containerID][:idx], ctrsMap[containerID][idx+1:]...)

	if len(ctrsMap[containerID]) == 0 {
		delete(ctrsMap, containerID)
	}

	return fs.storeContainersMap(ctrsMap)
}
