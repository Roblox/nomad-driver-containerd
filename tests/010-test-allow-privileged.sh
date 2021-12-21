#!/bin/bash

source $SRCDIR/utils.sh
job_name=privileged-not-allowed

# allow_privileged=false set in the plugin config, should deny all privileged jobs.
test_allow_privileged() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    cp agent.hcl agent.hcl.bkp

    sed -i '9 i \    allow_privileged = false' agent.hcl
    sudo systemctl restart nomad
    is_systemd_service_active "nomad.service" true

    echo "INFO: Starting nomad ${job_name} job using nomad-driver-containerd."
    nomad job run -detach privileged_not_allowed.nomad
    # Sleep for 5 seconds, to allow ${alloc_id} to get populated.
    sleep 5s

    echo "INFO: Checking status of ${job_name} job."
    alloc_id=$(nomad job status ${job_name}|grep failed|awk 'NR==1'|cut -d ' ' -f 1)
    output=$(nomad alloc status $alloc_id)
    echo -e "$output" |grep "Running privileged jobs are not allowed" &>/dev/null
    if [ $? -ne 0 ];then
       echo "ERROR: ${job_name} should have failed to run."
       return 1
    fi

    echo "INFO: purge nomad ${job_name} job."
    nomad job stop -detach -purge ${job_name}

    mv agent.hcl.bkp agent.hcl
    popd
}

cleanup() {
    if [ -f agent.hcl.bkp ]; then
       mv agent.hcl.bkp agent.hcl
    fi
    sudo systemctl restart nomad
    is_systemd_service_active "nomad.service" true
}

trap cleanup EXIT

test_allow_privileged
