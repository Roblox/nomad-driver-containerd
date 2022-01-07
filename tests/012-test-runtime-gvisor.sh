#!/bin/bash

source $SRCDIR/utils.sh

job_name=gvisor-sleep

test_gvisor_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad ${job_name} job using nomad-driver-containerd."
    nomad job run -detach ${job_name}.nomad

    # Even though $(nomad job status) reports job status as "running"
    # The actual container process might not be running yet.
    echo "INFO: Wait for ${job_name} container to get into RUNNING state"
    is_container_active ${job_name} true

    gvisor_status=$(nomad job status -short ${job_name}|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$gvisor_status" != "running" ];then
        echo "ERROR: Error in getting ${job_name} job status. Has status of '$gvisor_status'"
        exit 1
    fi

    echo "INFO: Check dmesg for gvisor log line"
    nomad alloc exec -job ${job_name} dmesg | grep -i gvisor
    if [ $? != 0 ]; then
        echo "ERROR: Error in finding gvisor in dmesg logs."
        exit 1
    fi

    echo "INFO: Stopping nomad ${job_name} job."
    nomad job stop -detach ${job_name}
    gvisor_status=$(nomad job status -short ${job_name}|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$gvisor_status" != "dead(stopped)" ];then
        echo "ERROR: Error in getting ${job_name} job status. Has status of '$gvisor_status'"
        exit 1
    fi

    echo "INFO: purge nomad ${job_name} job."
    nomad job stop -detach -purge ${job_name}
    popd
}

test_gvisor_nomad_job
