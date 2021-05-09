job "auth" {
  datacenters = ["dc1"]

  reschedule {
    delay = "9s"
    delay_function = "constant"
    unlimited = true
  }

  group "auth-group" {
    task "auth-task" {
      driver = "containerd-driver"

      config {
        image = "shm32/hello-world:private"
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
