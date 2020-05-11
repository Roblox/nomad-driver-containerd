package containerd

import (
	"github.com/containerd/containerd"
)

func (d *Driver) isContainerdRunning() (bool, error) {
	return d.client.IsServing(d.ctxContainerd)
}

func (d *Driver) getContainerdVersion() (containerd.Version, error) {
	return d.client.Version(d.ctxContainerd)
}

func (d *Driver) pullOCIImage(imageName string) (containerd.Image, error) {
	return d.client.Pull(d.ctxContainerd, imageName, containerd.WithPullUnpack)
}
