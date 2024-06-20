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
	"strings"
	"time"

	etchosts "github.com/Roblox/nomad-driver-containerd/etchosts"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/contrib/seccomp"
	"github.com/containerd/containerd/oci"
	refdocker "github.com/containerd/containerd/reference/docker"
	remotesdocker "github.com/containerd/containerd/remotes/docker"
	"github.com/docker/go-units"
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
	MemoryHardLimit       int64
	CPUShares             int64
	User                  string
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

type CredentialsOpt func(string) (string, string, error)

func (d *Driver) parshAuth(auth *RegistryAuth) CredentialsOpt {
	return func(string) (string, string, error) {
		var username, password string
		if d.config.Auth.Username != "" && d.config.Auth.Password != "" {
			username = d.config.Auth.Username
			password = d.config.Auth.Password
		}

		// Job auth will take precedence over plugin auth options.
		if auth.Username != "" && auth.Password != "" {
			username = auth.Username
			password = auth.Password
		}
		return username, password, nil
	}
}

func withResolver(creds CredentialsOpt) containerd.RemoteOpt {
	resolver := remotesdocker.NewResolver(remotesdocker.ResolverOptions{
		Hosts: remotesdocker.ConfigureDefaultRegistries(remotesdocker.WithAuthorizer(
			remotesdocker.NewDockerAuthorizer(remotesdocker.WithAuthCreds(creds)))),
	})
	return containerd.WithResolver(resolver)
}

func withFileLimit(maxOpenFiles uint64) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, spec *oci.Spec) error {
		newRlimits := []specs.POSIXRlimit{{
			Type: "RLIMIT_NOFILE",
			Hard: maxOpenFiles,
			Soft: maxOpenFiles,
		}}

		// Copy existing rlimits excluding previous RLIMIT_NOFILE
		for _, rlimit := range spec.Process.Rlimits {
			if rlimit.Type != "RLIMIT_NOFILE" {
				newRlimits = append(newRlimits, rlimit)
			}
		}

		spec.Process.Rlimits = newRlimits

		return nil
	}
}

func (d *Driver) pullImage(imageName, imagePullTimeout string, auth *RegistryAuth) (containerd.Image, error) {
	pullTimeout, err := time.ParseDuration(imagePullTimeout)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse image_pull_timeout: %v", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, pullTimeout)
	defer cancel()

	named, err := refdocker.ParseDockerRef(imageName)
	if err != nil {
		return nil, err
	}

	pullOpts := []containerd.RemoteOpt{
		containerd.WithPullUnpack,
		withResolver(d.parshAuth(auth)),
	}

	return d.client.Pull(ctxWithTimeout, named.String(), pullOpts...)
}

func (d *Driver) createContainer(containerConfig *ContainerConfig, config *TaskConfig) (containerd.Container, error) {
	if config.Command != "" && config.Entrypoint != nil {
		return nil, fmt.Errorf("Both command and entrypoint are set. Only one of them needs to be set.")
	}

	// Entrypoint or Command set by the user, to override entrypoint or cmd defined in the image.
	var args []string
	if config.Command != "" {
		args = append(args, config.Command)
	} else if config.Entrypoint != nil && config.Entrypoint[0] != "" {
		args = append(args, config.Entrypoint...)
	}

	// Arguments to the command set by the user.
	if len(config.Args) > 0 {
		args = append(args, config.Args...)
	}

	var opts []oci.SpecOpts

	if config.Entrypoint != nil {
		opts = append(opts, oci.WithImageConfig(containerConfig.Image))
		// WithProcessArgs replaces the args on the generated spec.
		opts = append(opts, oci.WithProcessArgs(args...))
	} else {
		// WithImageConfigArgs configures the spec to from the configuration of an Image
		// with additional args that replaces the CMD of the image.
		opts = append(opts, oci.WithImageConfigArgs(containerConfig.Image, args))
	}

	if !d.config.AllowPrivileged && config.Privileged {
		return nil, fmt.Errorf("Running privileged jobs are not allowed. Set allow_privileged to true in plugin config to allow running privileged jobs.")
	}

	// Enable privileged mode.
	if config.Privileged {
		opts = append(opts, oci.WithPrivileged, oci.WithAllDevicesAllowed, oci.WithHostDevices, oci.WithNewPrivileges)
	}

	// WithPidsLimit sets the container's pid limit or maximum
	if config.PidsLimit > 0 {
		opts = append(opts, oci.WithPidsLimit(config.PidsLimit))
	}

	if config.PidMode != "" {
		if strings.ToLower(config.PidMode) != "host" {
			return nil, fmt.Errorf("Invalid pid_mode. Set pid_mode=host to enable host pid namespace.")
		} else {
			opts = append(opts, oci.WithHostNamespace(specs.PIDNamespace))
		}
	}

	// Set the resource limit for open file descriptors
	if config.FileLimit > 0 {
		opts = append(opts, withFileLimit(uint64(config.FileLimit)))
	}

	// Size of /dev/shm
	if len(config.ShmSize) > 0 {
		shmBytes, err := units.RAMInBytes(config.ShmSize)
		if err != nil {
			return nil, fmt.Errorf("Error in setting shm_size: %v", err)
		}
		opts = append(opts, oci.WithDevShmSize(shmBytes/1024))
	}

	// Set sysctls
	if len(config.Sysctl) > 0 {
		opts = append(opts, WithSysctls(config.Sysctl))
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
	opts = append(opts, WithMemoryLimits(containerConfig.MemoryLimit, containerConfig.MemoryHardLimit))

	// Set CPU Shares.
	opts = append(opts, oci.WithCPUShares(uint64(containerConfig.CPUShares)))

	// Set Hostname
	hostname := containerConfig.ContainerName
	if config.Hostname != "" {
		hostname = config.Hostname
	}
	opts = append(opts, oci.WithHostname(hostname))

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

		// Allow paths relative to $NOMAD_TASK_DIR.
		// More details: https://github.com/Roblox/nomad-driver-containerd/issues/116#issuecomment-983171458
		if mount.Type == "bind" && strings.HasPrefix(mount.Source, "local") {
			mount.Source = containerConfig.TaskDirSrc + mount.Source[5:]
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

	// User will specify extra_hosts to be added to container's /etc/hosts.
	// If host_network=true, extra_hosts will be added to host's /etc/hosts.
	// If host_network=false, extra hosts will be added to the default /etc/hosts provided to the container.
	// If the user doesn't set anything (host_network, extra_hosts), a default /etc/hosts will be provided to the container.
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

	if containerConfig.User != "" {
		opts = append(opts, oci.WithUser(containerConfig.User))
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

func (d *Driver) loadContainer(id string) (containerd.Container, error) {
	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return d.client.LoadContainer(ctxWithTimeout, id)
}

func (d *Driver) createTask(container containerd.Container, stdoutPath, stderrPath string) (containerd.Task, error) {
	stdout, stderr, err := getStdoutStderrFifos(stdoutPath, stderrPath)
	if err != nil {
		return nil, err
	}

	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return container.NewTask(ctxWithTimeout, cio.NewCreator(cio.WithStreams(nil, stdout, stderr)))
}

func (d *Driver) getTask(container containerd.Container, stdoutPath, stderrPath string) (containerd.Task, error) {
	stdout, stderr, err := getStdoutStderrFifos(stdoutPath, stderrPath)
	if err != nil {
		return nil, err
	}

	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	return container.Task(ctxWithTimeout, cio.NewAttach(cio.WithStreams(nil, stdout, stderr)))
}
