#!/bin/bash

source $SRCDIR/utils.sh

test_signal_handler_nomad_job() {
    pushd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

    echo "INFO: Starting nomad signal handler job using nomad-driver-containerd."
    nomad job run -detach signal.nomad

    echo "INFO: Checking status of signal handler job."
    signal_status=$(nomad job status -short signal|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $signal_status != "running" ];then
        echo "ERROR: Error in getting signal handler job status."
        exit 1
    fi

    echo "INFO: Inspecting signal handler job."
    signal_status=$(nomad job inspect signal|jq -r '.Job .Status')
    if [ $signal_status != "running" ]; then
        echo "ERROR: Error in inspecting signal handler job."
        exit 1
    fi

    # Even though $(nomad job status) reports signal job status as "running"
    # The actual container process might not be running yet.
    # We need to wait for actual container to start running before trying to send invalid signal.
    echo "INFO: Wait for signal container to get into RUNNING state, before trying to send invalid signal."
    is_container_active signal false

    echo "INFO: Test invalid signal."
    alloc_id=$(nomad job status signal|awk 'END{print}'|cut -d ' ' -f 1)
    local outfile=$(mktemp /tmp/signal.XXXXXX)
    nomad alloc signal -s INVALID $alloc_id >> $outfile 2>&1
    if ! grep -q "Invalid signal" $outfile; then
        echo "ERROR: Invalid signal didn't error out."
        cleanup "$outfile"
        exit 1
    fi
    cleanup "$outfile"

    echo "INFO: Stopping nomad signal handler job."
    nomad job stop -detach signal
    signal_status=$(nomad job status -short signal|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
    if [ $signal_status != "dead(stopped)" ];then
        echo "ERROR: Error in stopping signal handler job."
        exit 1
    fi

    echo "INFO: purge nomad signal handler job."
    nomad job stop -detach -purge signal
    popd
}

cleanup() {
  local tmpfile=$1
  rm $tmpfile > /dev/null 2>&1
}

test_signal_handler_nomad_job
