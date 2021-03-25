job "entrypoint" {
  datacenters = ["dc1"]

  group "entrypoint-group" {
    task "entrypoint-task" {
      driver = "containerd-driver"

      config {
        image      = "ubuntu:16.04"
        entrypoint = ["/bin/echo"]
        args       = ["container1", "container2"]
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
