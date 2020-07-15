#!/bin/bash

set -euo pipefail

main() {
  echo "INFO: Welcome! nomad-driver-containerd setup."
  echo "WARN: Some installation steps are time consuming. Please be patient!"

  # Save project root directory.
  root_dir="/home/vagrant/go/src/nomad-driver-containerd"
  pushd $root_dir >/dev/null 2>&1
  echo "INFO: Drop systemd unit containerd.service into /lib/systemd/system/containerd.service."
  drop_containerd_unit_file
  echo "INFO: Reload containerd.service systemd unit."
  systemctl daemon-reload
  echo "INFO: Starting containerd daemon."
  systemctl start containerd
  if ! systemctl -q is-active "containerd.service"; then
     echo "ERROR: containerd.service didn't come up. journalctl -u containerd.service for more info."
     exit 1
  fi

  echo "INFO: Setup nomad server + nomad-driver-containerd."
  export PATH=$PATH:/usr/local/go/bin
  echo "INFO: Cleanup any old binaries."
  make clean >/dev/null 2>&1
  echo "INFO: Build nomad-driver-containerd binary: containerd-driver."
  make build >/dev/null 2>&1
  echo "INFO: Create plugin-dir for containerd-driver: /tmp/nomad-driver-containerd."
  mkdir -p /tmp/nomad-driver-containerd
  echo "INFO: Move containerd-driver to /tmp/nomad-driver-containerd."
  mv containerd-driver /tmp/nomad-driver-containerd
  drop_nomad_unit_file $root_dir
  echo "INFO: Reload nomad.service systemd unit."
  systemctl daemon-reload
  echo "INFO: Starting nomad server + nomad-driver-containerd."
  systemctl start nomad
  if ! systemctl -q is-active "nomad.service"; then
     echo "ERROR: nomad.service didn't come up. journalctl -u nomad.service for more info."
     exit 1
  fi
  popd >/dev/null 2>&1
  echo "INFO: Setup finished successfully."
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

main "$@"
