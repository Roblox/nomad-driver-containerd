// TODO: update the module path below to match your own repository
module github.com/Roblox/nomad-driver-containerd

go 1.12

require (
	github.com/LK4D4/joincontext v0.0.0-20171026170139-1724345da6d5 // indirect
	github.com/NVIDIA/gpu-monitoring-tools v0.0.0-20191126014920-0d8df858cca4 // indirect
	github.com/Sirupsen/logrus v0.0.0-00010101000000-000000000000 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/appc/spec v0.8.11 // indirect
	github.com/checkpoint-restore/go-criu v0.0.0-20191125063657-fcdcd07065c5 // indirect
	github.com/containerd/containerd v1.3.0
	github.com/containerd/go-cni v0.0.0-20191121212822-60d125212faf // indirect
	github.com/containernetworking/plugins v0.8.3 // indirect
	github.com/coreos/go-iptables v0.4.3 // indirect
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/cyphar/filepath-securejoin v0.2.2 // indirect
	github.com/docker/cli v0.0.0-20191202230238-13fb276442f5 // indirect
	github.com/docker/docker v1.13.1 // indirect
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-events v0.0.0-20190806004212-e31b211e4f1c // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/fsouza/go-dockerclient v1.6.0 // indirect
	github.com/hashicorp/consul v1.6.2 // indirect
	github.com/hashicorp/consul-template v0.23.0
	github.com/hashicorp/go-envparse v0.0.0-20190703193109-150b3a2a4611 // indirect
	github.com/hashicorp/go-getter v1.4.0 // indirect
	github.com/hashicorp/go-hclog v0.10.0
	github.com/hashicorp/go-plugin v1.0.1
	github.com/hashicorp/hcl2 v0.0.0-20191002203319-fb75b3253c80 // indirect
	github.com/hashicorp/nomad v0.10.1
	github.com/hashicorp/nomad/api v0.0.0-20191203164002-b31573ae7206 // indirect
	github.com/mitchellh/go-ps v0.0.0-20190716172923-621e5597135b // indirect
	github.com/moby/moby v1.13.1
	github.com/mrunalp/fileutils v0.0.0-20171103030105-7d4729fb3618 // indirect
	github.com/opencontainers/runc v1.0.0-rc8.0.20190611121236-6cc515888830 // indirect
	github.com/opencontainers/selinux v1.3.1 // indirect
	github.com/seccomp/libseccomp-golang v0.9.1 // indirect
	github.com/shirou/gopsutil v2.19.11+incompatible // indirect
	github.com/syndtr/gocapability v0.0.0-20180916011248-d98352740cb2 // indirect
	github.com/ugorji/go v1.1.7 // indirect
	github.com/vbatts/tar-split v0.11.1 // indirect
	github.com/zclconf/go-cty v1.1.1 // indirect
	go4.org v0.0.0-20191010144846-132d2879e1e9 // indirect
)

// use lower-case sirupsen
replace github.com/Sirupsen/logrus => github.com/sirupsen/logrus v1.4.2

// don't use shirou/gopsutil, use the hashicorp fork
replace github.com/shirou/gopsutil => github.com/hashicorp/gopsutil v2.17.13-0.20190117153606-62d5761ddb7d+incompatible

// don't use ugorji/go, use the hashicorp fork
replace github.com/ugorji/go => github.com/hashicorp/go-msgpack v0.0.0-20190927123313-23165f7bc3c2

// Workaround for upstream issue in containerd go mod.
// https://github.com/containerd/containerd/issues/3031
replace github.com/docker/distribution v2.7.1+incompatible => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible

// fix the version of hashicorp/go-msgpack to 96ddbed8d05b
replace github.com/hashicorp/go-msgpack => github.com/hashicorp/go-msgpack v0.0.0-20191101193846-96ddbed8d05b
