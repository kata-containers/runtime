// Copyright (c) 2019 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0
//

package types

import (
	"testing"
)

func TestNetInterworkingModelIsValid(t *testing.T) {
	tests := []struct {
		name string
		n    NetInterworkingModel
		want bool
	}{
		{"Invalid Model", NetXConnectInvalidModel, false},
		{"Default Model", NetXConnectDefaultModel, true},
		{"Bridged Model", NetXConnectBridgedModel, true},
		{"TC Filter Model", NetXConnectTCFilterModel, true},
		{"Macvtap Model", NetXConnectMacVtapModel, true},
		{"Enlightened Model", NetXConnectEnlightenedModel, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.n.IsValid(); got != tt.want {
				t.Errorf("NetInterworkingModel.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNetInterworkingModelSetModel(t *testing.T) {
	var n NetInterworkingModel
	tests := []struct {
		name      string
		modelName string
		wantErr   bool
	}{
		{"Invalid Model", "Invalid", true},
		{"default Model", DefaultNetModelStr, false},
		{"bridged Model", BridgedNetModelStr, false},
		{"macvtap Model", MacvtapNetModelStr, false},
		{"enlightened Model", EnlightenedNetModelStr, false},
		{"tcfilter Model", TcFilterNetModelStr, false},
		{"none Model", NoneNetModelStr, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := n.SetModel(tt.modelName); (err != nil) != tt.wantErr {
				t.Errorf("NetInterworkingModel.SetModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
