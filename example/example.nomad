job "example" {
  datacenters = ["dc1"]

  group "cache" {
    task "redis" {
      driver = "containerd-driver"

      config {
        image = "docker.io/library/redis:alpine"
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
