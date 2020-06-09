package containerd

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/hashicorp/consul-template/signals"
	"github.com/hashicorp/go-hclog"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/drivers/shared/eventer"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	"github.com/hashicorp/nomad/plugins/shared/structs"
	"github.com/moby/moby/pkg/namesgenerator"
)

const (
	// pluginName is the name of the plugin
	// this is used for logging and (along with the version) for uniquely
	// identifying plugin binaries fingerprinted by the client
	pluginName = "containerd-driver"

	// pluginVersion allows the client to identify and use newer versions of
	// an installed plugin
	pluginVersion = "v0.1.0"

	// fingerprintPeriod is the interval at which the plugin will send
	// fingerprint responses
	fingerprintPeriod = 30 * time.Second

	// taskHandleVersion is the version of task handle which this plugin sets
	// and understands how to decode
	// this is used to allow modification and migration of the task schema
	// used by the plugin
	taskHandleVersion = 1
)

var (
	// pluginInfo describes the plugin
	pluginInfo = &base.PluginInfoResponse{
		Type:              base.PluginTypeDriver,
		PluginApiVersions: []string{drivers.ApiVersion010},
		PluginVersion:     pluginVersion,
		Name:              pluginName,
	}

	// configSpec is the specification of the plugin's configuration
	// this is used to validate the configuration specified for the plugin
	// on the client.
	// this is not global, but can be specified on a per-client basis.
	configSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"enabled": hclspec.NewDefault(
			hclspec.NewAttr("enabled", "bool", false),
			hclspec.NewLiteral("true"),
		),
		"containerd_runtime": hclspec.NewAttr("containerd_runtime", "string", true),
	})

	// taskConfigSpec is the specification of the plugin's configuration for
	// a task
	// this is used to validate the configuration specified for the plugin
	// when a job is submitted.
	taskConfigSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"image": hclspec.NewAttr("image", "string", true),
	})

	// capabilities indicates what optional features this driver supports
	// this should be set according to the target run time.
	// https://godoc.org/github.com/hashicorp/nomad/plugins/drivers#Capabilities
	capabilities = &drivers.Capabilities{
		SendSignals: true,
		Exec:        true,
		FSIsolation: drivers.FSIsolationNone,
	}
)

// Config contains configuration information for the plugin
type Config struct {
	Enabled           bool   `codec:"enabled"`
	ContainerdRuntime string `codec:"containerd_runtime"`
}

// TaskConfig contains configuration information for a task that runs with
// this plugin
type TaskConfig struct {
	Image string `codec:"image"`
}

// TaskState is the runtime state which is encoded in the handle returned to
// Nomad client.
// This information is needed to rebuild the task state and handler during
// recovery.
type TaskState struct {
	StartedAt     time.Time
	ContainerName string
}

type Driver struct {
	// eventer is used to handle multiplexing of TaskEvents calls such that an
	// event can be broadcast to all callers
	eventer *eventer.Eventer

	// config is the plugin configuration set by the SetConfig RPC
	config *Config

	// nomadConfig is the client config from Nomad
	nomadConfig *base.ClientDriverConfig

	// tasks is the in memory datastore mapping taskIDs to driver handles
	tasks *taskStore

	// ctx is the context for the driver. It is passed to other subsystems to
	// coordinate shutdown
	ctx context.Context

	// signalShutdown is called when the driver is shutting down and cancels
	// the ctx passed to any subsystems
	signalShutdown context.CancelFunc

	// logger will log to the Nomad agent
	logger log.Logger

	// context for containerd
	ctxContainerd context.Context

	// containerd client
	client *containerd.Client
}

