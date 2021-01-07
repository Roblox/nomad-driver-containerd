job "capabilities" {
  datacenters = ["dc1"]

  group "capabilities-group" {
    task "capabilities-task" {
      driver = "containerd-driver"

      config {
        image           = "docker.io/library/ubuntu:16.04"
        command         = "sleep"
        args            = ["600s"]
        readonly_rootfs = true
        cap_add         = ["CAP_SYS_ADMIN", "CAP_IPC_OWNER", "CAP_IPC_LOCK"]
        cap_drop        = ["CAP_CHOWN", "CAP_SYS_CHROOT", "CAP_DAC_OVERRIDE"]
      }

      resources {
        cpu    = 500
        memory = 256
      }
    }
  }
}
