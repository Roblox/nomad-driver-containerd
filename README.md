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
