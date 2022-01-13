job "gpu-nvidia-smi-sleep" {
  datacenters = ["dc1"]

  group "gpu-nvidia-smi-sleepgroup" {
    task "gpu-nvidia-smi-sleep-task" {
      driver = "containerd-driver"

      # https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/user-guide.html#driver-capabilities
      env {
        NVIDIA_DRIVER_CAPABILITIES = "utility"
      }

      config {
        image = "debian:stretch"
        args = ["nvidia-smi",";", "sleep", "infinity"]
      }

      resources {
        cpu    = 64
        memory = 64

        # https://www.nomadproject.io/docs/devices/external/nvidia
        # https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html#containerd
        device "nvidia/gpu" {
          count = 1
        }
      }
    }
  }
}
