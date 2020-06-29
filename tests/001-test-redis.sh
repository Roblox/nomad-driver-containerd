#!/bin/bash

test_redis_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "Starting nomad redis job using nomad-driver-containerd."
    nomad job run redis.nomad

    redis_status=$(nomad job status -short redis|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $redis_status != "running" ];then
        echo "Error in getting redis job status."
        exit 1
    fi

    # Even though $(nomad job status) reports redis job status as "running"
    # The actual container process might not be running yet.
    # We need to wait for actual container to start running before trying exec.
    echo "Wait for redis container to get into RUNNING state, before trying exec."
    is_redis_container_active

    echo "Inspecting redis job."
    redis_status=$(nomad job inspect redis|jq -r '.Job .Status')
    if [ $redis_status != "running" ];then
        echo "Error in inspecting redis job."
        exit 1
    fi

    echo "Exec redis job."
    exec_output=$(nomad alloc exec -job redis echo hello_exec)
    if [ $exec_output != "hello_exec" ]; then
        echo "Error in exec'ing redis job."
        exit 1
    fi

    echo "Stopping nomad redis job."
    nomad job stop redis
    redis_status=$(nomad job status -short redis|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $redis_status != "dead(stopped)" ];then
        echo "Error in stopping redis job."
        exit 1
    fi

    popd
}

is_redis_container_active() {
        set +e
        i="0"
        while test $i -lt 5
        do
                sudo CONTAINERD_NAMESPACE=nomad ctr task ls|grep -q RUNNING
                if [ $? -eq 0 ]; then
                        echo "redis container is up and running"
                        break
                fi
                echo "redis container is down, sleep for 3 seconds."
                sleep 3s
                i=$[$i+1]
        done
        set -e

        if [ $i -ge 5 ]; then
                echo "redis container didn't come up. exit 1."
                exit 1
        fi
}

test_redis_nomad_job