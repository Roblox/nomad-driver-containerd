
job "dns" {
  datacenters = ["dc1"]

  group "dns-group" {

    network {
        dns {
            servers = ["127.0.0.1", "127.0.0.2"]
            searches = ["internal.corp"]
            options = ["ndots:2"]
        }
    }

    task "dns-task" {
      driver = "containerd-driver"
      config {
        image           = "ubuntu:16.04"
        command         = "sleep"
        args            = ["600s"]
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
