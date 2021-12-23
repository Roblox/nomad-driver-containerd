#!/bin/bash

source $SRCDIR/utils.sh

job_name=annotations

test_annotations_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad annotations job using nomad-driver-containerd."
    nomad job run -detach annotations.nomad

    # Even though $(nomad job status) reports job status as "running"
    # The actual container process might not be running yet.
    echo "INFO: Wait for ${job_name} container to get into RUNNING state"
    is_container_active ${job_name} true

    annotations_status=$(nomad job status -short annotations|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$annotations_status" != "running" ];then
        echo "ERROR: Error in getting annotations job status. Has status of '$annotations_status'"
        exit 1
    fi

    echo "INFO: Check annotations are found when inspecting container"
    sudo ctr -n nomad containers ls| grep annotations | cut -d ' ' -f1 | sudo xargs ctr -n nomad containers info | jq '.Spec.annotations.test' | tr -d '"' | xargs -I % test % = "annotations"
    if [[ $? -ne 0 ]]; then
        echo "ERROR: Error in getting annotations from container"
        exit 1
    fi

    echo "INFO: Stopping nomad annotations job."
    nomad job stop -detach annotations
    annotations_status=$(nomad job status -short annotations|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$annotations_status" != "dead(stopped)" ];then
        echo "ERROR: Error in stopping ${job_name} job. Has status of '$annotations_status'"
        exit 1
    fi

    echo "INFO: purge nomad annotations job."
    nomad job stop -detach -purge annotations
    popd
}

test_annotations_nomad_job
