#!/bin/bash

set -euo pipefail

export NOMAD_VERSION=0.11.2
export PATH=$PATH:/usr/local/go/bin
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
	while true
	do
		set +e
		status=$(curl -s http://127.0.0.1:4646/v1/nodes|jq '.[0] ."Drivers" ."containerd-driver" ."Healthy"')
		rc=$?
		set -e
		if [[ $rc -eq 0 && $status = "true" ]]; then
			echo "nomad is up and running"
			break
		fi
		echo "nomad is down, wait for 5 seconds."
		sleep 5s
	done
	cd ~/go/src/github.com/Roblox/nomad-driver-containerd/example
	echo "Starting nomad redis job using nomad-driver-containerd."
	nomad job run example.nomad
	echo "Starting nomad signal handler job using nomad-driver-containerd."
	nomad job run signal.nomad
	echo "Inspecting redis job."
	nomad job inspect example
	echo "Inspecting signal handler job."
	nomad job inspect signal
	echo "Stopping nomad redis job."
	nomad job stop example
	echo "Stopping nomad signal handler job."
	nomad job stop signal
	echo "Sleep for 5 seconds, to allow nomad jobs to shutdown gracefully."
	sleep 5s
	echo "Tests finished successfully."
}

setup() {
	sudo systemctl stop apt-daily-upgrade apt-daily >/dev/null 2>&1

	set +e
	sudo pkill --signal SIGKILL -P $(ps faux | grep 'daily' | awk '{print $2}')
	set -e

	sudo apt-get install -y apt-utils curl unzip make build-essential

	# Stop docker daemon. We only want containerd daemon running.
	sudo systemctl stop docker

	# Remove default golang (1.7.3) and install a custom version (1.14.3) of golang.
	# This is required for supporting go mod, and to be able to compile nomad-driver-containerd.
	sudo rm -rf /usr/local/go

	# Change $(pwd) to /tmp
	cd /tmp

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
	while true
	do
		if (systemctl -q is-active "nomad.service"); then
			echo "systemd nomad.service is up and running"
			break
		fi
	done
}

main "$@"