// NewPlugin returns a new containerd driver plugin
func NewPlugin(logger log.Logger) drivers.DriverPlugin {
	ctx, cancel := context.WithCancel(context.Background())
	logger = logger.Named(pluginName)

	// This will create a new containerd client which will talk to
	// default containerd socket path.
	client, err := containerd.New("/run/containerd/containerd.sock")
	if err != nil {
		logger.Error("Error in creating containerd client", "err", err)
		return nil
	}

	// Calls to containerd API are namespaced.
	// "nomad" is the namespace that will be used for all nomad-driver-containerd
	// related containerd API calls.
	ctxContainerd := namespaces.WithNamespace(context.Background(), "nomad")

	return &Driver{
		eventer:        eventer.NewEventer(ctx, logger),
		config:         &Config{},
		tasks:          newTaskStore(),
		ctx:            ctx,
		ctxContainerd:  ctxContainerd,
		client:         client,
		signalShutdown: cancel,
		logger:         logger,
	}
}

// PluginInfo returns information describing the plugin.
func (d *Driver) PluginInfo() (*base.PluginInfoResponse, error) {
	return pluginInfo, nil
}

// ConfigSchema returns the plugin configuration schema.
func (d *Driver) ConfigSchema() (*hclspec.Spec, error) {
	return configSpec, nil
}

// SetConfig is called by the client to pass the configuration for the plugin.
func (d *Driver) SetConfig(cfg *base.Config) error {
	var config Config
	if len(cfg.PluginConfig) != 0 {
		if err := base.MsgPackDecode(cfg.PluginConfig, &config); err != nil {
			return err
		}
	}

	// Save the configuration to the plugin
	d.config = &config

	// Save the Nomad agent configuration
	if cfg.AgentConfig != nil {
		d.nomadConfig = cfg.AgentConfig.Driver
	}

	return nil
}

// TaskConfigSchema returns the HCL schema for the configuration of a task.
func (d *Driver) TaskConfigSchema() (*hclspec.Spec, error) {
	return taskConfigSpec, nil
}

// Capabilities returns the features supported by the driver.
func (d *Driver) Capabilities() (*drivers.Capabilities, error) {
	return capabilities, nil
}

// Fingerprint returns a channel that will be used to send health information
// and other driver specific node attributes.
func (d *Driver) Fingerprint(ctx context.Context) (<-chan *drivers.Fingerprint, error) {
	ch := make(chan *drivers.Fingerprint)
	go d.handleFingerprint(ctx, ch)
	return ch, nil
}

// handleFingerprint manages the channel and the flow of fingerprint data.
func (d *Driver) handleFingerprint(ctx context.Context, ch chan<- *drivers.Fingerprint) {
	defer close(ch)

	// Nomad expects the initial fingerprint to be sent immediately
	ticker := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			// after the initial fingerprint we can set the proper fingerprint
			// period
			ticker.Reset(fingerprintPeriod)
			ch <- d.buildFingerprint()
		}
	}
}

// buildFingerprint returns the driver's fingerprint data
func (d *Driver) buildFingerprint() *drivers.Fingerprint {
	fp := &drivers.Fingerprint{
		Attributes:        map[string]*structs.Attribute{},
		Health:            drivers.HealthStateHealthy,
		HealthDescription: drivers.DriverHealthy,
	}

	isRunning, err := d.isContainerdRunning()
	if err != nil {
		d.logger.Error("Error in buildFingerprint(): failed to get containerd status: %v", err)
		fp.Health = drivers.HealthStateUndetected
		fp.HealthDescription = "Undetected"
		return fp
	}

	if !isRunning {
		fp.Health = drivers.HealthStateUnhealthy
		fp.HealthDescription = "Unhealthy"
		return fp
	}

	// Get containerd version
	version, err := d.getContainerdVersion()
	if err != nil {
		d.logger.Warn("Error in buildFingerprint(): failed to get containerd version: %v", err)
		return fp
	}

	fp.Attributes["driver.containerd.containerd_version"] = structs.NewStringAttribute(version.Version)
	fp.Attributes["driver.containerd.containerd_revision"] = structs.NewStringAttribute(version.Revision)
	return fp
}

