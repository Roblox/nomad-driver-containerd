job "privileged-not-allowed" {
  datacenters = ["dc1"]

  group "privileged-not-allowed-group" {
    task "privileged-not-allowed-task" {
      driver = "containerd-driver"

      config {
        image           = "ubuntu:16.04"
        command         = "sleep"
        args            = ["600s"]
        privileged      = true
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
