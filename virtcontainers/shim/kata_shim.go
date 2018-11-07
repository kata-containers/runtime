// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package shim

import (
	"fmt"
)

// KataShim is the structure type of kata shim
type KataShim struct{}

// KataShimConfig is the structure providing specific configuration
// for kataShim implementation.
type KataShimConfig struct {
	Path  string
	Debug bool
}

// Start is the KataBuiltInShim start implementation for kata shim.
// It starts the kata shim binary with URL and token flags provided by
// the proxy.
func (s *KataShim) Start(shimType Type, shimConfig interface{}, params Params) (int, error) {
	config, ok := NewShimConfig(shimType, shimConfig).(Config)
	if !ok {
		return -1, fmt.Errorf("Wrong shim config type, should be KataShimConfig type")
	}

	if config.Path == "" {
		return -1, fmt.Errorf("Shim path cannot be empty")
	}

	if params.URL == "" {
		return -1, fmt.Errorf("URL cannot be empty")
	}

	if params.Container == "" {
		return -1, fmt.Errorf("Container cannot be empty")
	}

	if params.Token == "" {
		return -1, fmt.Errorf("Process token cannot be empty")
	}

	args := []string{config.Path, "-agent", params.URL, "-container", params.Container, "-exec-id", params.Token}

	if params.Terminal {
		args = append(args, "-terminal")
	}

	if config.Debug {
		args = append(args, "-log", "debug")
	}

	if config.Trace {
		args = append(args, "-trace")
	}

	return startShim(args, params)
}
