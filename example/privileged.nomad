job "privileged" {
  datacenters = ["dc1"]

  group "privileged-group" {
    task "privileged-task" {
      driver = "containerd-driver"

      config {
        image           = "docker.io/library/ubuntu:16.04"
        command         = "sleep"
        args            = ["600s"]
        privileged      = true
        devices         = [
            "/dev/loop0",
            "/dev/loop1"
        ]
        mounts = [
           {
                type = "bind"
                target = "/tmp/t1"
                source = "/tmp/s1"
                options = ["rbind", "ro"]
           }
        ]
      }

      resources {
        cpu    = 500
        memory = 256
        network {
          mbits = 10
        }
      }
    }
  }
}
