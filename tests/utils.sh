#!/bin/bash

wait_nomad_job_status() {
	local job_name=$1
	local expected_status="$2"

	local status
	local i=0
	while [ $i -lt 5 ]; do
		status=$(nomad job status $job_name|grep Allocations -A2|tail -n 1 |awk '{print $6}')
		if [ "$status" == "$expected_status" ]; then
			return
		fi
		sleep 4
		i=$((i + 1))
	done

	echo "ERROR: ${job_name} didn't enter $expected_status status. exit 1."
	exit 1
}

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

is_systemd_service_active() {
	local service_name=$1
	local is_sleep=$2

	i="0"
	while test $i -lt 5 && !(systemctl -q is-active "$service_name"); do
		printf "INFO: %s is down, sleep for 4 seconds.\n" $service_name
		sleep 4s
		i=$[$i+1]
		done

	if [ $i -ge 5 ]; then
	printf "ERROR: %s didn't come up. exit 1.\n" $service_name
		exit 1
	fi

	if [ "$is_sleep" = true ]; then
		sleep 7s
	fi
	printf "INFO: %s is up and running\n" $service_name
}
