job "stress-swap" {
  datacenters = ["dc1"]
  type = "batch"

  group "stress-group" {
    task "stress-swap" {
      driver = "containerd-driver"

      config {
        image = "mohsenmottaghi/container-stress"
        command = "stress"
        args = ["--vm", "5", "--vm-bytes", "256M", "--timeout", "20s"]
        memory_swap = "2048m"
        memory_swappiness = 90
      }

      resources {
        cpu    = 64
        memory = 1024
      }
    }
  }
}