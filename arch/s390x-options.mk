# Copyright (c) 2018 Yash Jain
# #
# # SPDX-License-Identifier: Apache-2.0
# #
# # IBM Z mainframe s390x settings

MACHINETYPE := s390-ccw-virtio 
KERNELPARAMS :=
MACHINEACCELERATORS :=
KERNELTYPE := uncompressed # Not sure ablout this
QEMUCMD := qemu-system-s390x
