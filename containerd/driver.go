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
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/hashicorp/consul-template/signals"
	"github.com/hashicorp/go-hclog"
	log "github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/client/lib/cpustats"
	"github.com/hashicorp/nomad/client/taskenv"
	"github.com/hashicorp/nomad/drivers/shared/eventer"
	"github.com/hashicorp/nomad/drivers/shared/resolvconf"
	"github.com/hashicorp/nomad/helper/pluginutils/hclutils"
	"github.com/hashicorp/nomad/plugins/base"
	"github.com/hashicorp/nomad/plugins/drivers"
	"github.com/hashicorp/nomad/plugins/shared/hclspec"
	"github.com/hashicorp/nomad/plugins/shared/structs"
	"github.com/opencontainers/runc/libcontainer/cgroups"
)

const (
	// PluginName is the name of the plugin
	// this is used for logging and (along with the version) for uniquely
	// identifying plugin binaries fingerprinted by the client
	PluginName = "containerd-driver"

	// PluginVersion allows the client to identify and use newer versions of
	// an installed plugin
	PluginVersion = "v0.9.3"

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
		PluginVersion:     PluginVersion,
		Name:              PluginName,
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
		"stats_interval":     hclspec.NewAttr("stats_interval", "string", false),
		"allow_privileged": hclspec.NewDefault(
			hclspec.NewAttr("allow_privileged", "bool", false),
			hclspec.NewLiteral("true"),
		),
		"auth": hclspec.NewBlock("auth", false, hclspec.NewObject(map[string]*hclspec.Spec{
			"username": hclspec.NewAttr("username", "string", true),
			"password": hclspec.NewAttr("password", "string", true),
		})),
	})

	// taskConfigSpec is the specification of the plugin's configuration for
	// a task
	// this is used to validate the configuration specified for the plugin
	// when a job is submitted.
	taskConfigSpec = hclspec.NewObject(map[string]*hclspec.Spec{
		"image":      hclspec.NewAttr("image", "string", true),
		"command":    hclspec.NewAttr("command", "string", false),
		"args":       hclspec.NewAttr("args", "list(string)", false),
		"cap_add":    hclspec.NewAttr("cap_add", "list(string)", false),
		"cap_drop":   hclspec.NewAttr("cap_drop", "list(string)", false),
		"cwd":        hclspec.NewAttr("cwd", "string", false),
		"devices":    hclspec.NewAttr("devices", "list(string)", false),
		"privileged": hclspec.NewAttr("privileged", "bool", false),
		"pids_limit": hclspec.NewAttr("pids_limit", "number", false),
		"pid_mode":   hclspec.NewAttr("pid_mode", "string", false),
		"file_limit": hclspec.NewAttr("file_limit", "number", false),
		"hostname":   hclspec.NewAttr("hostname", "string", false),
		"host_dns": hclspec.NewDefault(
			hclspec.NewAttr("host_dns", "bool", false),
			hclspec.NewLiteral("true"),
		),
		"image_pull_timeout": hclspec.NewDefault(
			hclspec.NewAttr("image_pull_timeout", "string", false),
			hclspec.NewLiteral(`"5m"`),
		),
		"extra_hosts":     hclspec.NewAttr("extra_hosts", "list(string)", false),
		"entrypoint":      hclspec.NewAttr("entrypoint", "list(string)", false),
		"seccomp":         hclspec.NewAttr("seccomp", "bool", false),
		"seccomp_profile": hclspec.NewAttr("seccomp_profile", "string", false),
		"shm_size":        hclspec.NewAttr("shm_size", "string", false),
		"sysctl":          hclspec.NewAttr("sysctl", "list(map(string))", false),
		"readonly_rootfs": hclspec.NewAttr("readonly_rootfs", "bool", false),
		"host_network":    hclspec.NewAttr("host_network", "bool", false),
		"auth": hclspec.NewBlock("auth", false, hclspec.NewObject(map[string]*hclspec.Spec{
			"username": hclspec.NewAttr("username", "string", true),
			"password": hclspec.NewAttr("password", "string", true),
		})),
		"mounts": hclspec.NewBlockList("mounts", hclspec.NewObject(map[string]*hclspec.Spec{
			"type": hclspec.NewDefault(
				hclspec.NewAttr("type", "string", false),
				hclspec.NewLiteral("\"volume\""),
			),
			"target":  hclspec.NewAttr("target", "string", true),
			"source":  hclspec.NewAttr("source", "string", false),
			"options": hclspec.NewAttr("options", "list(string)", false),
		})),
	})

	// capabilities indicates what optional features this driver supports
	// this should be set according to the target run time.
	// https://godoc.org/github.com/hashicorp/nomad/plugins/drivers#Capabilities
	capabilities = &drivers.Capabilities{
		SendSignals:       true,
		Exec:              true,
		FSIsolation:       drivers.FSIsolationImage,
		NetIsolationModes: []drivers.NetIsolationMode{drivers.NetIsolationModeGroup, drivers.NetIsolationModeTask},
	}
)

