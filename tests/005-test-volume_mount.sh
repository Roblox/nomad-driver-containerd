#!/bin/bash

source $SRCDIR/utils.sh

job_name=volume_mount
host_volume_path=/tmp/host_volume/s1

# test volume_mount
test_volume_mount_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    setup_bind_source

    echo "INFO: Starting nomad $job_name job using nomad-driver-containerd."
    nomad job run $job_name.nomad

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
    echo "INFO: Checking if bind mount exists."
    for mountpoint in t1 read_only_target ; do
        output=$(nomad alloc exec -job ${job_name} cat /tmp/${mountpoint}/bind.txt)
        if [ "$output" != "hello" ]; then
           echo "ERROR: bind mount /tmp/${mountpoint} does not exist in container rootfs."
           exit 1
        fi
    done

    # Check read only mount can not write.
    echo "INFO: Checking read only mount is not writable."
    nomad alloc exec -job ${job_name} touch /tmp/read_only_target/writable_test.txt &>/dev/null
    if [ -e ${host_volume_path}/writable_test.txt ];then
        echo "ERROR: Read only bind mount in /tmp/read_only_target should not be writable."
        exit 1
    fi

    # Check writable mount can write.
    echo "INFO: Checking non read_only mount is writable."
    nomad alloc exec -job ${job_name} touch /tmp/t1/writable_test.txt
    if [ ! -e ${host_volume_path}/writable_test.txt ];then
        echo "ERROR: bind mount in /tmp/t1 should be writable."
        exit 1
    fi

    echo "INFO: Stopping nomad ${job_name} job."
    nomad job stop ${job_name}
    job_status=$(nomad job status -short ${job_name}|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $job_status != "dead(stopped)" ];then
        echo "ERROR: Error in stopping ${job_name} job."
        exit 1
    fi

    echo "INFO: purge nomad ${job_name} job."
    nomad job stop -purge ${job_name}
    popd
}

setup_bind_source() {
    rm -f ${host_volume_path}/bind.txt
    rm -f ${host_volume_path}/writable_test.txt

    echo hello > ${host_volume_path}/bind.txt
}

test_volume_mount_nomad_job
