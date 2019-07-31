module github.com/kata-containers/runtime

go 1.12

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/Microsoft/go-winio v0.4.12
	github.com/Microsoft/hcsshim v0.8.6
	github.com/PuerkitoBio/purell v1.1.1
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578
	github.com/asaskevich/govalidator v0.0.0-20180720115003-f9ffefc3facf
	github.com/blang/semver v0.0.0-20190414102917-ba2c2ddd8906
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd
	github.com/containerd/cgroups v0.0.0-20180917172123-5017d4e9a9cf
	github.com/containerd/console v0.0.0-20181022165439-0650fd9eeb50
	github.com/containerd/containerd v0.0.0-20181210191522-f05672357f56
	github.com/containerd/cri-containerd v0.0.0-20190125013620-4dd6735020f5
	github.com/containerd/fifo v0.0.0-20190226154929-a9fb20d87448
	github.com/containerd/go-runc v0.0.0-20190226155025-7d11b49dc076
	github.com/containerd/ttrpc v0.0.0-20190211042230-69144327078c
	github.com/containerd/typeurl v0.0.0-20190228175220-2a93cfde8c20
	github.com/containernetworking/cni v0.6.0
	github.com/containernetworking/plugins v0.0.0-20171018223634-7f98c9461302
	github.com/coreos/go-systemd v0.0.0-20181031085051-9002847aa142
	github.com/cri-o/cri-o v0.0.0-20170928185954-3394b3b2d6af
	github.com/davecgh/go-spew v1.1.1
	github.com/dlespiau/covertool v0.0.0-20180314162135-b0c4c6d0583a
	github.com/docker/go-units v0.3.3
	github.com/globalsign/mgo v0.0.0-20181015135952-eeefdecb41b8
	github.com/go-ini/ini v1.28.2
	github.com/go-openapi/analysis v0.18.0
	github.com/go-openapi/errors v0.18.0
	github.com/go-openapi/jsonpointer v0.18.0
	github.com/go-openapi/jsonreference v0.18.0
	github.com/go-openapi/loads v0.18.0
	github.com/go-openapi/runtime v0.18.0
	github.com/go-openapi/spec v0.18.0
	github.com/go-openapi/strfmt v0.18.0
	github.com/go-openapi/swag v0.18.0
	github.com/go-openapi/validate v0.18.0
	github.com/godbus/dbus v0.0.0-20181101234600-2ff6f7ffd60f
	github.com/gogo/protobuf v0.0.0-20171007142547-342cbe0a0415
	github.com/golang/protobuf v1.3.0
	github.com/grpc-ecosystem/grpc-opentracing v0.0.0-20180507213350-8e809c8a8645
	github.com/hashicorp/errwrap v1.0.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d
	github.com/hpcloud/tail v1.0.0 // indirect
	github.com/intel/govmm v0.0.0-20190726135928-e0505242c067
	github.com/kata-containers/agent v0.0.0-20190325142335-48dd1c031530
	github.com/kata-containers/tests v0.0.0-20190730180451-cac5706e3d57 // indirect
	github.com/mailru/easyjson v0.0.0-20190221075403-6243d8e04c3f
	github.com/mdlayher/vsock v0.0.0-20181130155850-676f733b747c
	github.com/mitchellh/mapstructure v1.1.2
	github.com/onsi/ginkgo v1.8.0 // indirect
	github.com/opencontainers/go-digest v1.0.0-rc1
	github.com/opencontainers/runc v0.0.0-20170926091510-0351df1c5a66
	github.com/opencontainers/runtime-spec v0.0.0-20180913141938-5806c3563733
	github.com/opencontainers/specs v1.0.1 // indirect
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pkg/errors v0.8.1
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/procfs v0.0.0-20190328153300-af7bedc223fb
	github.com/safchain/ethtool v0.0.0-20180308075350-79559b488d88
	github.com/seccomp/libseccomp-golang v0.9.0
	github.com/sirupsen/logrus v0.0.0-20170822132746-89742aefa4b2
	github.com/stretchr/testify v1.2.2
	github.com/uber/jaeger-client-go v2.15.0+incompatible
	github.com/uber/jaeger-lib v1.5.0
	github.com/urfave/cli v0.0.0-20170926034118-ac249472b7de
	github.com/vishvananda/netlink v0.0.0-20171114041946-c2a3de3b38bd
	github.com/vishvananda/netns v0.0.0-20170707011535-86bef332bfc3
	golang.org/x/crypto v0.0.0-20190228161510-8dd112bcdc25
	golang.org/x/net v0.0.0-20190213061140-3a22650c66bd
	golang.org/x/sys v0.0.0-20190215142949-d0b11bdaac8a
	golang.org/x/text v0.3.0
	google.golang.org/genproto v0.0.0-20190307195333-5fe7a883aa19
	google.golang.org/grpc v1.19.0
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.2.2
)
