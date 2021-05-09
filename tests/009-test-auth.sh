#!/bin/bash

source $SRCDIR/utils.sh

job_name=auth

# test auth
test_auth_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad $job_name job using nomad-driver-containerd."
    nomad job run $job_name.nomad

    wait_nomad_job_status $job_name failed

    echo "INFO: Checking can not pull image without auth info."
    local alloc_id
    alloc_id=$(nomad job status auth|grep Allocations -A2|tail -n 1 |awk '{print $1}')
    nomad status "$alloc_id"|grep -q "pull access denied, repository does not exist or may require authorization"
    if [ $? -ne 0 ];then
        echo "ERROR: Can not found pull access denied in alloc log."
        exit 1
    fi

    echo "INFO: purge nomad ${job_name} job."
    nomad job stop -purge ${job_name}
    popd
}

test_auth_nomad_job
