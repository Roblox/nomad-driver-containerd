#!/bin/bash

source $SRCDIR/utils.sh

# privileged mode, devices and mounts are tested as part of this test.
test_privileged_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    setup_bind_source

    echo "INFO: Starting nomad privileged job using nomad-driver-containerd."
    nomad job run -detach privileged.nomad

    echo "INFO: Checking status of privileged job."
    job_status=$(nomad job status -short privileged|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $job_status != "running" ];then
        echo "ERROR: Error in getting privileged job status."
        exit 1
    fi

    # Even though $(nomad job status) reports privileged job status as "running"
    # The actual container process might not be running yet.
    # We need to wait for actual container to start running before trying exec.
    echo "INFO: Wait for privileged container to get into RUNNING state, before trying exec."
    is_container_active privileged true

    echo "INFO: Inspecting privileged job."
    job_status=$(nomad job inspect privileged|jq -r '.Job .Status')
    if [ $job_status != "running" ]; then
        echo "ERROR: Error in inspecting privileged job."
        exit 1
    fi

    # Check if container is running in privileged mode.
    echo "INFO: Checking if container is running in privileged mode."

    # If you are running the tests locally in the vagrant VM (Ubuntu 18.04.03)
    # the capability set (capsh --print) consists of 37 capabilities.
    # However, GHA environment is showing 39 capabilities.
    # The below check will set the expected_capabilities to 37 or 39
    # depending on the execution environment.
    expected_capabilities="37"
    if [[ "$GITHUB_ACTIONS" == "true" ]]; then
       expected_capabilities="40"
    fi

    actual_capabilities=$(nomad alloc exec -job privileged capsh --print|grep -i bounding|cut -d '=' -f 2|awk '{split($0,a,","); print a[length(a)]}')
    if [ "$expected_capabilities" != "$actual_capabilities" ]; then
       echo "ERROR: container is not running in privileged mode."
       exit 1
    fi

    # Check if bind mount exists.
    echo "INFO: Checking if bind mount exists."
    output=$(nomad alloc exec -job privileged cat /tmp/t1/bind.txt)
    if [ "$output" != "hello" ]; then
       echo "ERROR: bind mount does not exist in container rootfs."
       exit 1
    fi

    # Check if device /dev/loop0 exists.
    echo "INFO: Checking if /dev/loop0 exists in container rootfs."
    nomad alloc exec -job privileged stat /dev/loop0 >/dev/null 2>&1
    rc=$?
    if [ $rc -ne 0 ]; then
       echo "ERROR: /dev/loop0 does not exist in container rootfs."
       exit 1
    fi

    echo "INFO: Stopping nomad privileged job."
    nomad job stop -detach privileged
    job_status=$(nomad job status -short privileged|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $job_status != "dead(stopped)" ];then
        echo "ERROR: Error in stopping privileged job."
        exit 1
    fi

    echo "INFO: purge nomad privileged job."
    nomad job stop -detach -purge privileged
    popd
}

setup_bind_source() {
    mkdir -p /tmp/s1
    echo hello > /tmp/s1/bind.txt
}

test_privileged_nomad_job
