package containerd

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

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

func (h *taskHandle) run() {
	h.stateLock.Lock()
	if h.exitResult == nil {
		h.exitResult = &drivers.ExitResult{}
	}
	h.stateLock.Unlock()

	// TODO: wait for your task to complete and upate its state.
	//ps, err := h.exec.Wait(context.Background())
	h.stateLock.Lock()
	defer h.stateLock.Unlock()

	err := fmt.Errorf("Hello test error")

	if err != nil {
		h.exitResult.Err = err
		h.procState = drivers.TaskStateUnknown
		h.completedAt = time.Now()
		return
	}
	h.procState = drivers.TaskStateExited
	//h.exitResult.ExitCode = ps.ExitCode
	//h.exitResult.Signal = ps.Signal
	//h.completedAt = ps.Time
}

func (h *taskHandle) shutdown(timeout time.Duration, signal string) error {
	return nil
}

func (h *taskHandle) cleanup() error {
	return nil
}

func (h *taskHandle) stats(ctx context.Context, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	return nil, nil
}

func (h *taskHandle) signal(sig os.Signal) error {
	return nil
}
