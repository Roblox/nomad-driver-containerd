#!/bin/bash

source $SRCDIR/utils.sh

job_name=mosquitto
filename="/mosquitto/config/mosquitto.conf"

# test template stanza
test_template_stanza_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad $job_name job using nomad-driver-containerd."
    nomad job run -detach $job_name.nomad

    # Even though $(nomad job status) reports job status as "running"
    # The actual container process might not be running yet.
    # We need to wait for actual container to start running before trying exec.
    echo "INFO: Wait for ${job_name} container to get into RUNNING state, before trying exec."
    is_container_active ${job_name} true

    echo "INFO: Checking status of $job_name job."
    job_status=$(nomad job status -short $job_name|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ "$job_status" != "running" ];then
        echo "ERROR: Error in getting ${job_name} job status."
        exit 1
    fi

    # Check if bind mount exists.
    echo "INFO: Checking if bind mount was successful and $filename exists."
    nomad alloc exec -job $job_name cat $filename >/dev/null 2>&1
    rc=$?
    if [ $rc -ne 0 ]; then
       echo "ERROR: bind mount was unsuccessful. $filename does not exist."
       exit 1
    fi

    echo "INFO: Stopping nomad ${job_name} job."
    nomad job stop -detach ${job_name}
    job_status=$(nomad job status -short ${job_name}|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $job_status != "dead(stopped)" ];then
        echo "ERROR: Error in stopping ${job_name} job."
        exit 1
    fi

    echo "INFO: purge nomad ${job_name} job."
    nomad job stop -detach -purge ${job_name}
    popd
}

test_template_stanza_nomad_job
