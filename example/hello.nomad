job "hello" {
  datacenters = ["dc1"]

  group "hello-group" {
    task "hello-task" {
      driver = "containerd-driver"

      config {
        image = "docker.io/shm32/hello:world"
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
