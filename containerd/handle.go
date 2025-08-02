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
	"sync"
	"syscall"
	"time"

	v1 "github.com/containerd/cgroups/v3/cgroup1/stats"
	v2 "github.com/containerd/cgroups/v3/cgroup2/stats"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/typeurl/v2"
	"github.com/hashicorp/go-hclog"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/nomad/client/lib/cpustats"
	"github.com/hashicorp/nomad/plugins/drivers"
)

// taskHandle should store all relevant runtime information
// such as process ID if this is a local task or other meta
// data if this driver deals with external APIs
type taskHandle struct {
	// stateLock syncs access to all fields below
	stateLock sync.RWMutex

	logger         hclog.Logger
	taskConfig     *drivers.TaskConfig
	procState      drivers.TaskState
	startedAt      time.Time
	completedAt    time.Time
	exitResult     *drivers.ExitResult
	totalCpuStats  *cpustats.Tracker
	userCpuStats   *cpustats.Tracker
	systemCpuStats *cpustats.Tracker
	containerName  string
	container      containerd.Container
	task           containerd.Task
}

func (h *taskHandle) TaskStatus(ctxContainerd context.Context) *drivers.TaskStatus {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()

	h.procState = drivers.TaskStateExited

	isRunning, err := h.IsRunning(ctxContainerd)
	if err != nil {
		h.procState = drivers.TaskStateUnknown
	} else if isRunning {
		h.procState = drivers.TaskStateRunning
	}

	return &drivers.TaskStatus{
		ID:          h.taskConfig.ID,
		Name:        h.taskConfig.Name,
		State:       h.procState,
		StartedAt:   h.startedAt,
		CompletedAt: h.completedAt,
		ExitResult:  h.exitResult,
		DriverAttributes: map[string]string{
			"containerName": h.containerName,
		},
	}
}

func (h *taskHandle) IsRunning(ctxContainerd context.Context) (bool, error) {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()

	ctxWithTimeout, cancel := context.WithTimeout(ctxContainerd, 30*time.Second)
	defer cancel()

	status, err := h.task.Status(ctxWithTimeout)
	if err != nil {
		return false, fmt.Errorf("error in getting task status: %v", err)
	}

	return (status.Status == containerd.Running), nil
}

func (h *taskHandle) run(ctxContainerd context.Context) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	// Sleep for 5 seconds to allow h.task.Wait() to kick in.
	// TODO: Use goroutine and a channel to synchronize this, instead of sleep.
	time.Sleep(5 * time.Second)

	h.task.Start(ctxContainerd)
}

// exec launches a new process in a running container.
func (h *taskHandle) exec(ctx, ctxContainerd context.Context, taskID string, opts *drivers.ExecOptions) (*drivers.ExitResult, error) {
	defer opts.Stdout.Close()
	defer opts.Stderr.Close()

	spec, err := h.container.Spec(ctxContainerd)
	if err != nil {
		return nil, err
	}

	pspec := spec.Process
	pspec.Terminal = opts.Tty
	pspec.Args = opts.Command
	execID, err := uuid.GenerateUUID()
	if err != nil {
		return nil, err
	}

	cioOpts := []cio.Opt{cio.WithStreams(opts.Stdin, opts.Stdout, opts.Stderr)}
	if opts.Tty {
		cioOpts = append(cioOpts, cio.WithTerminal)
	}
	ioCreator := cio.NewCreator(cioOpts...)

	process, err := h.task.Exec(ctxContainerd, execID[:8], pspec, ioCreator)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			select {
			case s, ok := <-opts.ResizeCh:
				if !ok {
					return
				}
				if err = h.task.Resize(ctxContainerd, uint32(s.Width), uint32(s.Height)); err != nil {
					h.logger.Error("Failed to resize terminal", "error", err)
					return
				}
			}
		}
	}()

	defer process.Delete(ctxContainerd)

	statusC, err := process.Wait(ctxContainerd)
	if err != nil {
		return nil, err
	}

	if err := process.Start(ctxContainerd); err != nil {
		return nil, err
	}

	var code uint32
	status := <-statusC
	code, _, err = status.Result()
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			}
		}
	}()

	return &drivers.ExitResult{
		ExitCode: int(code),
	}, nil

}

func (h *taskHandle) shutdown(ctxContainerd context.Context, timeout time.Duration, signal syscall.Signal) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctxContainerd, 30*time.Second)
	defer cancel()

	if err := h.task.Kill(ctxWithTimeout, signal); err != nil {
		return err
	}

	// timeout = 5 seconds, passed by nomad client
	// TODO: Make timeout configurable in task_config. This will allow users to set a higher timeout
	// if they need more time for their container to shutdown gracefully.
	time.Sleep(timeout)

	status, err := h.task.Status(ctxWithTimeout)
	if err != nil {
		return err
	}

	if status.Status != containerd.Running {
		h.logger.Info("Task is not running anymore, no need to SIGKILL")
		return nil
	}

	return h.task.Kill(ctxWithTimeout, syscall.SIGKILL)
}

