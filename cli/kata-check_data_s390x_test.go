// Copyright (c) 2018 Yash Jain
//
// SPDX-License-Identifier: Apache-2.0
//
package main

const testCPUInfoTemplate = `
vendor_id       : IBM/S390
# processors    : 2
bogomips per cpu: 20325.00
max thread id   : 0
features	: esan3 zarch stfle msa ldisp eimm dfp etf3eh highgprs vx sie
facilities      : 0 1 2 3 4 6 7 9 10 12 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 30 31 32 33 34 35 36 37 40 41 42 43 44 45 46 47 48 49 50 51 52 53 55 57 74 75 76 77 80 81 82 128 129 1024 1025 1026 1027 1028 1030 1031 1033 1034 1036 1038 1039 1040 1041 1042 1043 1044 1045 1046 1047 1048 1049 1050 1051 1052 1054 1055 1056 1057 1058 1059 1060 1061 1064 1065 1066 1067 1068 1069 1070 1071 1072 1073 1074 1075 1076 1077 1079 1081 1098 1099 1100 1101 1104 1105 1106 1152 1153
cache0          : level=1 type=Data scope=Private size=128K line_size=256 associativity=8
cache1          : level=1 type=Instruction scope=Private size=96K line_size=256 associativity=6
cache2          : level=2 type=Data scope=Private size=2048K line_size=256 associativity=8
cache3          : level=2 type=Instruction scope=Private size=2048K line_size=256 associativity=8
cache4          : level=3 type=Unified scope=Shared size=65536K line_size=256 associativity=16
cache5          : level=4 type=Unified scope=Shared size=491520K line_size=256 associativity=30
processor 0: version = FF,  identification = 096A77,  machine = 2964
processor 1: version = FF,  identification = 096A77,  machine = 2964
`
