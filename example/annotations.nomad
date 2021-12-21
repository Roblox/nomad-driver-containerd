job "annotations" {
  datacenters = ["dc1"]

  group "annotations-group" {
    task "annotations-task" {
      driver = "containerd-driver"

      config {
        image = "alpine:3.15.0"
        args = ["sleep", "infinity"]

        annotations {
          test = "annotations"
        }
      }

      resources {
        cpu    = 64
        memory = 64
      }
    }
  }
}