// Config contains configuration information for the plugin
type Config struct {
	Enabled           bool         `codec:"enabled"`
	ContainerdRuntime string       `codec:"containerd_runtime"`
	StatsInterval     string       `codec:"stats_interval"`
	AllowPrivileged   bool         `codec:"allow_privileged"`
	Auth              RegistryAuth `codec:"auth"`
}

// Volume, bind, and tmpfs type mounts are supported.
// Mount contains configuration information about a mountpoint.
type Mount struct {
	Type    string   `codec:"type"`
	Target  string   `codec:"target"`
	Source  string   `codec:"source"`
	Options []string `codec:"options"`
}

// Auth info to pull image from registry.
type RegistryAuth struct {
	Username string `codec:"username"`
	Password string `codec:"password"`
}

// TaskConfig contains configuration information for a task that runs with
// this plugin
type TaskConfig struct {
	Image            string             `codec:"image"`
	Command          string             `codec:"command"`
	Args             []string           `codec:"args"`
	CapAdd           []string           `codec:"cap_add"`
	CapDrop          []string           `codec:"cap_drop"`
	Cwd              string             `codec:"cwd"`
	Devices          []string           `codec:"devices"`
	Seccomp          bool               `codec:"seccomp"`
	SeccompProfile   string             `codec:"seccomp_profile"`
	ShmSize          string             `codec:"shm_size"`
	Sysctl           hclutils.MapStrStr `codec:"sysctl"`
	Privileged       bool               `codec:"privileged"`
	PidsLimit        int64              `codec:"pids_limit"`
	PidMode          string             `codec:"pid_mode"`
	FileLimit        int64              `codec:"file_limit"`
	Hostname         string             `codec:"hostname"`
	HostDNS          bool               `codec:"host_dns"`
	ImagePullTimeout string             `codec:"image_pull_timeout"`
	ExtraHosts       []string           `codec:"extra_hosts"`
	Entrypoint       []string           `codec:"entrypoint"`
	ReadOnlyRootfs   bool               `codec:"readonly_rootfs"`
	HostNetwork      bool               `codec:"host_network"`
	Auth             RegistryAuth       `codec:"auth"`
	Mounts           []Mount            `codec:"mounts"`
}

// TaskState is the runtime state which is encoded in the handle returned to
// Nomad client.
// This information is needed to rebuild the task state and handler during
// recovery.
type TaskState struct {
	StartedAt     time.Time
	ContainerName string
	StdoutPath    string
	StderrPath    string
}

