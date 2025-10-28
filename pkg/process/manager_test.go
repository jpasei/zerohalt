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
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockConfig struct {
	command []string
	port    uint16
}

func (m *mockConfig) GetAppCommand() []string {
	return m.command
}

func (m *mockConfig) GetAppPort() uint16 {
	return m.port
}

func (m *mockConfig) GetAdditionalPorts() []uint16 {
	return []uint16{}
}

func (m *mockConfig) GetHealthPort() uint16 {
	return getAvailablePort()
}

func (m *mockConfig) GetHealthPath() string {
	return "/health"
}

func (m *mockConfig) GetShutdownConfig() ShutdownConfig {
	return &mockShutdownConfig{}
}

func (m *mockConfig) GetSignalConfig() SignalConfig {
	return SignalConfig{
		ShutdownSignals: []string{"SIGTERM"},
	}
}

func (m *mockConfig) GetConnectionCheckInterval() interface{} {
	return 1 * time.Second
}

type mockShutdownConfig struct{}

func (m *mockShutdownConfig) GetDrainTimeout() interface{} {
	return 60 * time.Second
}

func (m *mockShutdownConfig) GetShutdownTimeout() interface{} {
	return 30 * time.Second
}

func (m *mockShutdownConfig) GetSignalToApp() string {
	return "SIGTERM"
}

func (m *mockShutdownConfig) GetForceKillAfterTimeout() bool {
	return true
}

type mockHealthServer struct {
	started bool
	state   int
}

func (m *mockHealthServer) Start() error {
	m.started = true
	return nil
}

func (m *mockHealthServer) SetState(state int) {
	m.state = state
}

func (m *mockHealthServer) GetState() int {
	return m.state
}

type mockConnectionMonitor struct{}

func (m *mockConnectionMonitor) CountActiveConnections() (int, error) {
	return 0, nil
}

func (m *mockConnectionMonitor) WaitForZeroConnections(timeout interface{}) error {
	return nil
}

type mockShutdownCoordinator struct {
	process *os.Process
}

func (m *mockShutdownCoordinator) InitiateShutdown(sig os.Signal) error {
	return nil
}

func (m *mockShutdownCoordinator) SetAppProcess(appProcess *os.Process) {
	m.process = appProcess
}

func TestNewManager(t *testing.T) {
	cfg := &mockConfig{}
	manager := NewManager(cfg)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.config)
}

func TestManager_Run_NoCommand(t *testing.T) {
	cfg := &mockConfig{
		command: []string{},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	err := manager.Run(healthServer, connMonitor, shutdownCoord)

	assert.Error(t, err)
}

func TestManager_Run_HealthServerStart(t *testing.T) {
	cfg := &mockConfig{
		command: []string{"sleep", "100"},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(healthServer, connMonitor, shutdownCoord)
	}()

	time.Sleep(200 * time.Millisecond)

	assert.True(t, healthServer.started)

	manager.app.Process.Kill()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

type mockHealthServerWithError struct {
	started bool
	state   int
}

func (m *mockHealthServerWithError) Start() error {
	return os.ErrPermission
}

func (m *mockHealthServerWithError) SetState(state int) {
	m.state = state
}

func (m *mockHealthServerWithError) GetState() int {
	return m.state
}

func TestManager_Run_HealthServerStartError(t *testing.T) {
	cfg := &mockConfig{
		command: []string{"sleep", "100"},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServerWithError{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	err := manager.Run(healthServer, connMonitor, shutdownCoord)

	assert.Error(t, err)
}

func TestManager_Run_InvalidCommand(t *testing.T) {
	cfg := &mockConfig{
		command: []string{"/nonexistent/command"},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	err := manager.Run(healthServer, connMonitor, shutdownCoord)

	assert.Error(t, err)
}

func TestManager_Run_ShutdownSignal(t *testing.T) {
	cfg := &mockConfig{
		command: []string{"sleep", "10"},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(healthServer, connMonitor, shutdownCoord)
	}()

	time.Sleep(200 * time.Millisecond)

	manager.app.Process.Signal(syscall.SIGTERM)

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		manager.app.Process.Kill()
	}
}

type mockConfigWithSignals struct {
	mockConfig
	passThroughSignals []string
}

func (m *mockConfigWithSignals) GetSignalConfig() SignalConfig {
	return SignalConfig{
		ShutdownSignals:    []string{"SIGTERM"},
		PassThroughSignals: m.passThroughSignals,
	}
}

func TestManager_Run_MultipleSignals(t *testing.T) {
	cfg := &mockConfigWithSignals{
		mockConfig: mockConfig{
			command: []string{"sleep", "10"},
		},
		passThroughSignals: []string{"SIGHUP"},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(healthServer, connMonitor, shutdownCoord)
	}()

	time.Sleep(500 * time.Millisecond)

	currentProc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)

	currentProc.Signal(syscall.SIGUSR1)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGUSR2)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGHUP)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGTERM)

	select {
	case err := <-done:
		assert.NoError(t, err)
		manager.app.Process.Kill()
	case <-time.After(3 * time.Second):
		manager.app.Process.Kill()
		t.Fatal("Test timed out")
	}
}

