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
	"github.com/containerd/containerd/contrib/seccomp"
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

func (d *Driver) createContainer(image containerd.Image, containerName, containerSnapshotName, containerdRuntime, netnsPath, secretsDir, taskDir, allocDir string, env []string, memoryLimit, cpuShares int64, config *TaskConfig) (containerd.Container, error) {
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

	if !config.Seccomp && config.SeccompProfile != "" {
		return nil, fmt.Errorf("seccomp must be set to true, if using a custom seccomp_profile.")
	}

	// Enable default (or custom) seccomp profile.
	// Allowed syscalls for the default seccomp profile: https://github.com/containerd/containerd/blob/master/contrib/seccomp/seccomp_default.go#L51-L390
	if config.Seccomp {
		if config.SeccompProfile != "" {
			opts = append(opts, seccomp.WithProfile(config.SeccompProfile))
		} else {
			opts = append(opts, seccomp.WithDefaultProfile())
		}
	}

	// Launch container in read-only mode.
	if config.ReadOnlyRootfs {
		opts = append(opts, oci.WithRootFSReadonly())
	}

	// Enable host network.
	// WithHostHostsFile bind-mounts the host's /etc/hosts into the container as readonly.
	// WithHostResolvconf bind-mounts the host's /etc/resolv.conf into the container as readonly.
	if config.HostNetwork {
		opts = append(opts, oci.WithHostNamespace(specs.NetworkNamespace), oci.WithHostHostsFile, oci.WithHostResolvconf)
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

	// Set cgroups memory limit.
	opts = append(opts, oci.WithMemoryLimit(uint64(memoryLimit)))

	// Set CPU Shares.
	opts = append(opts, oci.WithCPUShares(uint64(cpuShares)))

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

		m := buildMountpoint(mount.Type, mount.Target, mount.Source, mount.Options)
		mounts = append(mounts, m)
	}

	// Setup "/secrets" (NOMAD_SECRETS_DIR) in the container.
	if secretsDir != "" {
		secretsMount := buildMountpoint("bind", "/secrets", secretsDir, []string{"rbind", "ro"})
		mounts = append(mounts, secretsMount)
	}

	// Setup "/local" (NOMAD_TASK_DIR) in the container.
	if taskDir != "" {
		taskMount := buildMountpoint("bind", "/local", taskDir, []string{"rbind", "ro"})
		mounts = append(mounts, taskMount)
	}

	// Setup "/alloc" (NOMAD_ALLOC_DIR) in the container.
	if allocDir != "" {
		allocMount := buildMountpoint("bind", "/alloc", allocDir, []string{"rbind", "ro"})
		mounts = append(mounts, allocMount)
	}

	if len(mounts) > 0 {
		opts = append(opts, oci.WithMounts(mounts))
	}

	// nomad use CNI plugins e.g bridge to setup a network (and network namespace) for the container.
	// CNI plugins need to be installed under /opt/cni/bin.
	// network namespace is created at /var/run/netns/<id>.
	// netnsPath is the path to the network namespace, which containerd joins to provide network
	// for the container.
	// NOTE: Only bridge networking mode is supported at this point.
	if netnsPath != "" {
		opts = append(opts, oci.WithLinuxNamespace(specs.LinuxNamespace{Type: specs.NetworkNamespace, Path: netnsPath}))
	}

	return d.client.NewContainer(
		d.ctxContainerd,
		containerName,
		containerd.WithRuntime(containerdRuntime, nil),
		containerd.WithNewSnapshot(containerSnapshotName, image),
		containerd.WithNewSpec(opts...),
	)
}

// buildMountpoint builds the mount point for the container.
func buildMountpoint(mountType, mountTarget, mountSource string, mountOptions []string) specs.Mount {
	m := specs.Mount{}
	m.Type = mountType
	m.Destination = mountTarget
	m.Source = mountSource
	m.Options = mountOptions
	return m
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
