log_level = "INFO"

plugin "containerd-driver" {
  config {
    enabled = true
    containerd_runtime = "io.containerd.runtime.v1.linux"
  }
}
