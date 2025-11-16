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

package main

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jpasei/zerohalt/pkg/config"
	"github.com/jpasei/zerohalt/pkg/health"
	"github.com/jpasei/zerohalt/pkg/monitor"
	"github.com/stretchr/testify/assert"
)

func TestConfigAdapter_GetAppCommand(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			Command: []string{"nginx", "-g", "daemon off;"},
		},
	}

	adapter := &ConfigAdapter{Config: cfg}
	cmd := adapter.GetAppCommand()

	assert.Len(t, cmd, 3)
	assert.Equal(t, "nginx", cmd[0])
}

func TestConfigAdapter_GetAppPort(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{Port: 8080},
	}

	adapter := &ConfigAdapter{Config: cfg}
	port := adapter.GetAppPort()

	assert.Equal(t, uint16(8080), port)
}

func TestConfigAdapter_GetAdditionalPorts(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{AdditionalPorts: []uint16{9090, 9091}},
	}

	adapter := &ConfigAdapter{Config: cfg}
	ports := adapter.GetAdditionalPorts()

	assert.Len(t, ports, 2)
}

func TestConfigAdapter_GetHealthPort(t *testing.T) {
	testPort := uint16(8888)
	cfg := &config.Config{
		Health: config.HealthConfig{Port: testPort},
	}

	adapter := &ConfigAdapter{Config: cfg}
	port := adapter.GetHealthPort()

	assert.Equal(t, testPort, port)
}

func TestConfigAdapter_GetHealthPath(t *testing.T) {
	cfg := &config.Config{
		Health: config.HealthConfig{Path: "/health"},
	}

	adapter := &ConfigAdapter{Config: cfg}
	path := adapter.GetHealthPath()

	assert.Equal(t, "/health", path)
}

func TestConfigAdapter_GetSignalConfig(t *testing.T) {
	cfg := &config.Config{
		Signal: config.SignalConfig{
			ShutdownSignals: []string{"SIGTERM", "SIGINT"},
		},
	}

	adapter := &ConfigAdapter{Config: cfg}
	sigCfg := adapter.GetSignalConfig()

	assert.Len(t, sigCfg.ShutdownSignals, 2)
}

func TestConfigAdapter_GetConnectionCheckInterval(t *testing.T) {
	cfg := &config.Config{
		Shutdown: config.ShutdownConfig{
			ConnectionCheckInterval: 1 * time.Second,
		},
	}

	adapter := &ConfigAdapter{Config: cfg}
	interval := adapter.GetConnectionCheckInterval()

	assert.Equal(t, 1*time.Second, interval.(time.Duration))
}

func TestShutdownConfigAdapter_GetDrainTimeout(t *testing.T) {
	cfg := &config.ShutdownConfig{
		DrainTimeout: 60 * time.Second,
	}

	adapter := &ShutdownConfigAdapter{ShutdownConfig: cfg}
	timeout := adapter.GetDrainTimeout()

	assert.Equal(t, 60*time.Second, timeout.(time.Duration))
}

func TestShutdownConfigAdapter_GetShutdownTimeout(t *testing.T) {
	cfg := &config.ShutdownConfig{
		ShutdownTimeout: 30 * time.Second,
	}

	adapter := &ShutdownConfigAdapter{ShutdownConfig: cfg}
	timeout := adapter.GetShutdownTimeout()

	assert.Equal(t, 30*time.Second, timeout.(time.Duration))
}

func TestShutdownConfigAdapter_GetSignalToApp(t *testing.T) {
	cfg := &config.ShutdownConfig{
		SignalToApp: "SIGTERM",
	}

	adapter := &ShutdownConfigAdapter{ShutdownConfig: cfg}
	signal := adapter.GetSignalToApp()

	assert.Equal(t, "SIGTERM", signal)
}

