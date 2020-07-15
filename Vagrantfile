# Specify minimum Vagrant version and Vagrant API version
Vagrant.require_version ">= 1.6.0"
VAGRANTFILE_API_VERSION = "2"

# Create box
Vagrant.configure("2") do |config|
  config.vm.define "containerd-linux"
  config.vm.box = "hashicorp/bionic64"
  config.vm.synced_folder ".", "/home/vagrant/go/src/nomad-driver-containerd"
  config.ssh.extra_args = ["-t", "cd /home/vagrant/go/src/nomad-driver-containerd; bash --login"]
  config.vm.network "forwarded_port", guest: 4646, host: 4646, host_ip: "127.0.0.1"
  config.vm.provider "virtualbox" do |vb|
      vb.name = "containerd-linux"
      vb.cpus = 2
      vb.memory = 2048
  end
  config.vm.provision "shell", inline: <<-SHELL
    apt-get update
    apt-get install -y unzip gcc runc
    echo "export GOPATH=/home/vagrant/go" >> /home/vagrant/.bashrc
    echo "export PATH=$PATH:/usr/local/go/bin" >> /home/vagrant/.bashrc
    source /home/vagrant/.bashrc

    # Install golang-1.14.3
    if [ ! -f "/usr/local/go/bin/go" ]; then
      curl -s -L -o go1.14.3.linux-amd64.tar.gz https://dl.google.com/go/go1.14.3.linux-amd64.tar.gz
      sudo tar -C /usr/local -xzf go1.14.3.linux-amd64.tar.gz
      sudo chmod +x /usr/local/go
      rm -f go1.14.3.linux-amd64.tar.gz
    fi

    # Install nomad-0.11.3
    if [ ! -f "/usr/bin/nomad" ]; then
      wget --quiet https://releases.hashicorp.com/nomad/0.11.3/nomad_0.11.3_linux_amd64.zip 
      unzip nomad_0.11.3_linux_amd64.zip -d /usr/bin
      chmod +x /usr/bin/nomad
      rm -f nomad_0.11.3_linux_amd64.zip
    fi

    # Install containerd-1.3.4
    if [ ! -f "/usr/local/bin/containerd" ]; then
       curl -L --silent -o containerd-1.3.4.linux-amd64.tar.gz https://github.com/containerd/containerd/releases/download/v1.3.4/containerd-1.3.4.linux-amd64.tar.gz
       tar -C /usr/local -xzf containerd-1.3.4.linux-amd64.tar.gz
       rm -f containerd-1.3.4.linux-amd64.tar.gz
    fi

    # Run setup
    cd /home/vagrant/go/src/nomad-driver-containerd/vagrant
    ./setup.sh
  SHELL
end
