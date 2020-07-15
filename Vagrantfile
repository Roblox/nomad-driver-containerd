# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.box = "hashicorp/bionic64"

  config.vm.synced_folder ".", "/home/vagrant/go/src/nomad-driver-containerd"

  config.vm.network "forwarded_port", guest: 4646, host: 4646, host_ip: "127.0.0.1"

  config.vm.provision "shell", inline: <<-SHELL
    apt-get update
    apt-get install -y unzip gcc runc
    if [ ! -f "/usr/local/bin/go" ]; then
      # Install golang 1.14.3
      curl -s -L -o go1.14.3.linux-amd64.tar.gz https://dl.google.com/go/go1.14.3.linux-amd64.tar.gz
      sudo tar -C /usr/local -xzf go1.14.3.linux-amd64.tar.gz
      sudo chmod +x /usr/local/go
      sudo cp /usr/local/go/bin/go /usr/local/bin/go
      rm -f go1.14.3.linux-amd64.tar.gz
    fi
    if [ ! -f "/usr/bin/nomad" ]; then
      wget --quiet https://releases.hashicorp.com/nomad/0.11.3/nomad_0.11.3_linux_amd64.zip 
      unzip nomad_0.11.3_linux_amd64.zip
      chmod +x nomad
      cp nomad /usr/bin/nomad
    fi
  SHELL
end