func TestShutdownConfigAdapter_GetForceKillAfterTimeout(t *testing.T) {
	tests := []struct {
		name  string
		value bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.ShutdownConfig{
				ForceKillAfterTimeout: tt.value,
			}

			adapter := &ShutdownConfigAdapter{ShutdownConfig: cfg}
			result := adapter.GetForceKillAfterTimeout()

			assert.Equal(t, tt.value, result)
		})
	}
}

func TestConfigAdapter_GetShutdownConfig(t *testing.T) {
	cfg := &config.Config{
		Shutdown: config.ShutdownConfig{
			DrainTimeout:    60 * time.Second,
			ShutdownTimeout: 30 * time.Second,
			SignalToApp:     "SIGTERM",
		},
	}

	adapter := &ConfigAdapter{Config: cfg}
	shutdownCfg := adapter.GetShutdownConfig()

	assert.NotNil(t, shutdownCfg)

	drainTimeout := shutdownCfg.GetDrainTimeout()
	assert.Equal(t, 60*time.Second, drainTimeout.(time.Duration))
}

func TestHealthServerAdapter_SetState(t *testing.T) {
	port := getAvailablePort()
	server := &HealthServerAdapter{
		Server: health.NewServer(port, "/health"),
	}

	server.SetState(health.StateHealthy)
	state := server.GetState()

	assert.Equal(t, health.StateHealthy, state)
}

func TestHealthServerAdapter_GetState(t *testing.T) {
	port := getAvailablePort()
	server := &HealthServerAdapter{
		Server: health.NewServer(port, "/health"),
	}

	state := server.GetState()

	assert.GreaterOrEqual(t, state, health.StateStarting)
}

func TestMonitorAdapter_WaitForZeroConnections(t *testing.T) {
	mon := &MonitorAdapter{
		Monitor: monitor.NewMonitor([]uint16{8080}, 100*time.Millisecond),
	}

	err := mon.WaitForZeroConnections(1 * time.Millisecond)

	t.Logf("WaitForZeroConnections() error = %v (expected for test)", err)
}

func TestRun_Version(t *testing.T) {
	args := []string{"zerohalt", "--version"}

	exitCode := run(args)

	assert.Equal(t, 0, exitCode)
}

func TestRun_NoArgs(t *testing.T) {
	args := []string{"zerohalt"}

	exitCode := run(args)

	assert.Equal(t, 1, exitCode)
}

func TestRun_ConfigError(t *testing.T) {
	os.Setenv("ZEROHALT_APP_PORT", "invalid")
	defer os.Unsetenv("ZEROHALT_APP_PORT")

	args := []string{"zerohalt", "sleep", "1"}

	exitCode := run(args)

	assert.Equal(t, 1, exitCode)
}

func TestRun_InvalidCommand(t *testing.T) {
	os.Unsetenv("ZEROHALT_APP_PORT")
	port := getAvailablePort()
	os.Setenv("ZEROHALT_HEALTH_PORT", fmt.Sprintf("%d", port))
	defer os.Unsetenv("ZEROHALT_HEALTH_PORT")

	args := []string{"zerohalt", "/nonexistent/invalid/command"}

	exitCode := run(args)

	assert.Equal(t, 1, exitCode)
}

func TestRun_SuccessfulExecution(t *testing.T) {
	os.Unsetenv("ZEROHALT_APP_PORT")
	port := getAvailablePort()
	os.Setenv("ZEROHALT_HEALTH_PORT", fmt.Sprintf("%d", port))
	defer os.Unsetenv("ZEROHALT_HEALTH_PORT")
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "0")
	defer os.Unsetenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT")

	args := []string{"zerohalt", "sleep", "0.1"}

	done := make(chan int)
	go func() {
		done <- run(args)
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		proc, err := os.FindProcess(os.Getpid())
		if err == nil {
			proc.Signal(os.Interrupt)
		}
	}()

	select {
	case exitCode := <-done:
		assert.Equal(t, 0, exitCode)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out after 5 seconds")
	}
}

