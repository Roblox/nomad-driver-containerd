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
	"os"
	"strings"
	"syscall"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/plugin"
	runcoptions "github.com/containerd/containerd/runtime/v2/runc/options"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// buildMountpoint builds the mount point for the container.
func buildMountpoint(mountType, mountTarget, mountSource string, mountOptions []string) specs.Mount {
	m := specs.Mount{}
	m.Type = mountType
	m.Destination = mountTarget
	m.Source = mountSource
	m.Options = mountOptions
	return m
}

// getStdoutStderrFifos return the container's stdout and stderr FIFO's.
func getStdoutStderrFifos(stdoutPath, stderrPath string) (*os.File, *os.File, error) {
	stdout, err := openFIFO(stdoutPath)
	if err != nil {
		return nil, nil, err
	}

	stderr, err := openFIFO(stderrPath)
	if err != nil {
		return nil, nil, err
	}
	return stdout, stderr, nil
}

// FIFO's are named pipes in linux.
// openFIFO() opens the nomad task stdout/stderr pipes and returns the fd.
func openFIFO(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
}

// WithSysctls sets the provided sysctls onto the spec
// Original code referenced from:
// https://github.com/containerd/containerd/blob/master/pkg/cri/opts/spec_linux.go#L546-L560
func WithSysctls(sysctls map[string]string) oci.SpecOpts {
	return func(ctx context.Context, client oci.Client, c *containers.Container, s *specs.Spec) error {
		if s.Linux == nil {
			s.Linux = &specs.Linux{}
		}
		if s.Linux.Sysctl == nil {
			s.Linux.Sysctl = make(map[string]string)
		}
		for k, v := range sysctls {
			s.Linux.Sysctl[k] = v
		}
		return nil
	}
}

// WithMemoryLimits accepts soft (`memory`) and hard (`memory_max`) limits as parameters and set the desired
// limits. With `Nomad<1.1.0` releases, soft (`memory`) will act as a hard limit, and if the container process exceeds
// that limit, it will be OOM'ed. With `Nomad>=1.1.0` releases, users can over-provision using `soft` and `hard`
// limits.  The container process will only get OOM'ed if the hard limit is exceeded.
func WithMemoryLimits(soft, hard int64) oci.SpecOpts {
	return func(_ context.Context, _ oci.Client, _ *containers.Container, s *oci.Spec) error {
		if s.Linux != nil {
			if s.Linux.Resources == nil {
				s.Linux.Resources = &specs.LinuxResources{}
			}
			if s.Linux.Resources.Memory == nil {
				s.Linux.Resources.Memory = &specs.LinuxMemory{}
			}

			if hard > 0 {
				s.Linux.Resources.Memory.Limit = &hard
				s.Linux.Resources.Memory.Reservation = &soft
			} else {
				s.Linux.Resources.Memory.Limit = &soft
			}
		}
		return nil
	}
}

// buildRuntime sets the container runtime e.g. runc or runsc (gVisor).
func buildRuntime(pluginRuntime, jobRuntime string) containerd.NewContainerOpts {
	var (
		runcOpts    runcoptions.Options
		runtimeOpts interface{} = &runcOpts
	)

	// plugin.RuntimeRuncV2 = io.containerd.runc.v2
	runtime := plugin.RuntimeRuncV2

	if jobRuntime != "" {
		if strings.HasPrefix(jobRuntime, "io.containerd.runc.") {
			runtime = jobRuntime
		} else {
			runcOpts.BinaryName = jobRuntime
		}
	} else if pluginRuntime != "" {
		if strings.HasPrefix(pluginRuntime, "io.containerd.runc.") {
			runtime = pluginRuntime
		} else {
			runcOpts.BinaryName = pluginRuntime
		}
	}

	return containerd.WithRuntime(runtime, runtimeOpts)
}
