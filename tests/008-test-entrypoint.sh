#!/bin/bash

source $SRCDIR/utils.sh

job_name=entrypoint

test_entrypoint_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad $job_name job using nomad-driver-containerd."
    nomad job run $job_name.nomad

    # Even though $(nomad job status) reports job status as "running"
    # The actual container process might not be running yet.
    # We need to wait for actual container to start running before executing $(nomad alloc logs).
    echo "INFO: Wait for ${job_name} container to get into RUNNING state."
    is_container_active ${job_name} true

    echo "INFO: Checking status of $job_name job."
    job_status=$(nomad job status -short $job_name|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$job_status" != "running" ];then
        echo "ERROR: Error in getting ${job_name} job status."
        return 1
    fi

    output=$(nomad logs -job ${job_name})
    for result in "container1" "container2" ; do
        echo -e "$output" |grep "$result" &>/dev/null
        if [ $? -ne 0 ];then
           echo "ERROR: $result not found in the output."
           return 1
        fi
    done

    echo "INFO: purge nomad ${job_name} job."
    nomad job stop -purge ${job_name}
    popd
}

test_entrypoint_nomad_job
