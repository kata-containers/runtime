// Copyright (c) 2018 HyperHQ Inc.
//
// SPDX-License-Identifier: Apache-2.0
//

package shim

// KataBuiltInShim is the structure type of kata builtin shim
type KataBuiltInShim struct{}

// Start is the KataBuiltInShim start implementation for kata builtin shim.
// It does nothing. The shim functionality is provided by the virtcontainers
// library.
func (s *KataBuiltInShim) Start(shimType Type, shimConfig interface{}, params Params) (int, error) {
	return -1, nil
}
