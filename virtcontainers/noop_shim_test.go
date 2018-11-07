// Copyright (c) 2017 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package virtcontainers

import (
	"testing"

	vshim "github.com/kata-containers/runtime/virtcontainers/shim"
)

func TestNoopShimStart(t *testing.T) {
	s := &vshim.NoopShim{}
	sandbox := &Sandbox{
		config: &SandboxConfig{
			ShimType:   vshim.NoopShimType,
			ShimConfig: vshim.Config{},
		},
	}
	params := vshim.Params{}
	expected := 0

	pid, err := s.Start(sandbox.config.ShimType, sandbox.config.ShimConfig, params)
	if err != nil {
		t.Fatal(err)
	}

	if pid != expected {
		t.Fatalf("PID should be %d", expected)
	}
}
