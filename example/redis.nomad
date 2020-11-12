job "redis" {
  datacenters = ["dc1"]

  group "redis-group" {
    task "redis-task" {
      driver = "containerd-driver"

      config {
        image   = "docker.io/library/redis:alpine"
        seccomp = true
        cwd     = "/home/redis"
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
