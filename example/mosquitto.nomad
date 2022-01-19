job "mosquitto" {
  datacenters = ["dc1"]

  group "msq-group" {
    task "msq-task" {
      driver = "containerd-driver"

      config {
        image           = "ubuntu:16.04"
        command         = "sleep"
        args            = ["600s"]
        mounts = [
           {
                type    = "bind"
                target  = "/mosquitto/config/mosquitto.conf"
                source  = "local/mosquitto.conf"
                options = ["rbind", "rw"]
           }
        ]
      }

      template {
        destination = "local/mosquitto.conf"
        data = <<EOF
bind_address 0.0.0.0
allow_anonymous true
persistence true
persistence_location /mosquitto/data/
log_dest stdout
EOF
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
