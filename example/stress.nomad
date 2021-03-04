job "stress" {
  datacenters = ["dc1"]

  group "stress-group" {
    task "stress-task" {
      driver = "containerd-driver"

      config {
        image = "shm32/stress:1.0"
      }

      restart {
        attempts = 5
        delay    = "30s"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
