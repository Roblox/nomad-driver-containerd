#!/bin/bash

set -euo pipefail

export NOMAD_VERSION=0.11.2
export CONTAINERD_VERSION=1.3.4
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:/usr/local/bin
export GOPATH=/home/circleci/go
export GO_VERSION=1.14.3

# These tests are designed to be run as part of continous integration (CI) and not on local host.
# Please don't run these tests (./run_tests.sh) on your local host, as these are meant to be
# destructive and can modify (or destroy) software on your host system.
main() {
	echo "Starting setup."
	setup
	echo "Setup finished successfully."

	echo "Checking if nomad-driver-containerd is up and running, and nomad is ready to accept jobs."
	is_containerd_driver_active

	cd ~/go/src/github.com/Roblox/nomad-driver-containerd/example

	echo "Starting nomad redis job using nomad-driver-containerd."
	nomad job run redis.nomad

	echo "Starting nomad signal handler job using nomad-driver-containerd."
	nomad job run signal.nomad

	echo "Checking status of redis job."
	redis_status=$(nomad job status -short redis|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
	if [ $redis_status != "running" ];then
		echo "Error in getting redis job status."
		exit 1
	fi
	# Even though $(nomad job status) reports redis job status as "running"
	# The actual container process might not be running yet.
	# We need to wait for actual container to start running before trying exec.
	echo "Wait for redis container to get into RUNNING state, before trying exec."
	is_redis_container_active

	echo "Checking status of signal handler job."
	signal_status=$(nomad job status -short signal|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
        if [ $signal_status != "running" ];then
		echo "Error in getting signal handler job status."
		exit 1
	fi

	echo "Inspecting redis job."
	redis_status=$(nomad job inspect redis|jq -r '.Job .Status')
	if [ $redis_status != "running" ];then
		echo "Error in inspecting redis job."
		exit 1
	fi

	echo "Inspecting signal handler job."
	signal_status=$(nomad job inspect signal|jq -r '.Job .Status')
	if [ $signal_status != "running" ]; then
		echo "Error in inspecting signal handler job."
		exit 1
	fi

	echo "Exec redis job."
	exec_output=$(nomad alloc exec -job redis echo hello_exec)
	if [ $exec_output != "hello_exec" ]; then
		echo "Error in exec'ing redis job."
		exit 1
	fi

	echo "Stopping nomad redis job."
	nomad job stop redis
	redis_status=$(nomad job status -short redis|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
	if [ $redis_status != "dead(stopped)" ];then
		echo "Error in stopping redis job."
		exit 1
	fi

	echo "Stopping nomad signal handler job."
	nomad job stop signal
	signal_status=$(nomad job status -short signal|grep Status|awk '{split($0,a,"="); print a[2]}'|tr -d ' ')
        if [ $signal_status != "dead(stopped)" ];then
                echo "Error in stopping signal handler job."
                exit 1
        fi
	echo "Tests finished successfully."
}

setup() {
	sudo systemctl stop apt-daily-upgrade apt-daily >/dev/null 2>&1

	set +e
	sudo pkill --signal SIGKILL -P $(ps faux | grep 'daily' | awk '{print $2}')
	set -e

	# Remove docker daemon and containerd.
	sudo systemctl stop docker
	sudo systemctl stop containerd
	sudo apt-get purge -y docker-ce docker-ce-cli containerd.io

	sudo apt-get update
	sudo apt-get install -y apt-utils curl runc unzip make build-essential

	# Change $(pwd) to /tmp
	cd /tmp

	# Install containerd 1.3.4
	curl -L -o containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz
	sudo tar -C /usr/local -xzf containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz
	rm -f containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz

	# Drop containerd systemd unit file into /lib/systemd/system.
	cat << EOF > containerd.service
# /lib/systemd/system/nomad.service
[Unit]
Description=containerd container runtime
Documentation=https://containerd.io
After=network.target

[Service]
ExecStartPre=-/sbin/modprobe overlay
ExecStart=/usr/local/bin/containerd
KillMode=process
Delegate=yes
LimitNOFILE=1048576
# Having non-zero Limit*s causes performance problems due to accounting overhead
# in the kernel. We recommend using cgroups to do container-local accounting.
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity

[Install]
WantedBy=multi-user.target
EOF

        sudo mv containerd.service /lib/systemd/system/containerd.service
        sudo systemctl daemon-reload
        echo "Starting containerd daemon."
        sudo systemctl start containerd
	is_systemd_service_active "containerd.service"

	# Remove default golang (1.7.3) and install a custom version (1.14.3) of golang.
	# This is required for supporting go mod, and to be able to compile nomad-driver-containerd.
	sudo rm -rf /usr/local/go

	# Install golang 1.14.3
	curl -L -o go${GO_VERSION}.linux-amd64.tar.gz https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz
	sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
	sudo chmod +x /usr/local/go
	rm -f go${GO_VERSION}.linux-amd64.tar.gz

	# Install nomad 0.11.2
	curl -L -o nomad_${NOMAD_VERSION}_linux_amd64.zip https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_linux_amd64.zip
	sudo unzip -d /usr/local/bin nomad_${NOMAD_VERSION}_linux_amd64.zip
	sudo chmod +x /usr/local/bin/nomad
	rm -f nomad_${NOMAD_VERSION}_linux_amd64.zip

	echo "Building nomad-driver-containerd."
	cd ~/go/src/github.com/Roblox/nomad-driver-containerd
	make build
	echo "move containerd-driver to /tmp/nomad-driver-containerd."
	mkdir -p /tmp/nomad-driver-containerd
	mv containerd-driver /tmp/nomad-driver-containerd

	# Drop nomad server (dev) + nomad-driver-containerd systemd unit file into /lib/systemd/system.
	cat << EOF > nomad.service
# /lib/systemd/system/nomad.service
[Unit]
Description=nomad server (dev) + nomad-driver-containerd
Documentation=https://nomadproject.io
After=network.target

[Service]
ExecStart=/usr/local/bin/nomad agent -dev -config=/home/circleci/go/src/github.com/Roblox/nomad-driver-containerd/example/agent.hcl -plugin-dir=/tmp/nomad-driver-containerd
KillMode=process
Delegate=yes
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity

[Install]
WantedBy=multi-user.target
EOF
	sudo mv nomad.service /lib/systemd/system/nomad.service
	sudo systemctl daemon-reload
	echo "Starting nomad server and nomad-driver-containerd."
	sudo systemctl start nomad
	is_systemd_service_active "nomad.service"
}

is_containerd_driver_active() {
	i="0"
	while test $i -lt 5
	do
		set +e
		status=$(curl -s http://127.0.0.1:4646/v1/nodes|jq '.[0] ."Drivers" ."containerd-driver" ."Healthy"')
		rc=$?
		set -e
		if [[ $rc -eq 0 && $status = "true" ]]; then
			echo "containerd driver is up and running."
			break
		fi
		echo "containerd driver is down, sleep for 3 seconds."
		sleep 3s
		i=$[$i+1]
	done

	if [ $i -ge 5 ]; then
		echo "containerd driver didn't come up. exit 1."
		exit 1
	fi
}

is_redis_container_active() {
	set +e
	i="0"
	while test $i -lt 5
	do
		sudo CONTAINERD_NAMESPACE=nomad ctr task ls|grep -q RUNNING
		if [ $? -eq 0 ]; then
			echo "redis container is up and running"
			break
		fi
		echo "redis container is down, sleep for 3 seconds."
		sleep 3s
		i=$[$i+1]
	done
	set -e

	if [ $i -ge 5 ]; then
		echo "redis container didn't come up. exit 1."
		exit 1
	fi
}

is_systemd_service_active() {
	local service_name=$1
	i="0"
	while test $i -lt 5 && !(systemctl -q is-active "$service_name"); do
		printf "%s is down, sleep for 3 seconds.\n" $service_name
		sleep 3s
		i=$[$i+1]
	done

	if [ $i -ge 5 ]; then
		printf "%s didn't come up. exit 1.\n" $service_name
		exit 1
	fi
	printf "%s is up and running\n" $service_name
}

main "$@"
