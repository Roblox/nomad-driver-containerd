Example jobs
======

## Redis Server

```
$ nomad job run redis.nomad
```
will start a `redis` server using `nomad-driver-containerd`

**Exec into redis container**

```
$ nomad job status redis
```
Copy the allocation ID from the output of `nomad job status` command.

```
$ nomad alloc exec -i -t <allocation_id> /bin/sh
```

## Signal Handler

```
$ nomad job run signal.nomad
```
will start the signal handler container.<br/>
You can send any signal [(from a list of supported signals)](https://github.com/hashicorp/consul-template/blob/master/signals/signals_unix.go) to this container and it will print the signal on `stdout` for you.

```
$ nomad job status signal
```
Copy the allocation ID from the output of `nomad job status` command.

```
$ nomad alloc signal -s <signal> <allocation_id>
```

## Stress

```
$ nomad job run stress.nomad
```
will start a stress test container.<br/>
This container is based on linux `stress-ng` tool which is used for generating
heavy load on CPU and memory to do stress testing.

This container executes the following command as an entrypoint to the container:
```
stress-ng --cpu 4 --io 4 --vm 4 --vm-bytes 256M --fork 4 --timeout 180s
```
The above command will run stress tests for 3 minutes (180 secs).

```
$ nomad job status stress
```
Copy the allocation ID from the output of `nomad job status` command.

While the container is running, you can check the stats using:
```
$ nomad alloc status -stats <allocation_id>
```

## Capabilities

```
$ nomad job run capabilities.nomad
```
will start an `ubuntu:16.04` container using `nomad-driver-containerd`.<br/>
This container sleeps for 10 mins (600 seconds), runs in `readonly` mode and
add (and drop) the following capabilities.

**New capabilities added:**
```
CAP_SYS_ADMIN
CAP_IPC_OWNER
CAP_IPC_LOCK
```
**Existing capabilities dropped:**
```
CAP_CHOWN
CAP_SYS_CHROOT
CAP_DAC_OVERRIDE
```
**Exec into capabilities container to check capabilities**

```
$ nomad job status capabilities
```
Copy the allocation ID from the output of `nomad job status` command.

```
$ nomad alloc exec -i -t <allocation_id> /bin/bash
```
Print capabilities (Inside the container)
```
$ capsh --print
```
Check readonly mode (Inside the container)
```
$ touch /tmp/file.txt
```
`touch` should throw the following error message:
```
touch: cannot touch '/tmp/file.txt': Read-only file system
```

## Privileged

```
$ nomad job run privileged.nomad
```
will start an `ubuntu:16.04` container using `nomad-driver-containerd`.<br/>
This container does the following:<br/>
<ol>
<li>Sleeps for 10 mins (600 seconds).</li>
<li>Runs in privileged mode i.e the bounding set contains all linux capabilities.</li>
<li>Add /dev/loop0 and /dev/loop1 loopback devices into the container.</li>
<li>Bind mounts /tmp/s1 (host) to /tmp/t1 (container).</li>
</ol>

**Exec into privileged container to check capabilities, devices and mounts.**

```
$ nomad job status privileged
```
Copy the allocation ID from the output of `nomad job status` command.

```
$ nomad alloc exec -i -t <allocation_id> /bin/bash
```
Print capabilities (Inside the container)
```
$ capsh --print
```
This should print all 37 capabilities as part of the bounding set.<br/>

Check for devices (Inside the container)
```
ls /dev -lt
```
This should list both `/dev/loop0` and `/dev/loop1` under devices.<br/>

Check bind mount (Inside the container)
```
mountpoint /tmp/t1
```
