package containerd

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/containerd/containerd"
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

func (h *taskHandle) TaskStatus() *drivers.TaskStatus {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()

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

func (h *taskHandle) IsRunning() bool {
	h.stateLock.RLock()
	defer h.stateLock.RUnlock()
	return h.procState == drivers.TaskStateRunning
}

func (h *taskHandle) run(ctxContainerd context.Context) {
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	// Sleep for 5 seconds to allow h.task.Wait() to kick in.
	time.Sleep(5 * time.Second)

	h.task.Start(ctxContainerd)
}

func (h *taskHandle) shutdown(ctxContainerd context.Context, signal syscall.Signal) error {
	return h.task.Kill(ctxContainerd, signal)
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

func (h *taskHandle) signal(sig os.Signal) error {
	return nil
}
