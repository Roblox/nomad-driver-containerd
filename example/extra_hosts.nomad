job "extra_hosts" {
  datacenters = ["dc1"]

  group "extra_hosts-group" {
    task "extra_hosts-task" {
      driver = "containerd-driver"
      config {
        image           = "ubuntu:16.04"
        extra_hosts     = ["postgres:127.0.1.1", "redis:127.0.1.2"]
        host_network    = true
        command         = "sleep"
        args            = ["600s"]
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
