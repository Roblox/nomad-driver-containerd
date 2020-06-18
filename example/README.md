Example jobs
======

## Redis Server

```
$ nomad job run redis.nomad
```
will start a `redis` server using `nomad-driver-containerd`

### Exec into redis container

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
will start the signal handler container. You can send any signal
[(from a list of supported signals)](https://github.com/hashicorp/consul-template/blob/master/signals/signals_unix.go)
to this container and it will print the signal on `stdout` for you.

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
will start a stress test container. This container is based on linux `stress-ng` tool which is used for generating
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
