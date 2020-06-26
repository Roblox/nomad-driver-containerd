package containerd

import (
	"fmt"

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

func (d *Driver) createContainer(image containerd.Image, containerName, containerSnapshotName, containerdRuntime string, env []string, config *TaskConfig) (containerd.Container, error) {
	if config.Command == "" && len(config.Args) > 0 {
		return nil, fmt.Errorf("Command is empty. Cannot set --args without --command.")
	}

	var args []string
	if config.Command != "" {
		args = append(args, config.Command)
	}

	if len(config.Args) > 0 {
		args = append(args, config.Args...)
	}

	var opts []oci.SpecOpts

	opts = append(opts, oci.WithImageConfigArgs(image, args))

	if config.Privileged {
		opts = append(opts, oci.WithPrivileged)
	}

	if config.ReadOnlyRootfs {
		opts = append(opts, oci.WithRootFSReadonly())
	}

	if len(config.CapAdd) > 0 {
		opts = append(opts, oci.WithAddedCapabilities(config.CapAdd))
	}

	if len(config.CapDrop) > 0 {
		opts = append(opts, oci.WithDroppedCapabilities(config.CapDrop))
	}

	opts = append(opts, oci.WithEnv(env))

	return d.client.NewContainer(
		d.ctxContainerd,
		containerName,
		containerd.WithRuntime(containerdRuntime, nil),
		containerd.WithNewSnapshot(containerSnapshotName, image),
		containerd.WithNewSpec(opts...),
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