// StartTask returns a task handle and a driver network if necessary.
func (d *Driver) StartTask(cfg *drivers.TaskConfig) (*drivers.TaskHandle, *drivers.DriverNetwork, error) {
	if _, ok := d.tasks.Get(cfg.ID); ok {
		return nil, nil, fmt.Errorf("task with ID %q already started", cfg.ID)
	}

	var driverConfig TaskConfig
	if err := cfg.DecodeDriverConfig(&driverConfig); err != nil {
		return nil, nil, fmt.Errorf("failed to decode driver config: %v", err)
	}

	d.logger.Info("starting task", "driver_cfg", hclog.Fmt("%+v", driverConfig))
	handle := drivers.NewTaskHandle(taskHandleVersion)
	handle.Config = cfg

	// Generate a random container name using docker namesgenerator package.
	// https://github.com/moby/moby/blob/master/pkg/namesgenerator/names-generator.go
	containerName := namesgenerator.GetRandomName(1)

	image, err := d.pullImage(driverConfig.Image)
	if err != nil {
		return nil, nil, fmt.Errorf("Error in pulling image: %v", err)
	}

	d.logger.Info(fmt.Sprintf("Successfully pulled %s image\n", image.Name()))

	containerSnapshotName := fmt.Sprintf("%s-snapshot", containerName)
	container, err := d.createContainer(image, containerName, containerSnapshotName, d.config.ContainerdRuntime)
	if err != nil {
		return nil, nil, fmt.Errorf("Error in creating container: %v", err)
	}

	d.logger.Info(fmt.Sprintf("Successfully created container with name: %s", containerName))

	task, err := d.createTask(container)
	if err != nil {
		return nil, nil, fmt.Errorf("Error in creating task: %v", err)
	}

	d.logger.Info(fmt.Sprintf("Successfully created task with ID: %s", task.ID()))

	h := &taskHandle{
		taskConfig:    cfg,
		procState:     drivers.TaskStateRunning,
		startedAt:     time.Now().Round(time.Millisecond),
		logger:        d.logger,
		container:     container,
		containerName: containerName,
		task:          task,
	}

	driverState := TaskState{
		StartedAt:     h.startedAt,
		ContainerName: containerName,
	}

	if err := handle.SetDriverState(&driverState); err != nil {
		return nil, nil, fmt.Errorf("failed to set driver state: %v", err)
	}

	d.tasks.Set(cfg.ID, h)
	go h.run(d.ctxContainerd)
	return handle, nil, nil
}

// RecoverTask recreates the in-memory state of a task from a TaskHandle.
func (d *Driver) RecoverTask(handle *drivers.TaskHandle) error {
	if handle == nil {
		return fmt.Errorf("error: handle cannot be nil")
	}

	if _, ok := d.tasks.Get(handle.Config.ID); ok {
		return nil
	}

	var taskState TaskState
	if err := handle.GetDriverState(&taskState); err != nil {
		return fmt.Errorf("failed to decode task state from handle: %v", err)
	}

	var driverConfig TaskConfig
	if err := handle.Config.DecodeDriverConfig(&driverConfig); err != nil {
		return fmt.Errorf("failed to decode driver config: %v", err)
	}

	container, err := d.loadContainer(taskState.ContainerName)
	if err != nil {
		return fmt.Errorf("Error in recovering container: %v", err)
	}

	task, err := d.getTask(container)
	if err != nil {
		return fmt.Errorf("Error in recovering task: %v", err)
	}

	status, err := task.Status(d.ctxContainerd)
	if err != nil {
		return fmt.Errorf("Error in recovering task status: %v", err)
	}

	h := &taskHandle{
		taskConfig:    handle.Config,
		procState:     drivers.TaskStateRunning,
		startedAt:     taskState.StartedAt,
		exitResult:    &drivers.ExitResult{},
		logger:        d.logger,
		container:     container,
		containerName: taskState.ContainerName,
		task:          task,
	}

	d.tasks.Set(handle.Config.ID, h)

	if status.Status == containerd.Stopped {
		go h.run(d.ctxContainerd)
	}

	d.logger.Info(fmt.Sprintf("Task with ID: %s recovered successfully.", handle.Config.ID))
	return nil
}

