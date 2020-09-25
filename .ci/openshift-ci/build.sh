#!/bin/bash
#
# Copyright (c) 2020 Red Hat Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
# Build and install kata-containers under <repository>/_out/build_install
#

set -e

export GOPATH=${GOPATH:-${HOME}/go}
cidir=$(dirname "$0")/..
source "${cidir}/lib.sh"

export kata_repo=$(git config --get remote.origin.url | sed -e 's#http\(s\)*://##')

mkdir -p $cidir/../_out/build_install
destdir=$(realpath $cidir/../_out/build_install)

clone_tests_repo
pushd "${tests_repo_dir}"
# Resolve the kata-container repositories. It relies on branch and
# kata_repo variables.
export branch=master
.ci/resolve-kata-dependencies.sh
.ci/openshift-ci/buildall_install.sh $destdir
popd
