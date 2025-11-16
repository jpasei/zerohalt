// Copyright 2025 JPA Solution Experts, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package process

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/jpasei/zerohalt/pkg/health"
	"github.com/jpasei/zerohalt/pkg/metrics"
)

type Config interface {
	GetAppCommand() []string
	GetAppPort() uint16
	GetAdditionalPorts() []uint16
	GetHealthPort() uint16
	GetHealthPath() string
	GetAppStartupTimeout() time.Duration
	GetHealthProbeInterval() time.Duration
	GetShutdownConfig() ShutdownConfig
	GetSignalConfig() SignalConfig
	GetConnectionCheckInterval() interface{}
}

type ShutdownConfig interface {
	GetDrainTimeout() interface{}
	GetShutdownTimeout() interface{}
	GetSignalToApp() string
	GetForceKillAfterTimeout() bool
}

type HealthServer interface {
	Start() error
	SetState(state health.HealthState)
	GetState() health.HealthState
	WaitForAppHealthy(timeout time.Duration, interval time.Duration) bool
}

type ConnectionMonitor interface {
	CountActiveConnections() (int, error)
	WaitForZeroConnections(timeout interface{}) error
}

type ShutdownCoordinator interface {
	InitiateShutdown(sig os.Signal) error
	SetAppProcess(appProcess *os.Process)
}

type Manager struct {
	config        Config
	app           *exec.Cmd
	healthServer  HealthServer
	connMonitor   ConnectionMonitor
	shutdownCoord ShutdownCoordinator
}

func NewManager(config Config) *Manager {
	return &Manager{
		config: config,
	}
}

func (m *Manager) Run(
	healthServer HealthServer,
	connMonitor ConnectionMonitor,
	shutdownCoord ShutdownCoordinator,
) error {
	m.healthServer = healthServer
	m.connMonitor = connMonitor
	m.shutdownCoord = shutdownCoord

	metrics.HealthApp.Set(float64(health.StateStarting))

	if err := m.healthServer.Start(); err != nil {
		return fmt.Errorf("failed to start health server: %w", err)
	}

	slog.Info("Health check server started", "port", m.config.GetHealthPort())

	command := m.config.GetAppCommand()
	if len(command) == 0 {
		return fmt.Errorf("no application command specified")
	}

	m.app = exec.Command(command[0], command[1:]...)
	m.app.Stdout = os.Stdout
	m.app.Stderr = os.Stderr
	m.app.Stdin = os.Stdin

	m.app.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := m.app.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	slog.Info("Application started", "pid", m.app.Process.Pid)

	m.shutdownCoord.SetAppProcess(m.app.Process)

	metrics.HealthApp.Set(float64(health.StateHealthy))

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			metrics.AppUptime.Inc()
		}
	}()

	signalConfig := m.config.GetSignalConfig()
	signalHandler := NewSignalHandler(&signalConfig, m.app.Process)
	sigChan := signalHandler.Setup()
	slog.Info("Signal handler initialized and ready")

	shutdownChan := make(chan error, 1)
	go func() {
		shutdownChan <- m.handleSignals(sigChan, signalHandler)
	}()

	startupTimeout := m.config.GetAppStartupTimeout()
	probeInterval := m.config.GetHealthProbeInterval()

	go m.waitForAppHealthy(startupTimeout, probeInterval)

	return <-shutdownChan
}

func (m *Manager) waitForAppHealthy(startupTimeout time.Duration, probeInterval time.Duration) {
	healthy := m.healthServer.WaitForAppHealthy(startupTimeout, probeInterval)

	healthyState := health.StateUnhealthy
	if healthy {
		healthyState = health.StateHealthy
	}

	m.healthServer.SetState(healthyState)

	if healthy {
		slog.Info("Health check now returning 200 OK")
		return
	}

	slog.Warn("Application did not become healthy within timeout, health endpoint will return 503 unhealthy")
}

func (m *Manager) handleSignals(sigChan chan os.Signal, signalHandler *SignalHandler) error {
	for {
		sig := <-sigChan
		action := signalHandler.Handle(sig)

		switch action {
		case ActionShutdown:
			return m.shutdownCoord.InitiateShutdown(sig)

		case ActionReapZombies:
			m.reapZombies()

		case ActionPassThrough:
			continue

		case ActionIgnore:
			continue
		}
	}
}

func (m *Manager) reapZombies() {
	var wstatus syscall.WaitStatus
	for {
		pid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
		if err != nil || pid <= 0 {
			break
		}
		slog.Debug("Reaped zombie process", "pid", pid)
	}
}