// WaitTask returns a channel used to notify Nomad when a task exits.
func (d *Driver) WaitTask(ctx context.Context, taskID string) (<-chan *drivers.ExitResult, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	ch := make(chan *drivers.ExitResult)
	go d.handleWait(ctx, handle, ch)
	return ch, nil
}

func (d *Driver) handleWait(ctx context.Context, handle *taskHandle, ch chan *drivers.ExitResult) {
	defer close(ch)
	var result *drivers.ExitResult

	exitStatusCh, err := handle.task.Wait(d.ctxContainerd)
	if err != nil {
		result = &drivers.ExitResult{
			Err: fmt.Errorf("executor: error waiting on process: %v", err),
		}
	} else {
		status := <-exitStatusCh
		code, _, err := status.Result()
		if err != nil {
			d.logger.Error(err.Error())
			return
		}
		result = &drivers.ExitResult{
			ExitCode: int(code),
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ctx.Done():
			return
		case ch <- result:
		}
	}
}

// StopTask stops a running task with the given signal and within the timeout window.
func (d *Driver) StopTask(taskID string, timeout time.Duration, signal string) error {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	if err := handle.shutdown(d.ctxContainerd, timeout, syscall.SIGTERM); err != nil {
		return fmt.Errorf("Shutdown failed: %v", err)
	}

	return nil
}

// DestroyTask cleans up and removes a task that has terminated.
func (d *Driver) DestroyTask(taskID string, force bool) error {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	isRunning, err := handle.IsRunning(d.ctxContainerd)
	if err != nil {
		return err
	}

	if isRunning && !force {
		return fmt.Errorf("cannot destroy running task")
	}

	if err := handle.cleanup(d.ctxContainerd); err != nil {
		return err
	}

	d.tasks.Delete(taskID)
	return nil
}

// InspectTask returns detailed status information for the referenced taskID.
func (d *Driver) InspectTask(taskID string) (*drivers.TaskStatus, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	return handle.TaskStatus(d.ctxContainerd), nil
}

// TaskStats returns a channel which the driver should send stats to at the given interval.
func (d *Driver) TaskStats(ctx context.Context, taskID string, interval time.Duration) (<-chan *drivers.TaskResourceUsage, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	return handle.stats(ctx, interval)
}

// TaskEvents returns a channel that the plugin can use to emit task related events.
func (d *Driver) TaskEvents(ctx context.Context) (<-chan *drivers.TaskEvent, error) {
	return d.eventer.TaskEvents(ctx)
}

// SignalTask forwards a signal to a task.
// This is an optional capability.
func (d *Driver) SignalTask(taskID string, signal string) error {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return drivers.ErrTaskNotFound
	}

	// The given signal will be forwarded to the target taskID.
	// Please checkout https://github.com/hashicorp/consul-template/blob/master/signals/signals_unix.go
	// for a list of supported signals.
	sig := os.Interrupt
	if s, ok := signals.SignalLookup[signal]; ok {
		sig = s
	} else {
		d.logger.Warn("unknown signal to send to task, using SIGINT instead", "signal", signal, "task_id", handle.taskConfig.ID)

	}
	return handle.signal(d.ctxContainerd, sig)
}

// ExecTaskStreaming returns the result of executing the given command inside a task.
func (d *Driver) ExecTaskStreaming(ctx context.Context, taskID string, opts *drivers.ExecOptions) (*drivers.ExitResult, error) {
	handle, ok := d.tasks.Get(taskID)
	if !ok {
		return nil, drivers.ErrTaskNotFound
	}

	return handle.exec(ctx, d.ctxContainerd, taskID, opts)
}

// ExecTask returns the result of executing the given command inside a task.
// This is an optional capability.
func (d *Driver) ExecTask(taskID string, cmd []string, timeout time.Duration) (*drivers.ExecTaskResult, error) {
	// TODO: implement driver specific logic to execute commands in a task.
	return nil, fmt.Errorf("This driver does not support exec")
}
