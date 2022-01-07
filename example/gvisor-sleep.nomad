job "gvisor-sleep" {
  datacenters = ["dc1"]

  group "gvisor-sleep-group" {
    task "gvisor-sleep-task" {
      driver = "containerd-driver"

      config {
        image = "alpine:3.15.0"
        args = ["sleep", "infinity"]
        runtime = "runsc"
      }

      resources {
        cpu    = 64
        memory = 64
      }
    }
  }
}
