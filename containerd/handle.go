package containerd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/plugins/drivers"
)

// taskHandle should store all relevant runtime information
// such as process ID if this is a local task or other meta
// data if this driver deals with external APIs
type taskHandle struct {
	// stateLock syncs access to all fields below
	stateLock sync.RWMutex

	logger        hclog.Logger
	taskConfig    *drivers.TaskConfig
	procState     drivers.TaskState
	startedAt     time.Time
	completedAt   time.Time
	exitResult    *drivers.ExitResult
	containerName string
	container     containerd.Container
	task          containerd.Task
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

	status, err := h.task.Status(ctxContainerd)
	if err != nil {
		return false, fmt.Errorf("Error in getting task status: %v", err)
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
	execID := getRandomID(8)

	cioOpts := []cio.Opt{cio.WithStreams(opts.Stdin, opts.Stdout, opts.Stderr)}
	if opts.Tty {
		cioOpts = append(cioOpts, cio.WithTerminal)
	}
	ioCreator := cio.NewCreator(cioOpts...)

	process, err := h.task.Exec(ctxContainerd, execID, pspec, ioCreator)
	if err != nil {
		return nil, err
	}

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
	if err := h.task.Kill(ctxContainerd, signal); err != nil {
		return err
	}

	// timeout = 5 seconds, passed by nomad client
	// TODO: Make timeout configurable in task_config. This will allow users to set a higher timeout
	// if they need more time for their container to shutdown gracefully.
	time.Sleep(timeout)

	status, err := h.task.Status(ctxContainerd)
	if err != nil {
		return err
	}

	if status.Status != containerd.Running {
		h.logger.Info("Task is not running anymore, no need to SIGKILL")
		return nil
	}

	return h.task.Kill(ctxContainerd, syscall.SIGKILL)
}

func (h *taskHandle) cleanup(ctxContainerd context.Context) error {
	if _, err := h.task.Delete(ctxContainerd); err != nil {
		return err
	}
	if err := h.container.Delete(ctxContainerd, containerd.WithSnapshotCleanup); err != nil {
		return err
	}
	return nil
}

func (h *taskHandle) stats(ctx context.Context, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	return nil, nil
}

func (h *taskHandle) signal(ctxContainerd context.Context, sig os.Signal) error {
	return h.task.Kill(ctxContainerd, sig.(syscall.Signal))
}
