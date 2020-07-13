# nomad-driver-containerd
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

## Building nomad-driver-containerd

Make sure your **$GOPATH** is setup correctly.
```
$ mkdir -p $GOPATH/src/github.com/Roblox
$ cd $GOPATH/src/github.com/Roblox
$ git clone git@github.com:Roblox/nomad-driver-containerd.git
$ cd nomad-driver-containerd
$ make build (This will build your containerd-driver binary)
```
## Wanna try it out!?

```
./setup.sh
```
The setup script will setup `containerd 1.3.4` and `nomad server+nomad-driver-containerd` (nomad server/client should already be installed on your system, and `setup.sh` only builds the driver) on your system, so you can try out [`example`](https://github.com/Roblox/nomad-driver-containerd/tree/master/example) jobs.

**NOTE** `setup.sh` overrides your existing `containerd` to `containerd-1.3.4`. This is needed for `io.containerd.runc.v2` runtime.<br/>
Your original containerd systemd unit file will be backed up at `/lib/systemd/system/containerd.service.bkp` in case you wanna revert later.

Once `setup.sh` is complete and the nomad server is up and running, you can check the registered task drivers (which will also show `containerd-driver`) using:
```
$ nomad node status (Note down the <node_id>)
$ nomad node status <node_id> | grep containerd-driver
```

## Run Example jobs.

There are few example jobs in the [`example`](https://github.com/Roblox/nomad-driver-containerd/tree/master/example) directory.

```
$ nomad job run <job_name.nomad>
```
will launch the job.<br/>

**NOTE:** You need to run `setup.sh` before trying out the example jobs.<br/>
More detailed instructions are in the [`example README.md`](https://github.com/Roblox/nomad-driver-containerd/tree/master/example)

## Supported options

| Option | Type | Required | Description |
| :---: | :---: | :---: | :--- |
| **image** | string | yes | OCI image (docker is also OCI compatible) for your container. |
| **command** | string | no | Command to override command defined in the image. |
| **args** | []string | no | Arguments to the command. |
| **privileged** | bool | no | Run container in privileged mode. Your container will have all linux capabilities when running in privileged mode. |
| **readonly_rootfs** | bool | no | Container root filesystem will be read-only. |
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

## License

Copyright 2020 Roblox Corporation

Licensed under the Apache License, Version 2.0 (the "License"). For more information read the [License](LICENSE).
