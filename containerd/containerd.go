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
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	etchosts "github.com/Roblox/nomad-driver-containerd/etchosts"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/contrib/seccomp"
	"github.com/containerd/containerd/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type ContainerConfig struct {
	Image                 containerd.Image
	ContainerName         string
	ContainerSnapshotName string
	NetworkNamespacePath  string
	SecretsDirSrc         string
	TaskDirSrc            string
	AllocDirSrc           string
	SecretsDirDest        string
	TaskDirDest           string
	AllocDirDest          string
	Env                   []string
	MemoryLimit           int64
	CPUShares             int64
}

func (d *Driver) isContainerdRunning() (bool, error) {
	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return d.client.IsServing(ctxWithTimeout)
}

func (d *Driver) getContainerdVersion() (containerd.Version, error) {
	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return d.client.Version(ctxWithTimeout)
}

func (d *Driver) pullImage(imageName string) (containerd.Image, error) {
	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 90*time.Second)
	defer cancel()

	return d.client.Pull(ctxWithTimeout, imageName, containerd.WithPullUnpack)
}

func (d *Driver) createContainer(containerConfig *ContainerConfig, config *TaskConfig) (containerd.Container, error) {
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

	opts = append(opts, oci.WithImageConfigArgs(containerConfig.Image, args))

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

	// Set current working directory (cwd).
	if config.Cwd != "" {
		opts = append(opts, oci.WithProcessCwd(config.Cwd))
	}

	// Set environment variables.
	opts = append(opts, oci.WithEnv(containerConfig.Env))

	// Set cgroups memory limit.
	opts = append(opts, oci.WithMemoryLimit(uint64(containerConfig.MemoryLimit)))

	// Set CPU Shares.
	opts = append(opts, oci.WithCPUShares(uint64(containerConfig.CPUShares)))

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

	// Setup host DNS (/etc/resolv.conf) into the container.
	if config.HostDNS {
		dnsMount := buildMountpoint("bind", "/etc/resolv.conf", "/etc/resolv.conf", []string{"rbind", "ro"})
		mounts = append(mounts, dnsMount)
	}

	// Setup "/secrets" (NOMAD_SECRETS_DIR) in the container.
	if containerConfig.SecretsDirSrc != "" && containerConfig.SecretsDirDest != "" {
		secretsMount := buildMountpoint("bind", containerConfig.SecretsDirDest, containerConfig.SecretsDirSrc, []string{"rbind", "rw"})
		mounts = append(mounts, secretsMount)
	}

	// Setup "/local" (NOMAD_TASK_DIR) in the container.
	if containerConfig.TaskDirSrc != "" && containerConfig.TaskDirDest != "" {
		taskMount := buildMountpoint("bind", containerConfig.TaskDirDest, containerConfig.TaskDirSrc, []string{"rbind", "rw"})
		mounts = append(mounts, taskMount)
	}

	// Setup "/alloc" (NOMAD_ALLOC_DIR) in the container.
	if containerConfig.AllocDirSrc != "" && containerConfig.AllocDirDest != "" {
		allocMount := buildMountpoint("bind", containerConfig.AllocDirDest, containerConfig.AllocDirSrc, []string{"rbind", "rw"})
		mounts = append(mounts, allocMount)
	}

	var extraHostsMount specs.Mount
	hostsFile := containerConfig.TaskDirSrc + "/etc_hosts"
	if len(config.ExtraHosts) > 0 {
		if config.HostNetwork {
			if err := etchosts.CopyEtcHosts(hostsFile); err != nil {
				return nil, err
			}
		} else {
			if err := etchosts.BuildEtcHosts(hostsFile); err != nil {
				return nil, err
			}
		}
		if err := etchosts.AddExtraHosts(hostsFile, config.ExtraHosts); err != nil {
			return nil, err
		}
		extraHostsMount = buildMountpoint("bind", "/etc/hosts", hostsFile, []string{"rbind", "rw"})
		mounts = append(mounts, extraHostsMount)
	} else if !config.HostNetwork {
		if err := etchosts.BuildEtcHosts(hostsFile); err != nil {
			return nil, err
		}
		extraHostsMount = buildMountpoint("bind", "/etc/hosts", hostsFile, []string{"rbind", "rw"})
		mounts = append(mounts, extraHostsMount)
	}

	if len(mounts) > 0 {
		opts = append(opts, oci.WithMounts(mounts))
	}

	// nomad use CNI plugins e.g bridge to setup a network (and network namespace) for the container.
	// CNI plugins need to be installed under /opt/cni/bin.
	// network namespace is created at /var/run/netns/<id>.
	// containerConfig.NetworkNamespacePath is the path to the network namespace, which
	// containerd joins to provide network for the container.
	// NOTE: Only bridge networking mode is supported at this point.
	if containerConfig.NetworkNamespacePath != "" {
		opts = append(opts, oci.WithLinuxNamespace(specs.LinuxNamespace{Type: specs.NetworkNamespace, Path: containerConfig.NetworkNamespacePath}))
	}

	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return d.client.NewContainer(
		ctxWithTimeout,
		containerConfig.ContainerName,
		containerd.WithRuntime(d.config.ContainerdRuntime, nil),
		containerd.WithNewSnapshot(containerConfig.ContainerSnapshotName, containerConfig.Image),
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
	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return d.client.LoadContainer(ctxWithTimeout, id)
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

	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return container.NewTask(ctxWithTimeout, cio.NewCreator(cio.WithStreams(nil, stdout, stderr)))
}

// FIFO's are named pipes in linux.
// openFIFO() opens the nomad task stdout/stderr pipes and returns the fd.
func openFIFO(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
}

func (d *Driver) getTask(container containerd.Container) (containerd.Task, error) {
	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return container.Task(ctxWithTimeout, cio.Load)
}
