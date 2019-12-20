#!/bin/bash
#
# Copyright (c) 2018 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

set -e

cidir=$(dirname "$0")
source "${cidir}/lib.sh"

pushd "${tests_repo_dir}"
touch /tmp/serial-ch
tail -f /tmp/serial-ch &
sudo journalctl -t kata-runtime -f &
.ci/run.sh
popd
