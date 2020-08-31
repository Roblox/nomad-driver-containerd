# nomad-driver-containerd
[![CircleCI](https://circleci-github.rcs.simulpong.com/gh/Roblox/nomad-driver-containerd/tree/master.svg?style=shield&circle-token=559609ed9ed99da393798c76f4db004f3cd66801)](https://circleci-github.rcs.simulpong.com/gh/Roblox/nomad-driver-containerd/tree/master)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://github.com/Roblox/nomad-driver-containerd/blob/master/LICENSE)
[![Release](https://img.shields.io/badge/version-0.1-blue)](https://github.com/Roblox/nomad-driver-containerd/releases/tag/v0.1)

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

- [Nomad](https://www.nomadproject.io/downloads.html) >=v0.11
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

## Supported options

**Driver Config**

| Option | Type | Required | Default | Description |
| :---: | :---: | :---: | :---: | :--- |
| **enabled** | bool | no | true | Enable/Disable task driver. |
| **containerd_runtime** | string | yes | N/A | Runtime for containerd e.g. `io.containerd.runc.v1` or `io.containerd.runc.v2`. |
| **stats_interval** | string | no | 1s | Interval for collecting `TaskStats` |

**Task Config**

| Option | Type | Required | Description |
| :---: | :---: | :---: | :--- |
| **image** | string | yes | OCI image (docker is also OCI compatible) for your container. |
| **command** | string | no | Command to override command defined in the image. |
| **args** | []string | no | Arguments to the command. |
| **privileged** | bool | no | Run container in privileged mode. Your container will have all linux capabilities when running in privileged mode. |
| **seccomp** | bool | no | Enable default seccomp profile. List of [`allowed syscalls`](https://github.com/containerd/containerd/blob/master/contrib/seccomp/seccomp_default.go#L51-L390). |
| **readonly_rootfs** | bool | no | Container root filesystem will be read-only. |
| **host_network** | bool | no | Enable host network. This is equivalent to `--net=host` in docker. |
| **cap_add** | []string | no | Add individual capabilities. |
| **cap_drop** | []string | no | Drop invidual capabilities. |
| **devices** | []string | no | A list of devices to be exposed to the container. |
| **mounts** | []block | no | A list of mounts to be mounted in the container. Volume, bind and tmpfs type mounts are supported. fstab style [`mount options`](https://github.com/containerd/containerd/blob/master/mount/mount_linux.go#L187-L211) are supported. |

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
 $ curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.1/cni-plugins-linux-amd64-v0.8.1.tgz
 $ sudo mkdir -p /opt/cni/bin
 $ sudo tar -C /opt/cni/bin -xzf cni-plugins.tgz
```

## Tests
```
$ make test
```
**NOTE**: These are destructive tests and can leave the system in a changed state.<br/>
It is highly recommended to run these tests either as part of a CI/CD system or on
a immutable infrastructure e.g VMs.

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

## Limitations

`nomad-driver-containerd` [`v0.1`](https://github.com/Roblox/nomad-driver-containerd/releases/tag/v0.1) is **not** production ready.
There are some open items which are currently being worked on.

1) **Port forwarding**: The ability to map a host port to a container port. This is currently not supported, but could be supported in future.

2) **Consul connect**: When a user launches a job in `nomad`, s/he can add a [`service stanza`](https://www.nomadproject.io/docs/job-specification/service) which will instruct `nomad` to register the service with `consul` for service discovery. This is currently not supported.

## License

Copyright 2020 Roblox Corporation

Licensed under the Apache License, Version 2.0 (the "License"). For more information read the [License](LICENSE).