func TestManager_Run_ActionPassThrough_ContinuesLoop(t *testing.T) {
	cfg := &mockConfigWithSignals{
		mockConfig: mockConfig{
			command: []string{"sleep", "5"},
		},
		passThroughSignals: []string{"SIGHUP", "SIGUSR1"},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(healthServer, connMonitor, shutdownCoord)
	}()

	time.Sleep(300 * time.Millisecond)

	currentProc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)

	currentProc.Signal(syscall.SIGHUP)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGUSR1)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGTERM)

	select {
	case err := <-done:
		assert.NoError(t, err)
		if manager.app.Process != nil {
			manager.app.Process.Kill()
		}
	case <-time.After(3 * time.Second):
		if manager.app.Process != nil {
			manager.app.Process.Kill()
		}
		t.Fatal("Test timed out")
	}
}

func TestManager_Run_ActionIgnore_ContinuesLoop(t *testing.T) {
	cfg := &mockConfigWithSignals{
		mockConfig: mockConfig{
			command: []string{"sleep", "5"},
		},
		passThroughSignals: []string{},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(healthServer, connMonitor, shutdownCoord)
	}()

	time.Sleep(300 * time.Millisecond)

	currentProc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)

	currentProc.Signal(syscall.SIGUSR1)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGUSR2)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGTERM)

	select {
	case err := <-done:
		assert.NoError(t, err)
		if manager.app.Process != nil {
			manager.app.Process.Kill()
		}
	case <-time.After(3 * time.Second):
		if manager.app.Process != nil {
			manager.app.Process.Kill()
		}
		t.Fatal("Test timed out")
	}
}

func TestManager_Run_PassThroughAndIgnore_ContinuesLoop(t *testing.T) {
	cfg := &mockConfigWithSignals{
		mockConfig: mockConfig{
			command: []string{"sleep", "5"},
		},
		passThroughSignals: []string{"SIGHUP"},
	}

	manager := NewManager(cfg)
	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	shutdownCoord := &mockShutdownCoordinator{}

	done := make(chan error, 1)
	go func() {
		done <- manager.Run(healthServer, connMonitor, shutdownCoord)
	}()

	time.Sleep(300 * time.Millisecond)

	currentProc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)

	currentProc.Signal(syscall.SIGHUP)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGUSR1)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGHUP)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGUSR2)
	time.Sleep(200 * time.Millisecond)

	currentProc.Signal(syscall.SIGTERM)

	select {
	case err := <-done:
		assert.NoError(t, err)
		if manager.app.Process != nil {
			manager.app.Process.Kill()
		}
	case <-time.After(3 * time.Second):
		if manager.app.Process != nil {
			manager.app.Process.Kill()
		}
		t.Fatal("Test timed out")
	}
}

func TestManager_handleSignals_ActionIgnore(t *testing.T) {
	cfg := &mockConfig{
		command: []string{"sleep", "5"},
	}

	manager := NewManager(cfg)
	manager.healthServer = &mockHealthServer{}
	manager.connMonitor = &mockConnectionMonitor{}
	manager.shutdownCoord = &mockShutdownCoordinator{}

	manager.app = exec.Command("sleep", "5")
	manager.app.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	err := manager.app.Start()
	assert.NoError(t, err)

	defer func() {
		if manager.app != nil && manager.app.Process != nil {
			manager.app.Process.Kill()
		}
	}()

	signalConfig := cfg.GetSignalConfig()
	signalHandler := NewSignalHandler(&signalConfig, manager.app.Process)

	sigChan := make(chan os.Signal, 10)

	done := make(chan error, 1)
	go func() {
		done <- manager.handleSignals(sigChan, signalHandler)
	}()

	time.Sleep(100 * time.Millisecond)

	sigChan <- syscall.SIGWINCH
	time.Sleep(100 * time.Millisecond)

	sigChan <- syscall.SIGQUIT
	time.Sleep(100 * time.Millisecond)

	sigChan <- syscall.SIGTERM

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Test timed out")
	}
}
