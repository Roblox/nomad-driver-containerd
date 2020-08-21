/*
Copyright 2020 Roblox Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0


Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package containerd

import (
	"fmt"
	"os"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
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

	// Command set by the user, to override entrypoint or cmd defined in the image.
	var args []string
	if config.Command != "" {
		args = append(args, config.Command)
	}

	// Arguments to the command set by the user.
	if len(config.Args) > 0 {
		args = append(args, config.Args...)
	}

	var opts []oci.SpecOpts

	opts = append(opts, oci.WithImageConfigArgs(image, args))

	// Enable privileged mode.
	if config.Privileged {
		opts = append(opts, oci.WithPrivileged)
	}

	// Launch container in read-only mode.
	if config.ReadOnlyRootfs {
		opts = append(opts, oci.WithRootFSReadonly())
	}

	// Add capabilities.
	if len(config.CapAdd) > 0 {
		opts = append(opts, oci.WithAddedCapabilities(config.CapAdd))
	}

	// Drop capabilities.
	if len(config.CapDrop) > 0 {
		opts = append(opts, oci.WithDroppedCapabilities(config.CapDrop))
	}

	// Set environment variables.
	opts = append(opts, oci.WithEnv(env))

	// Add linux devices into the container.
	for _, device := range config.Devices {
		opts = append(opts, oci.WithLinuxDevice(device, "rwm"))
	}

	// Set mounts. fstab style mount options are supported.
	// List of all supported mount options.
	// https://github.com/containerd/containerd/blob/master/mount/mount_linux.go#L187-L211
	mounts := make([]specs.Mount, 0)
	for _, mount := range config.Mounts {
		if (mount.Type == "bind" || mount.Type == "volume") && len(mount.Options) <= 0 {
			return nil, fmt.Errorf("Options cannot be empty for mount type: %s. You need to atleast pass rbind and ro.", mount.Type)
		}

		m := specs.Mount{}
		m.Type = mount.Type
		m.Destination = mount.Target
		m.Source = mount.Source
		m.Options = mount.Options
		mounts = append(mounts, m)
	}

	if len(mounts) > 0 {
		opts = append(opts, oci.WithMounts(mounts))
	}

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

func (d *Driver) createTask(container containerd.Container, stdoutPath, stderrPath string) (containerd.Task, error) {
	stdout, err := openFIFO(stdoutPath)
	if err != nil {
		return nil, err
	}

	stderr, err := openFIFO(stderrPath)
	if err != nil {
		return nil, err
	}

	return container.NewTask(d.ctxContainerd, cio.NewCreator(cio.WithStreams(nil, stdout, stderr)))
}

// FIFO's are named pipes in linux.
// openFIFO() opens the nomad task stdout/stderr pipes and returns the fd.
func openFIFO(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
}

func (d *Driver) getTask(container containerd.Container) (containerd.Task, error) {
	return container.Task(d.ctxContainerd, cio.Load)
}
