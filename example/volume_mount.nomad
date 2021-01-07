job "volume_mount" {
  datacenters = ["dc1"]

  group "volume_mount-group" {

    volume "data" {
      type = "host"
      source = "s1"
      read_only = false
    }

    volume "read_only_data" {
      type = "host"
      source = "s1"
      read_only = true
    }

    task "volume_mount-task" {
      driver = "containerd-driver"
      config {
        image           = "docker.io/library/ubuntu:16.04"
        command         = "sleep"
        args            = ["600s"]
      }

      volume_mount {
        destination = "/tmp/t1"
        volume = "data"
      }

      volume_mount {
        destination = "/tmp/read_only_target"
        volume = "read_only_data"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
