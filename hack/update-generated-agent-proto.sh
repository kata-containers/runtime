#
# Copyright 2017 HyperHQ Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
protoc --proto_path=pkg/agent/api/grpc --go_out=plugins=grpc:pkg/agent/api/grpc pkg/agent/api/grpc/hyperstart.proto pkg/agent/api/grpc/oci.proto
