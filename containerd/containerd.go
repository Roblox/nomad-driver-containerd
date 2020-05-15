package containerd

import (
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
)

func (d *Driver) isContainerdRunning() (bool, error) {
	return d.client.IsServing(d.ctxContainerd)
}

func (d *Driver) getContainerdVersion() (containerd.Version, error) {
	return d.client.Version(d.ctxContainerd)
}

func (d *Driver) pullImage(imageName string) (containerd.Image, error) {
	return d.client.Pull(d.ctxContainerd, imageName, containerd.WithPullUnpack)
}

func (d *Driver) createContainer(image containerd.Image, containerName, containerSnapshotName, containerdRuntime string) (containerd.Container, error) {
	return d.client.NewContainer(
		d.ctxContainerd,
		containerName,
		containerd.WithRuntime(containerdRuntime, nil),
		containerd.WithNewSnapshot(containerSnapshotName, image),
		containerd.WithNewSpec(oci.WithImageConfig(image)),
	)
}

func (d *Driver) loadContainer(id string) (containerd.Container, error) {
	return d.client.LoadContainer(d.ctxContainerd, id)
}

func (d *Driver) createTask(container containerd.Container) (containerd.Task, error) {
	return container.NewTask(d.ctxContainerd, cio.NewCreator(cio.WithStdio))
}

func (d *Driver) getTask(container containerd.Container) (containerd.Task, error) {
	return container.Task(d.ctxContainerd, cio.Load)
}
