module github.com/kata-containers/runtime

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Microsoft/go-winio v0.4.12 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/blang/semver v3.5.1+incompatible
	github.com/cilium/ebpf v0.0.0-20200106160548-c8f8abaa9ece // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/containerd/cgroups v0.0.0-20190717030353-c4b9ac5c7601
	github.com/containerd/console v0.0.0-20191206165004-02ecf6a7291e
	github.com/containerd/containerd v1.2.1-0.20181210191522-f05672357f56
	github.com/containerd/continuity v0.0.0-20200228182428-0f16d7a0959c // indirect
	github.com/containerd/cri-containerd v1.11.1-0.20190125013620-4dd6735020f5
	github.com/containerd/fifo v0.0.0-20190226154929-a9fb20d87448
	github.com/containerd/go-runc v0.0.0-20200220073739-7016d3ce2328 // indirect
	github.com/containerd/ttrpc v0.0.0-20190828172938-92c8520ef9f8 // indirect
	github.com/containerd/typeurl v0.0.0-20190228175220-2a93cfde8c20
	github.com/containernetworking/cni v0.7.1 // indirect
	github.com/containernetworking/plugins v0.8.2
	github.com/cri-o/cri-o v1.0.0-rc2.0.20170928185954-3394b3b2d6af
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/dlespiau/covertool v0.0.0-20180314162135-b0c4c6d0583a
	github.com/docker/go-units v0.3.3
	github.com/go-ini/ini v1.28.2
	github.com/go-openapi/analysis v0.18.0 // indirect
	github.com/go-openapi/errors v0.18.0
	github.com/go-openapi/jsonpointer v0.18.0 // indirect
	github.com/go-openapi/jsonreference v0.18.0 // indirect
	github.com/go-openapi/loads v0.18.0 // indirect
	github.com/go-openapi/runtime v0.18.0
	github.com/go-openapi/spec v0.18.0 // indirect
	github.com/go-openapi/strfmt v0.18.0
	github.com/go-openapi/swag v0.18.0
	github.com/go-openapi/validate v0.18.0
	github.com/gogo/protobuf v1.2.0
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/intel/govmm v0.0.0-20200221075853-3700c55dd766
	github.com/kata-containers/agent v0.0.0-20200220202609-d26a505efd33
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/mailru/easyjson v0.0.0-20190221075403-6243d8e04c3f // indirect
	github.com/mdlayher/vsock v0.0.0-20190429153235-7b7533a7ca4e // indirect
	github.com/mitchellh/mapstructure v1.1.2
	github.com/opencontainers/runc v1.0.0-rc10
	github.com/opencontainers/runtime-spec v1.0.2-0.20190408193819-a1b50f621a48
	github.com/opencontainers/selinux v1.4.0 // indirect
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pkg/errors v0.8.1
	github.com/prometheus/procfs v0.0.0-20190328153300-af7bedc223fb
	github.com/safchain/ethtool v0.0.0-20190326074333-42ed695e3de8
	github.com/seccomp/libseccomp-golang v0.9.0 // indirect
	github.com/sirupsen/logrus v1.4.2
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/stretchr/testify v1.4.0
	github.com/uber-go/atomic v0.0.0-00010101000000-000000000000 // indirect
	github.com/uber/jaeger-client-go v2.15.0+incompatible
	github.com/uber/jaeger-lib v1.5.0 // indirect
	github.com/urfave/cli v1.20.1-0.20170926034118-ac249472b7de
	github.com/vishvananda/netlink v1.0.1-0.20190604022042-c8c507c80ea2
	github.com/vishvananda/netns v0.0.0-20180720170159-13995c7128cc
	go.uber.org/atomic v1.6.0 // indirect
	golang.org/x/net v0.0.0-20190620200207-3b0461eec859
	golang.org/x/oauth2 v0.0.0-20191122200657-5d9234df094c
	golang.org/x/sys v0.0.0-20200106162015-b016eb3dc98e
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20190307195333-5fe7a883aa19 // indirect
	google.golang.org/grpc v1.19.0
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/uber-go/atomic => go.uber.org/atomic v1.6.1-0.20200224215847-b2c105d12ef6
