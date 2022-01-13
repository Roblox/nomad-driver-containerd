#!/bin/bash

source $SRCDIR/utils.sh

job_name=labels

test_labels_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad labels job using nomad-driver-containerd."
    nomad job run -detach labels.nomad

    # Even though $(nomad job status) reports job status as "running"
    # The actual container process might not be running yet.
    echo "INFO: Wait for ${job_name} container to get into RUNNING state"
    is_container_active ${job_name} true

    labels_status=$(nomad job status -short labels|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$labels_status" != "running" ];then
        echo "ERROR: Error in getting labels job status. Has status of '$labels_status'"
        exit 1
    fi

    echo "INFO: Check labels are found when inspecting container"
    sudo ctr -n nomad containers ls| grep labels | cut -d ' ' -f1 | sudo xargs ctr -n nomad containers info | jq '.Labels.test' | tr -d '"' | xargs -I % test % = "labels"
    if [[ $? -ne 0 ]]; then
        echo "ERROR: Error in getting labels from container"
        exit 1
    fi

    echo "INFO: Stopping nomad labels job."
    nomad job stop -detach labels
    labels_status=$(nomad job status -short labels|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$labels_status" != "dead(stopped)" ];then
        echo "ERROR: Error in stopping ${job_name} job. Has status of '$labels_status'"
        exit 1
    fi

    echo "INFO: purge nomad labels job."
    nomad job stop -detach -purge labels
    popd
}

test_labels_nomad_job
