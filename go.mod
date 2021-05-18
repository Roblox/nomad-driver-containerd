// TODO: update the module path below to match your own repository
module github.com/Roblox/nomad-driver-containerd

go 1.12

require (
	github.com/NVIDIA/gpu-monitoring-tools v0.0.0-20191126014920-0d8df858cca4 // indirect
	github.com/containerd/cgroups v0.0.0-20200609174450-80c669f4bad0
	github.com/containerd/containerd v1.4.3
	github.com/containerd/go-cni v0.0.0-20191121212822-60d125212faf // indirect
	github.com/containerd/typeurl v0.0.0-20180627222232-a93fcdb778cd
	github.com/containernetworking/plugins v0.8.3 // indirect
	github.com/coreos/go-iptables v0.4.3 // indirect
	github.com/docker/docker v17.12.0-ce-rc1.0.20200330121334-7f8b4b621b5d+incompatible
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/hashicorp/consul-template v0.25.1
	github.com/hashicorp/go-envparse v0.0.0-20190703193109-150b3a2a4611 // indirect
	github.com/hashicorp/go-hclog v0.14.1
	github.com/hashicorp/go-uuid v1.0.1
	github.com/hashicorp/nomad v1.1.0
	github.com/opencontainers/runtime-spec v1.0.3-0.20200929063507-e6143ca7d51d
	github.com/spf13/cobra v1.1.1
)

// use lower-case sirupsen
replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.1

// don't use shirou/gopsutil, use the hashicorp fork
replace github.com/shirou/gopsutil => github.com/hashicorp/gopsutil v2.17.13-0.20190117153606-62d5761ddb7d+incompatible

// don't use ugorji/go, use the hashicorp fork
replace github.com/ugorji/go => github.com/hashicorp/go-msgpack v0.0.0-20190927123313-23165f7bc3c2

// Workaround for upstream issue in containerd go mod.
// https://github.com/containerd/containerd/issues/3031
replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible

// fix the version of hashicorp/go-msgpack to 96ddbed8d05b
replace github.com/hashicorp/go-msgpack => github.com/hashicorp/go-msgpack v0.0.0-20191101193846-96ddbed8d05b

// Workaround Nomad using an old version
replace google.golang.org/api => google.golang.org/api v0.46.0
