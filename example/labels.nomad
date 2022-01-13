job "labels" {
  datacenters = ["dc1"]

  group "labels-group" {
    task "labels-task" {
      driver = "containerd-driver"

      config {
        image = "alpine:3.15.0"
        args = ["sleep", "infinity"]

        labels {
          test = "labels"
        }
      }

      resources {
        cpu    = 64
        memory = 64
      }
    }
  }
}
