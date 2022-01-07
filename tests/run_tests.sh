#!/bin/bash

set -eo pipefail

export NOMAD_VERSION=1.1.4
export CONTAINERD_VERSION=1.5.5
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:/usr/local/bin
if [ -e /home/runner ]; then
       export GOPATH=/home/runner/go
else
       export GOPATH=$HOME/go
fi
export GO_VERSION=1.17
export SRCDIR=`dirname $0`
source $SRCDIR/utils.sh

# Keeps track of overall pass/failure status of tests. Even if single test
# fails, PASS_STATUS will be set to 1 and returned to caller when all
# tests have run.
PASS_STATUS=0

# These tests are designed to be run as part of continous integration (CI) and not on local host.
# Please don't run these tests (./run_tests.sh) on your local host, as these are meant to be
# destructive and can modify (or destroy) software on your host system.
# If you are running the tests locally, run it in the vagrant VM provided with the repository.
main() {
	warn_on_local_host
	setup
	echo "INFO: Checking if nomad-driver-containerd is up and running, and nomad is ready to accept jobs."
	is_containerd_driver_active
	is_nomad_ready

	run_tests $@
	exit $PASS_STATUS
}

run_test () {
  testfile=$1

  echo "INFO: Running test `basename $testfile`"
  bash -c $testfile

  if [ $? -eq 0 ];then
    echo "PASS: $(basename $testfile)"
  else
    echo "FAIL: $(basename $testfile)"
    PASS_STATUS=1
  fi
}

run_tests() {
  if [ $# -gt 0 ]; then
    local files=$@
  else
    local files="$SRCDIR/[0-9][0-9][0-9]-test-*"
  fi
  for t in $files;do
    run_test ./$t
  done
}

warn_on_local_host() {
  if [[ -z "$GITHUB_ACTIONS" || "$GITHUB_ACTIONS" != "true" ]]; then
     echo "WARNING: Local host detected."
     echo "WARNING: These tests are designed to be run as part of continous integration (CI) and not recommended to be run on local host."
     echo "WARNING: These tests are destructive and can modify (or destroy) software on your host system."
     echo "WARNING: If running the tests locally, run it in the vagrant VM provided with the repository."
     read -p "Do you still want to run the tests (Y/N)? " -n 1 -r
     echo
     if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Right choice! Aborting..."
        exit 0
     fi
  fi
}

setup() {
	if [[ -z "$GITHUB_ACTIONS" || "$GITHUB_ACTIONS" != "true" ]]; then
           echo "INFO: Running tests on local host (vagrant VM). Setup is not required."
           return 0
        fi

	# Change $(pwd) to /tmp
	pushd /tmp

	sudo mkdir -p /etc/containerd/

	# Install containerd 1.5.5
	curl -L -o containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz
	sudo tar -C /usr/local -xzf containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz
	rm -f containerd-${CONTAINERD_VERSION}-linux-amd64.tar.gz

	# Drop containerd systemd unit file into /lib/systemd/system.
	cat << EOF > containerd.service
# /lib/systemd/system/containerd.service
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
        echo "INFO: Unmask containerd.service"
        sudo systemctl unmask containerd
        echo "INFO: Starting containerd daemon."
        sudo systemctl start containerd
	is_systemd_service_active "containerd.service" false

	# install gvisor and restart containerd service
  ARCH=$(uname -m)
  URL=https://storage.googleapis.com/gvisor/releases/release/latest/${ARCH}
  wget -q ${URL}/runsc ${URL}/runsc.sha512 \
  ${URL}/containerd-shim-runsc-v1 ${URL}/containerd-shim-runsc-v1.sha512
  sha512sum -c runsc.sha512 \
  -c containerd-shim-runsc-v1.sha512

  rm -f *.sha512

  sudo chmod a+rx runsc containerd-shim-runsc-v1
  sudo mv runsc containerd-shim-runsc-v1 /usr/local/bin

  sudo tee <<EOF /etc/containerd/config.toml >/dev/null
version = 2
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
  runtime_type = "io.containerd.runc.v2"
[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runsc]
  runtime_type = "io.containerd.runsc.v1"
EOF

  sudo systemctl restart containerd

	# Remove default golang (1.7.3) and install a custom version (1.17) of golang.
	# This is required for supporting go mod, and to be able to compile nomad-driver-containerd.
	sudo rm -rf /usr/local/go

	# Install golang 1.17
	curl -L -o go${GO_VERSION}.linux-amd64.tar.gz https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz
	sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
	sudo chmod +x /usr/local/go
	rm -f go${GO_VERSION}.linux-amd64.tar.gz

	# Install nomad 1.1.4
	curl -L -o nomad_${NOMAD_VERSION}_linux_amd64.zip https://releases.hashicorp.com/nomad/${NOMAD_VERSION}/nomad_${NOMAD_VERSION}_linux_amd64.zip
	sudo unzip -d /usr/local/bin nomad_${NOMAD_VERSION}_linux_amd64.zip
	sudo chmod +x /usr/local/bin/nomad
	rm -f nomad_${NOMAD_VERSION}_linux_amd64.zip

	echo "INFO: Building nomad-driver-containerd."
	cd ~/go/src/github.com/Roblox/nomad-driver-containerd
	make build
	echo "move containerd-driver to /tmp/nomad-driver-containerd."
	mkdir -p /tmp/nomad-driver-containerd
	mv containerd-driver /tmp/nomad-driver-containerd

	# Drop nomad server + nomad-driver-containerd systemd unit file into /lib/systemd/system.
	cat << EOF > nomad.service
# /lib/systemd/system/nomad.service
[Unit]
Description=nomad server + nomad-driver-containerd
Documentation=https://nomadproject.io
After=network.target

[Service]
ExecStart=/usr/local/bin/nomad agent -config=$GOPATH/src/github.com/Roblox/nomad-driver-containerd/example/agent.hcl -plugin-dir=/tmp/nomad-driver-containerd
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

	prepare_nomad_host_volume

	echo "INFO: Starting nomad server and nomad-driver-containerd."
	sudo systemctl start nomad
	is_systemd_service_active "nomad.service" false
	popd
}

prepare_nomad_host_volume() {
	echo "INFO: Prepare nomad host volume."
	mkdir -p /tmp/host_volume/s1
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
			echo "INFO: containerd driver is up and running."
			break
		fi
		echo "INFO: containerd driver is down, sleep for 4 seconds."
		sleep 4s
		i=$[$i+1]
	done

	if [ $i -ge 5 ]; then
		echo "ERROR: containerd driver didn't come up. exit 1."
		exit 1
	fi
}

is_nomad_ready() {
        i="0"
        while test $i -lt 5
        do
                set +e
                status=$(curl -s http://127.0.0.1:4646/v1/nodes|jq '.[0] ."Status"')
                rc=$?
                set -e
                if [[ $rc -eq 0 && $status = \"ready\" ]]; then
                        echo "INFO: nomad is ready to accept jobs."
                        break
                fi
                echo "INFO: nomad is initializing, sleep for 4 seconds."
                sleep 4s
                i=$[$i+1]
        done

        if [ $i -ge 5 ]; then
                echo "ERROR: nomad didn't come up. exit 1."
                exit 1
        fi
}

main "$@"
