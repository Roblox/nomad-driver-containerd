job "entrypoint" {
  datacenters = ["dc1"]

  group "entrypoint-group" {
    task "entrypoint-task" {
      driver = "containerd-driver"

      config {
        image       = "ubuntu:16.04"
        entrypoint  = ["/bin/bash"]
        args        = ["-c", "for i in {1..100}; do echo container1 container2; sleep 1s; done"]
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
