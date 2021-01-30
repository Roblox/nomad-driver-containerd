#!/bin/bash

is_container_active() {
	local job_name=$1
	local is_sleep=$2

        i="0"
        while test $i -lt 5
        do
                sudo CONTAINERD_NAMESPACE=nomad ctr task ls|grep -q RUNNING
                if [ $? -eq 0 ]; then
                        echo "INFO: ${job_name} container is up and running"
			if [ "$is_sleep" = true ]; then
                           sleep 7s
			fi
                        break
                fi
                echo "INFO: ${job_name} container is down, sleep for 4 seconds."
                sleep 4s
                i=$[$i+1]
        done

        if [ $i -ge 5 ]; then
                echo "ERROR: ${job_name} container didn't come up. exit 1."
                exit 1
        fi
}
