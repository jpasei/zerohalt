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

package shutdown

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/jpasei/zerohalt/pkg/monitor"
	"github.com/stretchr/testify/assert"
)

type mockHealthServer struct {
	state int
}

func (m *mockHealthServer) SetState(state int) {
	m.state = state
}

type mockConnectionMonitor struct {
	shouldTimeout bool
}

func (m *mockConnectionMonitor) WaitForZeroConnections(timeout interface{}) error {
	return nil
}

func TestNewCoordinator(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:    60 * time.Second,
		ShutdownTimeout: 30 * time.Second,
		SignalToApp:     "SIGTERM",
	}

	coordinator := NewCoordinator(cfg, &mockHealthServer{}, &mockConnectionMonitor{}, nil)

	assert.NotNil(t, coordinator)
	assert.NotNil(t, coordinator.config)
}

func TestCoordinator_SetAppProcess(t *testing.T) {
	cfg := &ShutdownConfig{}
	coordinator := NewCoordinator(cfg, &mockHealthServer{}, &mockConnectionMonitor{}, nil)

	process := &os.Process{Pid: 123}
	coordinator.SetAppProcess(process)

	assert.Equal(t, process, coordinator.appProcess)
}

func TestCoordinator_InitiateShutdown_NoProcess(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout: 1 * time.Second,
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}
	coordinator := NewCoordinator(cfg, healthServer, connMonitor, nil)

	err := coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.NoError(t, err)
	assert.Equal(t, 2, healthServer.state)
}

func TestCoordinator_InitiateShutdown_DrainTimeout(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:    100 * time.Millisecond,
		ShutdownTimeout: 1 * time.Second,
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{shouldTimeout: true}
	coordinator := NewCoordinator(cfg, healthServer, connMonitor, nil)

	err := coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.NoError(t, err)
}

func TestCoordinator_getSignalForApp(t *testing.T) {
	tests := []struct {
		name   string
		signal string
		want   os.Signal
	}{
		{"SIGHUP", "SIGHUP", syscall.SIGHUP},
		{"SIGINT", "SIGINT", syscall.SIGINT},
		{"SIGTERM", "SIGTERM", syscall.SIGTERM},
		{"SIGUSR1", "SIGUSR1", syscall.SIGUSR1},
		{"SIGUSR2", "SIGUSR2", syscall.SIGUSR2},
		{"SIGWINCH", "SIGWINCH", syscall.SIGWINCH},
		{"SIGQUIT", "SIGQUIT", syscall.SIGQUIT},
		{"default", "UNKNOWN", syscall.SIGTERM},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ShutdownConfig{SignalToApp: tt.signal}
			coordinator := NewCoordinator(cfg, &mockHealthServer{}, &mockConnectionMonitor{}, nil)

			got := coordinator.getSignalForApp()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCoordinator_InitiateShutdown_WithProcessCleanExit(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:    100 * time.Millisecond,
		ShutdownTimeout: 2 * time.Second,
		SignalToApp:     "SIGTERM",
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}

	cmd := exec.Command("sleep", "0.1")
	err := cmd.Start()
	assert.NoError(t, err)

	coordinator := NewCoordinator(cfg, healthServer, connMonitor, cmd.Process)

	err = coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.NoError(t, err)
	assert.Equal(t, 2, healthServer.state)
}

func TestCoordinator_InitiateShutdown_SignalError(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:    100 * time.Millisecond,
		ShutdownTimeout: 2 * time.Second,
		SignalToApp:     "SIGTERM",
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}

	cmd := exec.Command("sleep", "0.1")
	err := cmd.Start()
	assert.NoError(t, err)

	cmd.Process.Wait()

	coordinator := NewCoordinator(cfg, healthServer, connMonitor, cmd.Process)

	err = coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.NoError(t, err)
	assert.Equal(t, 2, healthServer.state)
}

func TestCoordinator_InitiateShutdown_Timeout(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:          100 * time.Millisecond,
		ShutdownTimeout:       200 * time.Millisecond,
		SignalToApp:           "SIGTERM",
		ForceKillAfterTimeout: false,
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}

	cmd := exec.Command("/bin/sh", "testdata/ignore_sigterm.sh")
	err := cmd.Start()
	assert.NoError(t, err)

	defer cmd.Process.Kill()

	time.Sleep(50 * time.Millisecond)

	coordinator := NewCoordinator(cfg, healthServer, connMonitor, cmd.Process)

	err = coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.Equal(t, ErrShutdownTimeout, err)
}

func TestCoordinator_InitiateShutdown_TimeoutWithForceKill(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:          100 * time.Millisecond,
		ShutdownTimeout:       200 * time.Millisecond,
		SignalToApp:           "SIGTERM",
		ForceKillAfterTimeout: true,
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}

	cmd := exec.Command("/bin/sh", "testdata/ignore_sigterm.sh")
	err := cmd.Start()
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	coordinator := NewCoordinator(cfg, healthServer, connMonitor, cmd.Process)

	err = coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.Equal(t, ErrShutdownTimeout, err)

	time.Sleep(100 * time.Millisecond)
}

func TestCoordinator_InitiateShutdown_DrainTimeoutWithMonitorError(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:    100 * time.Millisecond,
		ShutdownTimeout: 1 * time.Second,
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitorWithTimeout{shouldTimeout: true}
	coordinator := NewCoordinator(cfg, healthServer, connMonitor, nil)

	err := coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.NoError(t, err)
}

type mockConnectionMonitorWithTimeout struct {
	shouldTimeout bool
}

func (m *mockConnectionMonitorWithTimeout) WaitForZeroConnections(timeout interface{}) error {
	return monitor.ErrDrainTimeout
}

func TestCoordinator_InitiateShutdown_WaitReturnsNoChildError(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:    100 * time.Millisecond,
		ShutdownTimeout: 1 * time.Second,
		SignalToApp:     "SIGTERM",
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}

	cmd := exec.Command("sleep", "0.05")
	err := cmd.Start()
	assert.NoError(t, err)

	cmd.Wait()

	coordinator := NewCoordinator(cfg, healthServer, connMonitor, cmd.Process)

	err = coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.NoError(t, err)
	assert.Equal(t, 2, healthServer.state)
}

func TestCoordinator_InitiateShutdown_ForceKillOnTimeout(t *testing.T) {
	cfg := &ShutdownConfig{
		DrainTimeout:          50 * time.Millisecond,
		ShutdownTimeout:       150 * time.Millisecond,
		SignalToApp:           "SIGTERM",
		ForceKillAfterTimeout: true,
	}

	healthServer := &mockHealthServer{}
	connMonitor := &mockConnectionMonitor{}

	cmd := exec.Command("/bin/sh", "-c", "trap : TERM; sleep 60")
	err := cmd.Start()
	assert.NoError(t, err)

	defer func() {
		cmd.Process.Signal(syscall.SIGKILL)
		syscall.Wait4(cmd.Process.Pid, nil, 0, nil)
	}()

	time.Sleep(50 * time.Millisecond)

	coordinator := NewCoordinator(cfg, healthServer, connMonitor, cmd.Process)

	err = coordinator.InitiateShutdown(syscall.SIGTERM)

	assert.Equal(t, ErrShutdownTimeout, err)
	assert.Equal(t, 2, healthServer.state)
}
