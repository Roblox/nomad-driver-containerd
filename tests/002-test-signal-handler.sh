#!/bin/bash

test_signal_handler_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "Starting nomad signal handler job using nomad-driver-containerd."
    nomad job run signal.nomad

    echo "Checking status of signal handler job."
    signal_status=$(nomad job status -short signal|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $signal_status != "running" ];then
        echo "Error in getting signal handler job status."
        exit 1
    fi

    echo "Inspecting signal handler job."
    signal_status=$(nomad job inspect signal|jq -r '.Job .Status')
    if [ $signal_status != "running" ]; then
        echo "Error in inspecting signal handler job."
        exit 1
    fi

    echo "Stopping nomad signal handler job."
    nomad job stop signal
    signal_status=$(nomad job status -short signal|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $signal_status != "dead(stopped)" ];then
        echo "Error in stopping signal handler job."
        exit 1
    fi

    popd
}

test_signal_handler_nomad_job
