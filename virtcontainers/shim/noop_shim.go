// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package shim

// NoopShim is the structure type of test shim
type NoopShim struct{}

// Start is the noopShim start implementation for testing purpose.
// It does nothing.
func (s *NoopShim) Start(shimType Type, shimConfig interface{}, params Params) (int, error) {
	return 0, nil
}
