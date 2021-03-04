job "signal" {
  datacenters = ["dc1"]

  group "signal-group" {
    task "signal-task" {
      driver = "containerd-driver"

      config {
        image = "shm32/signal_handler:1.0"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