func TestMain_CallsRun(t *testing.T) {
	exitCalled := false
	var exitCode int

	osExit = func(code int) {
		exitCalled = true
		exitCode = code
	}
	defer func() {
		osExit = os.Exit
	}()

	oldArgs := os.Args
	os.Args = []string{"zerohalt", "--version"}
	defer func() {
		os.Args = oldArgs
	}()

	main()

	assert.True(t, exitCalled)
	assert.Equal(t, 0, exitCode)
}

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warning"},
		{"error", "error"},
		{"default", "invalid"},
		{"uppercase", "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupLogger(tt.level)
		})
	}
}

func TestRun_MetricsOnSamePortAsHealth(t *testing.T) {
	os.Unsetenv("ZEROHALT_APP_PORT")
	healthPort := getAvailablePort()
	os.Setenv("ZEROHALT_HEALTH_PORT", fmt.Sprintf("%d", healthPort))
	os.Setenv("ZEROHALT_METRICS_ENABLED", "true")
	os.Setenv("ZEROHALT_METRICS_PORT", fmt.Sprintf("%d", healthPort))
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "0")
	defer func() {
		os.Unsetenv("ZEROHALT_HEALTH_PORT")
		os.Unsetenv("ZEROHALT_METRICS_ENABLED")
		os.Unsetenv("ZEROHALT_METRICS_PORT")
		os.Unsetenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT")
	}()

	args := []string{"zerohalt", "sleep", "0.1"}

	done := make(chan int)
	go func() {
		done <- run(args)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", healthPort))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	proc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	proc.Signal(os.Interrupt)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out after 5 seconds")
	}
}

func TestRun_MetricsOnDifferentPort(t *testing.T) {
	os.Unsetenv("ZEROHALT_APP_PORT")
	healthPort := getAvailablePort()
	metricsPort := getAvailablePort()
	os.Setenv("ZEROHALT_HEALTH_PORT", fmt.Sprintf("%d", healthPort))
	os.Setenv("ZEROHALT_METRICS_ENABLED", "true")
	os.Setenv("ZEROHALT_METRICS_PORT", fmt.Sprintf("%d", metricsPort))
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "0")
	defer func() {
		os.Unsetenv("ZEROHALT_HEALTH_PORT")
		os.Unsetenv("ZEROHALT_METRICS_ENABLED")
		os.Unsetenv("ZEROHALT_METRICS_PORT")
		os.Unsetenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT")
	}()

	args := []string{"zerohalt", "sleep", "0.1"}

	done := make(chan int)
	go func() {
		done <- run(args)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", metricsPort))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	proc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	proc.Signal(os.Interrupt)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out after 5 seconds")
	}
}

func TestRun_MetricsNotOnHealthPortWhenDifferentPortConfigured(t *testing.T) {
	os.Unsetenv("ZEROHALT_APP_PORT")
	healthPort := getAvailablePort()
	metricsPort := getAvailablePort()
	os.Setenv("ZEROHALT_HEALTH_PORT", fmt.Sprintf("%d", healthPort))
	os.Setenv("ZEROHALT_METRICS_ENABLED", "true")
	os.Setenv("ZEROHALT_METRICS_PORT", fmt.Sprintf("%d", metricsPort))
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "0")
	defer func() {
		os.Unsetenv("ZEROHALT_HEALTH_PORT")
		os.Unsetenv("ZEROHALT_METRICS_ENABLED")
		os.Unsetenv("ZEROHALT_METRICS_PORT")
		os.Unsetenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT")
	}()

	args := []string{"zerohalt", "sleep", "0.1"}

	done := make(chan int)
	go func() {
		done <- run(args)
	}()

	time.Sleep(200 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", healthPort))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	proc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)
	proc.Signal(os.Interrupt)

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out after 5 seconds")
	}
}

