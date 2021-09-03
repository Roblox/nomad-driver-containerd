#!/bin/bash

source $SRCDIR/utils.sh

test_redis_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad redis job using nomad-driver-containerd."
    nomad job run -detach redis.nomad

    redis_status=$(nomad job status -short redis|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $redis_status != "running" ];then
        echo "ERROR: Error in getting redis job status."
        exit 1
    fi

    # Even though $(nomad job status) reports redis job status as "running"
    # The actual container process might not be running yet.
    # We need to wait for actual container to start running before trying exec.
    echo "INFO: Wait for redis container to get into RUNNING state, before trying exec."
    is_container_active redis false

    echo "INFO: Inspecting redis job."
    redis_status=$(nomad job inspect redis|jq -r '.Job .Status')
    if [ $redis_status != "running" ];then
        echo "ERROR: Error in inspecting redis job."
        exit 1
    fi

    echo "INFO: Exec redis job and check current working directory (cwd)."
    exec_output=$(nomad alloc exec -job redis pwd)
    if [ $exec_output != "/home/redis" ]; then
        echo "ERROR: Error in exec'ing redis job and checking current working directory (cwd)."
        exit 1
    fi

    echo "INFO: Exec redis job and check hostname is foobar."
    hostname=$(nomad alloc exec -job redis hostname)
    if [ $hostname != "foobar" ]; then
        echo "ERROR: hostname is not foobar."
        exit 1
    fi

    echo "INFO: Check if default seccomp is enabled."
    output=$(nomad alloc exec -job redis cat /proc/1/status | grep Seccomp)
    seccomp_code=$(echo $output|cut -d' ' -f2)
    if [ $seccomp_code != "2" ]; then
       echo "ERROR: default seccomp is not enabled."
       exit 1
    fi

    echo "INFO: Check if memory and memory_max are set correctly in the cgroup filesystem."
    task_name=$(sudo CONTAINERD_NAMESPACE=nomad ctr containers ls|awk 'NR!=1'|cut -d' ' -f1)
    memory_soft_limit=$(sudo cat /sys/fs/cgroup/memory/nomad/$task_name/memory.soft_limit_in_bytes)
    if [ $memory_soft_limit != "$(( 256 * 1024 * 1024 ))" ]; then
       echo "ERROR: memory should be 256 MB. Found ${memory_soft_limit}."
       exit 1
    fi
    memory_hard_limit=$(sudo cat /sys/fs/cgroup/memory/nomad/$task_name/memory.limit_in_bytes)
    if [ $memory_hard_limit != "$(( 512 * 1024 * 1024 ))" ]; then
       echo "ERROR: memory_max should be 512 MB. Found ${memory_hard_limit}."
       exit 1
    fi

    echo "INFO: Stopping nomad redis job."
    nomad job stop -detach redis
    redis_status=$(nomad job status -short redis|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $redis_status != "dead(stopped)" ];then
        echo "ERROR: Error in stopping redis job."
        exit 1
    fi

    echo "INFO: purge nomad redis job."
    nomad job stop -detach -purge redis
    popd
}

test_redis_nomad_job
