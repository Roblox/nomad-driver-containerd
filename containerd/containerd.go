package containerd

import (
	"github.com/containerd/containerd"
)

func isContainerdRunning(c *containerd.Client) (bool, error) {
	return true, nil
}

func getContainerdVersion(c *containerd.Client) (string, error) {
	return "1.3.3", nil
}