func (h *taskHandle) cleanup(ctxContainerd context.Context) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctxContainerd, 30*time.Second)
	defer cancel()

	if _, err := h.task.Delete(ctxWithTimeout); err != nil {
		return err
	}
	if err := h.container.Delete(ctxWithTimeout, containerd.WithSnapshotCleanup); err != nil {
		return err
	}
	return nil
}

func (h *taskHandle) stats(ctx, ctxContainerd context.Context, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	ch := make(chan *drivers.TaskResourceUsage)

	go h.handleStats(ch, ctx, ctxContainerd, interval)

	return ch, nil
}

func (h *taskHandle) handleStats(ch chan *drivers.TaskResourceUsage, ctx, ctxContainerd context.Context, interval time.Duration) {
	defer close(ch)

	timer := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			timer.Reset(interval)
		}

		// Get containerd task metric
		metric, err := h.task.Metrics(ctxContainerd)
		if err != nil {
			h.logger.Error("Failed to get task metric:", "error", err)
			return
		}

		anydata, err := typeurl.UnmarshalAny(metric.Data)
		if err != nil {
			h.logger.Error("Failed to unmarshal metric data:", "error", err)
			return
		}

		var taskResourceUsage *drivers.TaskResourceUsage

		switch data := anydata.(type) {
		case *v1.Metrics:
			taskResourceUsage = h.getV1TaskResourceUsage(data)
		case *v2.Metrics:
			taskResourceUsage = h.getV2TaskResourceUsage(data)
		default:
			h.logger.Error("Cannot convert metric data to cgroups.Metrics")
			return
		}

		select {
		case <-ctx.Done():
			return
		case ch <- taskResourceUsage:
		}
	}
}

// Convert containerd V1 task metrics to TaskResourceUsage.
func (h *taskHandle) getV1TaskResourceUsage(metrics *v1.Metrics) *drivers.TaskResourceUsage {
	totalPercent := h.totalCpuStats.Percent(float64(metrics.CPU.Usage.Total))
	cs := &drivers.CpuStats{
		SystemMode: h.systemCpuStats.Percent(float64(metrics.CPU.Usage.Kernel)),
		UserMode:   h.userCpuStats.Percent(float64(metrics.CPU.Usage.User)),
		Percent:    totalPercent,
		TotalTicks: h.totalCpuStats.TicksConsumed(totalPercent),
		Measured:   []string{"Percent", "System Mode", "User Mode"},
	}

	ms := &drivers.MemoryStats{
		RSS:      metrics.Memory.RSS,
		Cache:    metrics.Memory.Cache,
		Swap:     metrics.Memory.Swap.Usage,
		Usage:    metrics.Memory.Usage.Usage,
		MaxUsage: metrics.Memory.Usage.Max,
		Measured: []string{"RSS", "Cache", "Swap", "Usage"},
	}

	ts := time.Now().UTC().UnixNano()
	return &drivers.TaskResourceUsage{
		ResourceUsage: &drivers.ResourceUsage{
			CpuStats:    cs,
			MemoryStats: ms,
		},
		Timestamp: ts,
	}
}

// Convert containerd V2 task metrics to TaskResourceUsage.
func (h *taskHandle) getV2TaskResourceUsage(metrics *v2.Metrics) *drivers.TaskResourceUsage {
	totalPercent := h.totalCpuStats.Percent(float64(metrics.CPU.SystemUsec + metrics.CPU.UserUsec))
	cs := &drivers.CpuStats{
		SystemMode: h.systemCpuStats.Percent(float64(metrics.CPU.SystemUsec)),
		UserMode:   h.userCpuStats.Percent(float64(metrics.CPU.UserUsec)),
		Percent:    totalPercent,
		TotalTicks: h.totalCpuStats.TicksConsumed(totalPercent),
		Measured:   []string{"Percent", "System Mode", "User Mode"},
	}

	ms := &drivers.MemoryStats{
		Swap:     metrics.Memory.SwapUsage,
		Usage:    metrics.Memory.Usage,
		Measured: []string{"Swap", "Usage"},
	}

	ts := time.Now().UTC().UnixNano()
	return &drivers.TaskResourceUsage{
		ResourceUsage: &drivers.ResourceUsage{
			CpuStats:    cs,
			MemoryStats: ms,
		},
		Timestamp: ts,
	}
}
func (h *taskHandle) signal(ctxContainerd context.Context, sig os.Signal) error {
	ctxWithTimeout, cancel := context.WithTimeout(ctxContainerd, 30*time.Second)
	defer cancel()

	return h.task.Kill(ctxWithTimeout, sig.(syscall.Signal))
}
