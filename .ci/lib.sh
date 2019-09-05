#
# Copyright (c) 2018 Intel Corporation
#
# SPDX-License-Identifier: Apache-2.0

export tests_repo="${tests_repo:-github.com/kata-containers/tests}"

get_tests_repo()
{
	# KATA_CI_NO_NETWORK is (has to be) ignored if there is
	# no existing clone.
	if [ -d "$tests_repo_dir" -a -n "$KATA_CI_NO_NETWORK" ]
	then
		return
	fi

	mkdir -p ../tests
	#curl -fsSL https://codeload.github.com/kata-containers/tests/tar.gz/${TRAVIS_BRANCH:-master} | tar xzf - -C ../tests --strip-components=1
	curl -fsSL https://codeload.github.com/kata-containers/tests/tar.gz/master | tar xzf - -C ../tests --strip-components=1
	export tests_repo_dir="$(pwd)/../tests"
}

run_static_checks()
{
	get_tests_repo
	bash "$tests_repo_dir/.ci/static-checks.sh" "github.com/kata-containers/runtime"
}

run_go_test()
{
	get_tests_repo
	bash "$tests_repo_dir/.ci/go-test.sh"
}
