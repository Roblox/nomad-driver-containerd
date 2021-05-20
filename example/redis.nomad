job "redis" {
  datacenters = ["dc1"]

  group "redis-group" {
    task "redis-task" {
      driver = "containerd-driver"

      config {
        image    = "redis:alpine"
        hostname = "foobar"
        seccomp  = true
        cwd      = "/home/redis"
      }

      resources {
        cpu        = 500
        memory     = 256
        memory_max = 512
      }
    }
  }
}
