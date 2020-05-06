# Copyright (c) 2020 Red Hat Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
# Build the image which wraps the kata-containers installation along with the
# install script. It is used on a daemonset to deploy kata on OpenShift.
#
FROM centos:7

RUN yum install -y rsync

# Load the installation files.
COPY ./_out ./_out

COPY ./entrypoint.sh /usr/bin

ENTRYPOINT /usr/bin/entrypoint.sh
