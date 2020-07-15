#!/bin/bash

set -euo pipefail

export CONTAINERD_VERSION=1.3.4

main() {
  echo "INFO: Welcome! nomad-driver-containerd setup."
  check_root
  check_os
  check_nomad
  check_golang

  echo "WARN: Some installation steps are time consuming. Please be patient!"

  # Save present working directory (pwd).
  curr_dir=$(echo $PWD)

  if systemctl -q is-active "containerd.service"; then
     echo "WARN: Containerd detected on the system."
     read -p "INFO: Backup existing containerd and deploy containerd-${CONTAINERD_VERSION} (Y/N)? Press Y to continue. " -n 1 -r
     echo
     if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "INFO: Aborting setup..."
        exit 0
     fi
     systemctl stop containerd
     if [ -f "/lib/systemd/system/containerd.service" ]; then
        echo "INFO: Backup containerd systemd unit /lib/systemd/system/containerd.service."
        mv /lib/systemd/system/containerd.service /lib/systemd/system/containerd.service.bkp
        echo "WARN: Backup file saved at: /lib/systemd/system/containerd.service.bkp"
     fi
  fi
  setup_containerd

  read -p "INFO: Setup nomad server + nomad-driver-containerd (Y/N)? Press Y to continue. " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
     echo "INFO: Aborting setup..."
     cleanup
     exit 0
  fi
  echo "INFO: Cleanup any old binaries."
  make clean >/dev/null 2>&1
  echo "INFO: Build nomad-driver-containerd binary: containerd-driver."
  make build >/dev/null 2>&1 || (cleanup && exit 1)
  echo "INFO: Create plugin-dir for containerd-driver: /tmp/nomad-driver-containerd."
  mkdir -p /tmp/nomad-driver-containerd || (cleanup && exit 1)
  echo "INFO: Move containerd-driver to /tmp/nomad-driver-containerd."
  mv containerd-driver /tmp/nomad-driver-containerd || (cleanup && exit 1)
  drop_nomad_unit_file $curr_dir
  echo "INFO: Reload nomad.service systemd unit."
  systemctl daemon-reload
  echo "INFO: Starting nomad server + nomad-driver-containerd."
  systemctl start nomad || (cleanup && exit 1)
  if ! systemctl -q is-active "nomad.service"; then
     echo "ERROR: nomad.service didn't come up. journalctl -u nomad.service for more info."
     exit 1
  fi
  echo "INFO: Setup finished successfully."
}

cleanup() {
  echo "INFO: Starting cleanup."
  pushd $curr_dir >/dev/null 2>&1
  if [ -f "/lib/systemd/system/containerd.service.bkp" ]; then
     if systemctl -q is-active "containerd.service"; then
	echo "INFO: Stopping containerd."
        systemctl stop containerd.service
     fi

     if [ -f "/tmp/containerd.service" ]; then
        echo "INFO: Cleanup /tmp/containerd.service."
	rm -f /tmp/containerd.service
     fi

     if [ -f "/lib/systemd/system/containerd.service" ]; then
        echo "INFO: Cleanup: /lib/systemd/system/containerd.service."
	rm -f /lib/systemd/system/containerd.service
     fi

     if [ -f "/tmp/containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz" ]; then
        echo "INFO: Cleanup: /tmp/containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz."
	rm -f /tmp/containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz
     fi
  fi

  if systemctl -q is-active "nomad.service"; then
     echo "INFO: Stopping nomad server+nomad-driver-containerd."
     systemctl stop nomad
  fi

  if [ -f "$curr_dir/nomad.service" ]; then
     echo "INFO: Cleanup $curr_dir/nomad.service."
     rm -f $curr_dir/nomad.service
  fi

  if [ -f "/lib/systemd/system/nomad.service" ]; then
     echo "INFO: Cleanup: /lib/systemd/system/nomad.service."
     rm -f /lib/systemd/system/nomad.service
  fi

  echo "INFO: Cleanup /tmp/nomad-driver-containerd."
  rm -rf /tmp/nomad-driver-containerd

  echo "INFO: Cleanup containerd-driver binary."
  make clean >/dev/null 2>&1
  popd >/dev/null 2>&1
  echo "INFO: Cleanup complete."
}

setup_containerd() {
  read -p "INFO: Download containerd (Y/N)? Press Y to continue. " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
     echo "INFO: Aborting setup..."
     cleanup
     exit 0
  fi
  pushd /tmp >/dev/null 2>&1
  curl -L --silent -o containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz || (cleanup && exit 1)
  tar -C /usr/local -xzf containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz || (cleanup && exit 1)
  rm -f containerd-${CONTAINERD_VERSION}.linux-amd64.tar.gz
  read -p "INFO: Drop systemd unit containerd.service into /lib/systemd/system/containerd.service (Y/N)? Press Y to continue. " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
     echo "INFO: Aborting setup..."
     cleanup
     exit 0
  fi
  drop_containerd_unit_file
  echo "INFO: Reload containerd.service systemd unit."
  systemctl daemon-reload
  echo "INFO: Starting containerd daemon."
  systemctl start containerd || (cleanup && exit 1)
  popd >/dev/null 2>&1
  if ! systemctl -q is-active "containerd.service"; then
     echo "ERROR: containerd.service didn't come up. journalctl -u containerd.service for more info."
     exit 1
  fi
}

drop_nomad_unit_file() {
  local nomad=$(which nomad)
  # Drop nomad server (dev) + nomad-driver-containerd systemd unit file into /lib/systemd/system.
  cat << EOF > nomad.service
# /lib/systemd/system/nomad.service
[Unit]
Description=nomad server (dev) + nomad-driver-containerd
Documentation=https://nomadproject.io
After=network.target containerd.service

[Service]
ExecStart=$nomad agent -dev -bind=0.0.0.0 -config=$1/example/agent.hcl -plugin-dir=/tmp/nomad-driver-containerd
KillMode=process
Delegate=yes
LimitNOFILE=1048576
LimitNPROC=infinity
LimitCORE=infinity
TasksMax=infinity

[Install]
WantedBy=multi-user.target
EOF
mv nomad.service /lib/systemd/system/nomad.service
}

drop_containerd_unit_file() {
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
mv containerd.service /lib/systemd/system/containerd.service
}

check_golang() {
  set +e
  go version >/dev/null 2>&1
  rc=$?
  set -e
  if [ $rc -ne 0 ];then
     echo "ERROR: Golang is missing. Please install golang >=1.11 to continue with the setup."
     exit 1
  fi
}

check_nomad() {
  set +e
  which nomad >/dev/null 2>&1
  rc=$?
  set -e
  if [ $rc -ne 0 ];then
     echo "ERROR: Nomad is missing. Please install nomad >=0.11 to continue with the setup."
     exit 1
  fi
}

check_root() {
  if [ $(id -u) != 0 ]; then
    echo "ERROR: Run as root user."
    exit 1
  fi
}

check_os() {
  set +e
  cat /etc/os-release|grep -q -i "Ubuntu"
  rc=$?
  set -e
  if [ $rc -ne 0 ];then
     echo "ERROR: Unsupported host OS. Run tests on Ubuntu."
     exit 1
  fi
}

main "$@"
