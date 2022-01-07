# Specify minimum Vagrant version and Vagrant API version
Vagrant.require_version ">= 1.6.0"
VAGRANTFILE_API_VERSION = "2"

# Create box
Vagrant.configure("2") do |config|
  config.vm.define "containerd-linux"
  config.vm.box = "hashicorp/bionic64"
  config.vm.provider "libvirt" do |v, override|
    override.vm.box = "generic/debian10"
    override.vm.synced_folder ".", "/home/vagrant/go/src/github.com/Roblox/nomad-driver-containerd", type: "nfs", nfs_version: 4, nfs_udp: false
  end
  config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/Roblox/nomad-driver-containerd"
  config.ssh.extra_args = ["-t", "cd /home/vagrant/go/src/github.com/Roblox/nomad-driver-containerd; bash --login"]
  config.vm.network "forwarded_port", guest: 4646, host: 4646, host_ip: "127.0.0.1"
  config.vm.provider "virtualbox" do |vb|
      vb.name = "containerd-linux"
      vb.cpus = 2
      vb.memory = 2048
  end
  config.vm.provision "shell", inline: <<-SHELL
    apt-get update
    apt-get install -y unzip gcc runc jq
    echo "export GOPATH=/home/vagrant/go" >> /home/vagrant/.bashrc
    echo "export PATH=$PATH:/usr/local/go/bin" >> /home/vagrant/.bashrc
    echo "export CONTAINERD_NAMESPACE=nomad" >> /home/vagrant/.bashrc
    source /home/vagrant/.bashrc
    # without keeping HOME env, 'sudo make test' will try to find files under /root/go/
    echo "Defaults env_keep += HOME" | sudo tee /etc/sudoers.d/keep_home

    # Install golang-1.17
    if [ ! -f "/usr/local/go/bin/go" ]; then
      curl -s -L -o go1.17.linux-amd64.tar.gz https://dl.google.com/go/go1.17.linux-amd64.tar.gz
      sudo tar -C /usr/local -xzf go1.17.linux-amd64.tar.gz
      sudo chmod +x /usr/local/go
      rm -f go1.17.linux-amd64.tar.gz
    fi

    # Install nomad-1.1.4
    if [ ! -f "/usr/bin/nomad" ]; then
      wget --quiet https://releases.hashicorp.com/nomad/1.1.4/nomad_1.1.4_linux_amd64.zip
      unzip nomad_1.1.4_linux_amd64.zip -d /usr/bin
      chmod +x /usr/bin/nomad
      rm -f nomad_1.1.4_linux_amd64.zip
    fi

    # Install containerd-1.5.5
    if [ ! -f "/usr/local/bin/containerd" ]; then
       curl -L --silent -o containerd-1.5.5-linux-amd64.tar.gz https://github.com/containerd/containerd/releases/download/v1.5.5/containerd-1.5.5-linux-amd64.tar.gz
       tar -C /usr/local -xzf containerd-1.5.5-linux-amd64.tar.gz
       rm -f containerd-1.5.5-linux-amd64.tar.gz
    fi

    # Install nerdctl 0.11.1
    if [ ! -f "/usr/local/bin/nerdctl" ]; then
       curl -L --silent -o nerdctl-0.11.1-linux-amd64.tar.gz https://github.com/containerd/nerdctl/releases/download/v0.11.1/nerdctl-0.11.1-linux-amd64.tar.gz
       tar -C /usr/local/bin -xzf nerdctl-0.11.1-linux-amd64.tar.gz
       rm -f nerdctl-0.11.1-linux-amd64.tar.gz
    fi

    # Install gvisor/runsc
    if [ ! -f "/usr/local/bin/runsc" ]; then
        ARCH=$(uname -m)
        URL=https://storage.googleapis.com/gvisor/releases/release/latest/${ARCH}
        wget -q ${URL}/runsc ${URL}/runsc.sha512 \
        ${URL}/containerd-shim-runsc-v1 ${URL}/containerd-shim-runsc-v1.sha512
        sha512sum -c runsc.sha512 \
        -c containerd-shim-runsc-v1.sha512

        rm -f *.sha512

        chmod a+rx runsc containerd-shim-runsc-v1
        mv runsc containerd-shim-runsc-v1 /usr/local/bin
    fi

    # Create source directory for privileged.nomad example job.
    mkdir -p /tmp/s1

    # Prepare nomad host volume
    mkdir -p /tmp/host_volume/s1

    # Run setup
    cd /home/vagrant/go/src/github.com/Roblox/nomad-driver-containerd/vagrant
    ./setup.sh
  SHELL
end