func TestRun_AppDependentHealthMode(t *testing.T) {
	os.Unsetenv("ZEROHALT_APP_PORT")
	healthPort := getAvailablePort()
	appPort := getAvailablePort()
	os.Setenv("ZEROHALT_HEALTH_PORT", fmt.Sprintf("%d", healthPort))
	os.Setenv("ZEROHALT_HEALTH_MODE", "app-dependent")
	os.Setenv("ZEROHALT_APP_HEALTH_URL", fmt.Sprintf("http://127.0.0.1:%d/health", appPort))
	os.Setenv("ZEROHALT_APP_STARTUP_TIMEOUT", "1s")
	os.Setenv("ZEROHALT_HEALTH_PROBE_INTERVAL", "100ms")
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "0")
	defer func() {
		os.Unsetenv("ZEROHALT_HEALTH_PORT")
		os.Unsetenv("ZEROHALT_HEALTH_MODE")
		os.Unsetenv("ZEROHALT_APP_HEALTH_URL")
		os.Unsetenv("ZEROHALT_APP_STARTUP_TIMEOUT")
		os.Unsetenv("ZEROHALT_HEALTH_PROBE_INTERVAL")
		os.Unsetenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT")
	}()

	args := []string{"zerohalt", "sleep", "0.1"}

	done := make(chan int)
	go func() {
		done <- run(args)
	}()

	go func() {
		time.Sleep(2000 * time.Millisecond)
		proc, err := os.FindProcess(os.Getpid())
		if err == nil {
			proc.Signal(os.Interrupt)
		}
	}()

	select {
	case exitCode := <-done:
		assert.Equal(t, 0, exitCode)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out after 5 seconds")
	}
}

func TestRun_MetricsServerError(t *testing.T) {
	os.Unsetenv("ZEROHALT_APP_PORT")
	healthPort := getAvailablePort()
	metricsPort := getAvailablePort()

	blocker := &http.Server{
		Addr: fmt.Sprintf(":%d", metricsPort),
	}
	go blocker.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer blocker.Close()

	os.Setenv("ZEROHALT_HEALTH_PORT", fmt.Sprintf("%d", healthPort))
	os.Setenv("ZEROHALT_METRICS_ENABLED", "true")
	os.Setenv("ZEROHALT_METRICS_PORT", fmt.Sprintf("%d", metricsPort))
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "0")
	defer func() {
		os.Unsetenv("ZEROHALT_HEALTH_PORT")
		os.Unsetenv("ZEROHALT_METRICS_ENABLED")
		os.Unsetenv("ZEROHALT_METRICS_PORT")
		os.Unsetenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT")
	}()

	args := []string{"zerohalt", "sleep", "0.1"}

	done := make(chan int)
	go func() {
		done <- run(args)
	}()

	go func() {
		time.Sleep(500 * time.Millisecond)
		proc, err := os.FindProcess(os.Getpid())
		if err == nil {
			proc.Signal(os.Interrupt)
		}
	}()

	select {
	case exitCode := <-done:
		assert.Equal(t, 0, exitCode)
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out after 5 seconds")
	}
}

func TestRunMetricsServer_ServerClosed(t *testing.T) {
	metricsPort := getAvailablePort()
	cfg := &config.Config{
		Metrics: config.MetricsConfig{
			Port: metricsPort,
			Path: "/metrics",
		},
	}

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", metricsPort),
	}

	done := make(chan bool)
	go func() {
		runMetricsServer(server, cfg)
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)
	err := server.Close()
	assert.NoError(t, err)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out waiting for server to close")
	}
}

func TestRunMetricsServer_ListenError(t *testing.T) {
	metricsPort := getAvailablePort()

	blocker := &http.Server{
		Addr: fmt.Sprintf(":%d", metricsPort),
	}
	go blocker.ListenAndServe()
	time.Sleep(100 * time.Millisecond)
	defer blocker.Close()

	cfg := &config.Config{
		Metrics: config.MetricsConfig{
			Port: metricsPort,
			Path: "/metrics",
		},
	}

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", metricsPort),
	}

	done := make(chan bool)
	go func() {
		runMetricsServer(server, cfg)
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Test timed out waiting for server error")
	}
}
