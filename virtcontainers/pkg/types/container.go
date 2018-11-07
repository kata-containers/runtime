// Copyright 2018 Intel Corporation.
//
// SPDX-License-Identifier: Apache-2.0
//

package types

import (
	"time"
)

// Process gathers data related to a container process.
type Process struct {
	// Token is the process execution context ID. It must be
	// unique per sandbox.
	// Token is used to manipulate processes for containers
	// that have not started yet, and later identify them
	// uniquely within a sandbox.
	Token string

	// Pid is the process ID as seen by the host software
	// stack, e.g. CRI-O, containerd. This is typically the
	// shim PID.
	Pid int

	StartTime time.Time
}