type Driver struct {
	// eventer is used to handle multiplexing of TaskEvents calls such that an
	// event can be broadcast to all callers
	eventer *eventer.Eventer

	// config is the plugin configuration set by the SetConfig RPC
	config *Config

	// nomadConfig is the client config from Nomad
	nomadConfig *base.ClientDriverConfig

	// compute contains information about the available cpu compute
	compute cpustats.Compute

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
	logger = logger.Named(PluginName)

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
	namespace := "nomad"
	// Unless we are operating in cgroups.v2 mode, in which case we use the
	// name "nomad.slice", which ends up being the cgroup parent.
	if cgroups.IsCgroup2UnifiedMode() {
		namespace = "nomad.slice"
	}
	ctxContainerd := namespaces.WithNamespace(context.Background(), namespace)

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

func (tc *TaskConfig) setVolumeMounts(cfg *drivers.TaskConfig) error {
	for _, m := range cfg.Mounts {
		hm := Mount{
			Type:    "bind",
			Target:  m.TaskPath,
			Source:  m.HostPath,
			Options: []string{"rbind"},
		}
		if m.Readonly {
			hm.Options = append(hm.Options, "ro")
		}

		tc.Mounts = append(tc.Mounts, hm)
	}

	if cfg.DNS != nil {
		dnsMount, err := resolvconf.GenerateDNSMount(cfg.TaskDir().Dir, cfg.DNS)
		if err != nil {
			return fmt.Errorf("failed to build mount for resolv.conf: %v", err)
		}
		tc.HostDNS = false
		tc.Mounts = append(tc.Mounts, Mount{
			Type:    "bind",
			Target:  dnsMount.TaskPath,
			Source:  dnsMount.HostPath,
			Options: []string{"bind", "ro"},
		})
	}
	return nil
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
		d.compute = cfg.AgentConfig.Compute()
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
	if err != nil || !isRunning {
		if err != nil {
			d.logger.Error("Error in buildFingerprint(): failed to get containerd status", "error", err)
		}
		fp.Health = drivers.HealthStateUnhealthy
		fp.HealthDescription = "Unhealthy"
		return fp
	}

	// Get containerd version
	version, err := d.getContainerdVersion()
	if err != nil {
		d.logger.Warn("Error in buildFingerprint(): failed to get containerd version:", "error", err)
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

	containerConfig := ContainerConfig{}

	if driverConfig.HostNetwork && cfg.NetworkIsolation != nil {
		return nil, nil, fmt.Errorf("host_network and bridge network mode are mutually exclusive, and only one of them should be set")
	}

	if err := driverConfig.setVolumeMounts(cfg); err != nil {
		return nil, nil, err
	}

	d.logger.Info("starting task", "driver_cfg", hclog.Fmt("%+v", driverConfig))
	handle := drivers.NewTaskHandle(taskHandleVersion)
	handle.Config = cfg

	// Use Nomad's docker naming convention for the container name
	// https://www.nomadproject.io/docs/drivers/docker#container-name
	containerName := cfg.Name + "-" + cfg.AllocID
	if cgroups.IsCgroup2UnifiedMode() {
		// In cgroup.v2 mode, the name is slightly different.
		containerName = fmt.Sprintf("%s.%s.scope", cfg.AllocID, cfg.Name)
	}
	containerConfig.ContainerName = containerName

	var err error
	containerConfig.Image, err = d.pullImage(driverConfig.Image, driverConfig.ImagePullTimeout, &driverConfig.Auth)
	if err != nil {
		return nil, nil, fmt.Errorf("Error in pulling image %s: %v", driverConfig.Image, err)
	}

	d.logger.Info(fmt.Sprintf("Successfully pulled %s image\n", containerConfig.Image.Name()))

	// Setup environment variables.
	for key, val := range cfg.Env {
		if skipOverride(key) {
			continue
		}
		containerConfig.Env = append(containerConfig.Env, fmt.Sprintf("%s=%s", key, val))
	}

	// Setup source paths for secrets, task and alloc directories.
	containerConfig.SecretsDirSrc = cfg.TaskDir().SecretsDir
	containerConfig.TaskDirSrc = cfg.TaskDir().LocalDir
	containerConfig.AllocDirSrc = cfg.TaskDir().SharedAllocDir

	// Setup destination paths for secrets, task and alloc directories.
	containerConfig.SecretsDirDest = cfg.Env[taskenv.SecretsDir]
	containerConfig.TaskDirDest = cfg.Env[taskenv.TaskLocalDir]
	containerConfig.AllocDirDest = cfg.Env[taskenv.AllocDir]

	containerConfig.ContainerSnapshotName = fmt.Sprintf("%s-snapshot", containerName)
	if cfg.NetworkIsolation != nil && cfg.NetworkIsolation.Path != "" {
		containerConfig.NetworkNamespacePath = cfg.NetworkIsolation.Path
	}

	// memory and cpu are coming from the resources stanza of the nomad job.
	// https://www.nomadproject.io/docs/job-specification/resources
	containerConfig.MemoryLimit = cfg.Resources.NomadResources.Memory.MemoryMB * 1024 * 1024
	containerConfig.MemoryHardLimit = cfg.Resources.NomadResources.Memory.MemoryMaxMB * 1024 * 1024
	containerConfig.CPUShares = cfg.Resources.LinuxResources.CPUShares

	containerConfig.User = cfg.User

	container, err := d.createContainer(&containerConfig, &driverConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("Error in creating container: %v", err)
	}

	d.logger.Info(fmt.Sprintf("Successfully created container with name: %s\n", containerName))
	task, err := d.createTask(container, cfg.StdoutPath, cfg.StderrPath)
	if err != nil {
		return nil, nil, fmt.Errorf("Error in creating task: %v", err)
	}

	d.logger.Info(fmt.Sprintf("Successfully created task with ID: %s\n", task.ID()))

	h := &taskHandle{
		taskConfig:     cfg,
		procState:      drivers.TaskStateRunning,
		startedAt:      time.Now().Round(time.Millisecond),
		logger:         d.logger,
		totalCpuStats:  cpustats.New(d.compute),
		userCpuStats:   cpustats.New(d.compute),
		systemCpuStats: cpustats.New(d.compute),
		container:      container,
		containerName:  containerName,
		task:           task,
	}

	driverState := TaskState{
		StartedAt:     h.startedAt,
		ContainerName: containerName,
		StdoutPath:    cfg.StdoutPath,
		StderrPath:    cfg.StderrPath,
	}

	if err := handle.SetDriverState(&driverState); err != nil {
		return nil, nil, fmt.Errorf("failed to set driver state: %v", err)
	}

	d.tasks.Set(cfg.ID, h)

	go h.run(d.ctxContainerd)
	return handle, nil, nil
}

// skipOverride determines whether the environment variable (key) needs an override or not.
func skipOverride(key string) bool {
	skipOverrideList := []string{"PATH"}
	for _, k := range skipOverrideList {
		if key == k {
			return true
		}
	}
	return false
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

	container, err := d.loadContainer(taskState.ContainerName)
	if err != nil {
		return fmt.Errorf("Error in recovering container: %v", err)
	}

	task, err := d.getTask(container, taskState.StdoutPath, taskState.StderrPath)
	if err != nil {
		return fmt.Errorf("Error in recovering task: %v", err)
	}

	ctxWithTimeout, cancel := context.WithTimeout(d.ctxContainerd, 30*time.Second)
	defer cancel()

	status, err := task.Status(ctxWithTimeout)
	if err != nil {
		return fmt.Errorf("Error in recovering task status: %v", err)
	}

	h := &taskHandle{
		taskConfig:     handle.Config,
		procState:      drivers.TaskStateRunning,
		startedAt:      taskState.StartedAt,
		exitResult:     &drivers.ExitResult{},
		logger:         d.logger,
		totalCpuStats:  cpustats.New(d.compute),
		userCpuStats:   cpustats.New(d.compute),
		systemCpuStats: cpustats.New(d.compute),
		container:      container,
		containerName:  taskState.ContainerName,
		task:           task,
	}

	d.tasks.Set(handle.Config.ID, h)

	if status.Status == containerd.Stopped {
		go h.run(d.ctxContainerd)
	}

	d.logger.Info(fmt.Sprintf("Task with ID: %s recovered successfully.\n", handle.Config.ID))
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
			ExitCode: 255,
			Err:      fmt.Errorf("executor: error waiting on process: %v", err),
		}
	} else {
		status := <-exitStatusCh
		code, _, err := status.Result()
		result = &drivers.ExitResult{
			ExitCode: int(code),
			Err:      err,
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

	if d.config.StatsInterval != "" {
		statsInterval, err := time.ParseDuration(d.config.StatsInterval)
		if err != nil {
			d.logger.Warn("Error parsing driver stats interval, fallback on default interval")
		} else {
			msg := fmt.Sprintf("Overriding client stats interval: %v with driver stats interval: %v\n", interval, d.config.StatsInterval)
			d.logger.Debug(msg)
			interval = statsInterval
		}
	}

	return handle.stats(ctx, d.ctxContainerd, interval)
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
	sig, ok := signals.SignalLookup[signal]
	if !ok {
		return fmt.Errorf("Invalid signal: %s", signal)
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
