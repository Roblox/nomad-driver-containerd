# nomad-driver-containerd
[![CI Actions Status](https://github.com/Roblox/nomad-driver-containerd/workflows/CI/badge.svg)](https://github.com/Roblox/nomad-driver-containerd/actions)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/Roblox/nomad-driver-containerd/blob/master/LICENSE)
[![Release](https://img.shields.io/badge/version-0.9.3-blue)](https://github.com/Roblox/nomad-driver-containerd/releases/tag/v0.9.3)
[![Docs](https://img.shields.io/badge/docs-website-green.svg)](https://www.nomadproject.io/docs/drivers/external/containerd)

<img src="images/nomad.png" width="232" height="100" />&nbsp;<img src="images/docker.png" width="116" height="100" />&nbsp;<img src="images/containerd.png" width="285" height="100" />

We are actively looking for contributors and maintainers for this project.
If you have experience in container internals e.g. cgroups, namespaces, or have contributed to any open source projects
around containers e.g. [`docker`](https://github.com/moby/moby), [`containerd`](https://github.com/containerd/containerd), [`nerdctl`](https://github.com/containerd/nerdctl), [`podman`](https://github.com/containers/podman) etc or build tooling which involves dealing with
container internals, and are interested in contributing to this project, I would love to talk to you!
Golang experience is preferred but not required.

Please reach out to me [`@_shishir_m`](https://twitter.com/_shishir_m) or open an issue in this repository with your contact details, if you are interested in contributing to this project.

## Overview
Nomad task driver for launching containers using containerd.

**Containerd** [`(containerd.io)`](https://containerd.io) is a lightweight container daemon for
running and managing container lifecycle.<br/>
Docker daemon also uses containerd.

```
dockerd (docker daemon) --> containerd --> containerd-shim --> runc
```

**nomad-driver-containerd** enables nomad client to launch containers directly using containerd, without docker!<br/>
Docker daemon is not required on the host system.

## nomad-driver-containerd architecture
<img src="images/nomad_driver_containerd.png" width="850" height="475" />

## Requirements

- [Nomad](https://www.nomadproject.io/downloads.html) >=v1.0
- [Go](https://golang.org/doc/install) >=v1.11
- [Containerd](https://containerd.io/downloads/) >=1.3
- [Vagrant](https://www.vagrantup.com/downloads.html) >=v2.2
- [VirtualBox](https://www.virtualbox.org/) v6.0 (or any version vagrant is compatible with)

## Building nomad-driver-containerd

Make sure your **$GOPATH** is setup correctly.
```
$ mkdir -p $GOPATH/src/github.com/Roblox
$ cd $GOPATH/src/github.com/Roblox
$ git clone git@github.com:Roblox/nomad-driver-containerd.git
$ cd nomad-driver-containerd
$ make build (This will build your containerd-driver binary)
```

If you want to compile for `arm64`, you can run:

```
make -f Makefile.arm64
```

## Screencast
[![asciicast](https://asciinema.org/a/348173.svg)](https://asciinema.org/a/348173)

## Wanna try it out!?

```
$ vagrant up
```
or `vagrant provision` if the vagrant VM is already running.

Once setup (`vagrant up` OR `vagrant provision`) is complete and the nomad server is up and running, you can check the registered task drivers (which will also show `containerd-driver`) using:
```
$ nomad node status (Note down the <node_id>)
$ nomad node status <node_id> | grep containerd-driver
```

**NOTE:** [`setup.sh`](vagrant/setup.sh) is part of the vagrant setup and should not be executed directly.

## Run Example jobs.

There are few example jobs in the [`example`](https://github.com/Roblox/nomad-driver-containerd/tree/master/example) directory.

```
$ nomad job run <job_name.nomad>
```
will launch the job.<br/>

More detailed instructions are in the [`example README.md`](https://github.com/Roblox/nomad-driver-containerd/tree/master/example)

To interact with `images` and `containers` directly, you can use [`nerdctl`](https://github.com/containerd/nerdctl) which is a docker compatible CLI for `containerd`. `nerdctl` is already installed in the vagrant VM at `/usr/local/bin`.

## Supported options

**Driver Config**

| Option | Type | Required | Default | Description |
| :---: | :---: | :---: | :---: | :--- |
| **enabled** | bool | no | true | Enable/Disable task driver. |
| **containerd_runtime** | string | yes | N/A | Runtime for containerd e.g. `io.containerd.runc.v1` or `io.containerd.runc.v2`. |
| **stats_interval** | string | no | 1s | Interval for collecting `TaskStats`. |
| **allow_privileged** | bool | no | true | If set to `false`, driver will deny running privileged jobs. |
| **auth** | block | no | N/A | Provide authentication for a private registry. See [Authentication](#authentication-private-registry) for more details. |

**Task Config**

| Option | Type | Required | Description |
| :---: | :---: | :---: | :--- |
| **image** | string | yes | OCI image (docker is also OCI compatible) for your container. |
| **image_pull_timeout** | string | no | A time duration that controls how long `containerd-driver` will wait before cancelling an in-progress pull of the OCI image as specified in `image`. Defaults to `"5m"`. |
| **command** | string | no | Command to override command defined in the image. |
| **args** | []string | no | Arguments to the command. |
| **entrypoint** | []string | no | A string list overriding the image's entrypoint. |
| **cwd** | string | no | Specify the current working directory for your container process. If the directory does not exist, one will be created for you. |
| **privileged** | bool | no | Run container in privileged mode. Your container will have all linux capabilities when running in privileged mode. |
| **pids_limit** | int64 | no | An integer value that specifies the pid limit for the container. Defaults to unlimited. |
| **pid_mode** | string | no | `host` or not set (default). Set to `host` to share the PID namespace with the host. |
| **file_limit** | int64 | no | An integer value that specifies the file descriptor ulimit for the container. Defaults to 1024 by containerd. |
| **hostname** | string | no | The hostname to assign to the container. When launching more than one of a task (using `count`) with this option set, every container the task starts will have the same hostname. |
| **host_dns** | bool | no | Default (`true`). By default, a container launched using `containerd-driver` will use host `/etc/resolv.conf`. This is similar to [`docker behavior`](https://docs.docker.com/config/containers/container-networking/#dns-services). However, if you don't want to use host DNS, you can turn off this flag by setting `host_dns=false`. |
| **seccomp** | bool | no | Enable default seccomp profile. List of [`allowed syscalls`](https://github.com/containerd/containerd/blob/master/contrib/seccomp/seccomp_default.go#L51-L395). |
| **seccomp_profile** | string | no | Path to custom seccomp profile. `seccomp` must be set to `true` in order to use `seccomp_profile`. The default `docker` seccomp profile found [`here`](https://github.com/moby/moby/blob/master/profiles/seccomp/default.json) can be used as a reference, and modified to create a custom seccomp profile. |
| **shm_size** | string | no | Size of /dev/shm e.g. "128M" if you want 128 MB of /dev/shm. |
| **sysctl** | map[string]string | no | A key-value map of sysctl configurations to set to the containers on start. |
| **readonly_rootfs** | bool | no | Container root filesystem will be read-only. |
| **host_network** | bool | no | Enable host network. This is equivalent to `--net=host` in docker. |
| **extra_hosts** | []string | no | A list of hosts, given as host:IP, to be added to /etc/hosts. |
| **cap_add** | []string | no | Add individual capabilities. |
| **cap_drop** | []string | no | Drop invidual capabilities. |
| **devices** | []string | no | A list of devices to be exposed to the container. |
| **auth** | block | no | Provide authentication for a private registry. See [Authentication](#authentication-private-registry) for more details. |
| **mounts** | []block | no | A list of mounts to be mounted in the container. Volume, bind and tmpfs type mounts are supported. fstab style [`mount options`](https://github.com/containerd/containerd/blob/master/mount/mount_linux.go#L211-L235) are supported. |

**Mount block**<br/>
       &emsp;&emsp;\{<br/>
          &emsp;&emsp;&emsp;- **type** (string) (Optional): Supported values are `volume`, `bind` or `tmpfs`. **Default:** volume.<br/>
          &emsp;&emsp;&emsp;- **target** (string) (Required): Target path in the container.<br/>
          &emsp;&emsp;&emsp;- **source** (string) (Optional): Source path on the host.<br/>
          &emsp;&emsp;&emsp;- **options** ([]string) (Optional): fstab style [`mount options`](https://github.com/containerd/containerd/blob/master/mount/mount_linux.go#L187-L211). **NOTE**: For bind mounts, atleast `rbind` and `ro` are required.<br/>
       &emsp;&emsp;\}

**Bind mount example**
```
mounts = [
           {
                type    = "bind"
                target  = "/target/t1"
                source  = "/src/s1"
                options = ["rbind", "ro"]
           }
        ]
```

In additon to the `mounts` option in `Task Config`, you can also mount your volumes into the container using nomad [`volume_mount stanza`](https://www.nomadproject.io/docs/job-specification/volume_mount)

See [`example job`](https://github.com/Roblox/nomad-driver-containerd/blob/master/example/volume_mount.nomad) for `volume_mount`.

**Custom seccomp profile example**

The default `docker` seccomp profile found [`here`](https://github.com/moby/moby/blob/master/profiles/seccomp/default.json)
can be downloaded, and modified (by removing/adding syscalls) to create a custom seccomp profile.<br/>
The custom seccomp profile can then be saved under `/opt/seccomp/seccomp.json` on the Nomad client nodes.

A nomad job can be launched using this custom seccomp profile.
```
config {
	seccomp         = true
	seccomp_profile = "/opt/seccomp/seccomp.json"
}
```

**Sysctl example**

```
config {
  sysctl = {
    "net.core.somaxconn"  = "16384"
    "net.ipv4.ip_forward" = "1"
  }
}
```

## Authentication (Private registry)

`auth` stanza allow you to set credentials for your private registry e.g. if you want to pull
an image from a private repository in docker hub.<br/>
`auth` stanza can be set either in `Driver Config` or `Task Config` or both.<br/>
If set at both places, `Task Config` auth will take precedence over `Driver Config` auth.

**NOTE**: In the below example, `user` and `pass` are just placeholder values which need to be replaced by actual `username` and `password`, when specifying the credentials. Below `auth` stanza can be used for both `Driver Config` and `Task Config`.

```
auth {
    username = "user"
    password = "pass"
}
```

## Networking

`nomad-driver-containerd` supports **host** and **bridge** networks.<br/>

**NOTE:** `host` and `bridge` are mutually exclusive options, and only one of them should be used at a time.

1. **Host** network can be enabled by setting `host_network` to `true` in task config
of the job spec (see under [`Supported options`](https://github.com/Roblox/nomad-driver-containerd#supported-options)).

2. **Bridge** network can be enabled by setting the `network` stanza in the task group section of the job spec.

```
network {
  mode = "bridge"
}
```
You need to install CNI plugins on Nomad client nodes under `/opt/cni/bin` before you can use `bridge` networks.

**Instructions for installing CNI plugins.**<br/>
```
 $ curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.6/cni-plugins-linux-amd64-v0.8.6.tgz
 $ sudo mkdir -p /opt/cni/bin
 $ sudo tar -C /opt/cni/bin -xzf cni-plugins.tgz
```
Also, ensure your Linux operating system distribution has been configured to allow container traffic through the bridge network to be routed via iptables. These tunables can be set as follows:

```
$ echo 1 > /proc/sys/net/bridge/bridge-nf-call-arptables
$ echo 1 > /proc/sys/net/bridge/bridge-nf-call-ip6tables
$ echo 1 > /proc/sys/net/bridge/bridge-nf-call-iptables
```
To preserve these settings on startup of a nomad client node, add a file including the following to `/etc/sysctl.d/` or remove the file your Linux distribution puts in that directory.

```
net.bridge.bridge-nf-call-arptables = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
```

## Port forwarding

nomad supports both **static** and **dynamic** port mapping.

1. **Static ports**

Static port mapping can be added in the `network` stanza.
```
network {
  mode = "bridge"
  port "lb" {
    static = 8889
    to     = 8889
  }
}
```
Here, `host` port `8889` is mapped to `container` port `8889`.<br/>
**NOTE**: static ports are usually not recommended, except for `system` or specialized jobs like load balancers.

2. **Dynamic ports**

Dynamic port mapping is also enabled in the `network` stanza.
```
network {
  mode = "bridge"
  port "http" {
    to = 8080
  }
}
```
Here, nomad will allocate a dynamic port on the `host` and that port will be mapped to `8080` in the container.

You can also read more about `network stanza` in the [`nomad official documentation`](https://www.nomadproject.io/docs/job-specification/network)

## Service discovery

Nomad schedules workloads of various types across a cluster of generic hosts. Because of this, placement is not known in advance and you will need to use service discovery to connect tasks to other services deployed across your cluster. Nomad integrates with Consul to provide service discovery and monitoring.

A [`service`](https://www.nomadproject.io/docs/job-specification/service) stanza can be added to your job spec, to enable service discovery.

The service stanza instructs Nomad to register a service with Consul.

## Tests

If you are running the tests locally, use the [`vagrant VM`](Vagrantfile) provided in the repository.

```
$ vagrant up
$ vagrant ssh containerd-linux
$ sudo make test
```
**NOTE**: These are destructive tests and can leave the system in a changed state.<br/>
It is highly recommended to run these tests either as part of a CI/CD system e.g. circleci or on
a immutable infrastructure e.g vagrant VMs.

You can also run an individual test by specifying the test name. e.g.

```
$ cd tests
$ sudo ./run_tests.sh 001-test-redis.sh
```

## Cleanup
```
make clean
```
This will delete your binary: `containerd-driver`

```
vagrant destroy
```
This will destroy your vagrant VM.

## Currently supported environments
Ubuntu (>= 16.04)

## License

Copyright 2020 Roblox Corporation

Licensed under the Apache License, Version 2.0 (the "License"). For more information read the [License](LICENSE).
