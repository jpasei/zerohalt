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

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadFromEnv_Defaults(t *testing.T) {
	os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, uint16(8080), cfg.App.Port)
	assert.Equal(t, uint16(8888), cfg.Health.Port)
}

func TestLoadFromEnv_CustomValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_APP_PORT", "9090")
	os.Setenv("ZEROHALT_HEALTH_PORT", "9091")
	os.Setenv("ZEROHALT_HEALTH_PATH", "/status")
	os.Setenv("ZEROHALT_LOG_LEVEL", "debug")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, uint16(9090), cfg.App.Port)
	assert.Equal(t, uint16(9091), cfg.Health.Port)
	assert.Equal(t, "/status", cfg.Health.Path)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestLoadFromEnv_InvalidPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_APP_PORT", "invalid")
	defer os.Clearenv()

	_, err := LoadFromEnv()
	assert.Error(t, err)
}

func TestLoadFromEnv_InvalidHealthPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_HEALTH_PORT", "99999")
	defer os.Clearenv()

	_, err := LoadFromEnv()
	assert.Error(t, err)
}

func TestLoadFromEnv_InvalidTimeout(t *testing.T) {
	tests := []struct {
		name string
		env  string
		val  string
	}{
		{"invalid startup timeout", "ZEROHALT_APP_STARTUP_TIMEOUT", "invalid"},
		{"invalid drain timeout", "ZEROHALT_DRAIN_TIMEOUT", "not-a-duration"},
		{"invalid shutdown timeout", "ZEROHALT_SHUTDOWN_TIMEOUT", "bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv(tt.env, tt.val)
			defer os.Clearenv()

			_, err := LoadFromEnv()
			assert.Error(t, err)
		})
	}
}

func TestLoadFromEnv_Timeouts(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_DRAIN_TIMEOUT", "90s")
	os.Setenv("ZEROHALT_SHUTDOWN_TIMEOUT", "45s")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, 90*time.Second, cfg.Shutdown.DrainTimeout)
	assert.Equal(t, 45*time.Second, cfg.Shutdown.ShutdownTimeout)
}

func TestLoadFromEnv_Signals(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_PASSTHROUGH_SIGNALS", "SIGHUP,SIGUSR1")
	os.Setenv("ZEROHALT_SHUTDOWN_SIGNALS", "SIGTERM,SIGINT")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Len(t, cfg.Signal.PassThroughSignals, 2)
	assert.Len(t, cfg.Signal.ShutdownSignals, 2)
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := DefaultConfig()

	err := cfg.Validate()
	assert.NoError(t, err)
}

func TestValidate_InvalidHealthPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Health.Port = 0

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidate_InvalidHealthPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Health.Path = ""

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidate_InvalidHealthMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Health.Mode = HealthMode("invalid")

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidate_InvalidDrainTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Shutdown.DrainTimeout = 0

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidate_InvalidShutdownTimeout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Shutdown.ShutdownTimeout = -1 * time.Second

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidate_InvalidPassThroughSignal(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Signal.PassThroughSignals = []string{"INVALID"}

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidate_InvalidShutdownSignal(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Signal.ShutdownSignals = []string{"INVALID_SIGNAL"}

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidate_SignalConflict(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Signal.PassThroughSignals = []string{"SIGTERM"}
	cfg.Signal.ShutdownSignals = []string{"SIGTERM", "SIGINT"}

	err := cfg.Validate()
	assert.Error(t, err)
}

func TestValidateSignalConflicts(t *testing.T) {
	tests := []struct {
		name        string
		passthrough []string
		shutdown    []string
		wantErr     bool
	}{
		{
			name:        "no conflict",
			passthrough: []string{"SIGHUP"},
			shutdown:    []string{"SIGTERM", "SIGINT"},
			wantErr:     false,
		},
		{
			name:        "conflict with SIGTERM",
			passthrough: []string{"SIGTERM"},
			shutdown:    []string{"SIGTERM", "SIGINT"},
			wantErr:     true,
		},
		{
			name:        "conflict with SIGINT",
			passthrough: []string{"SIGINT"},
			shutdown:    []string{"SIGTERM", "SIGINT"},
			wantErr:     true,
		},
		{
			name:        "empty passthrough",
			passthrough: []string{},
			shutdown:    []string{"SIGTERM", "SIGINT"},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Signal.PassThroughSignals = tt.passthrough
			cfg.Signal.ShutdownSignals = tt.shutdown

			err := cfg.validateSignalConflicts()
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestLoadFromEnv_HealthURL(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_APP_HEALTH_URL", "http://localhost:8080/health")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/health", cfg.App.HealthURL)
}

func TestLoadFromEnv_StartupTimeout(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_APP_STARTUP_TIMEOUT", "120s")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, 120*time.Second, cfg.App.StartupTimeout)
}

func TestLoadFromEnv_HealthMode(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_HEALTH_MODE", "command")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, HealthModeCommand, cfg.Health.Mode)
}

func TestLoadFromEnv_HealthCommand(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_HEALTH_COMMAND", "curl -f http://localhost:8080/health")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Len(t, cfg.Health.Command, 3)
	assert.Equal(t, "curl", cfg.Health.Command[0])
}

func TestLoadFromEnv_SignalToApp(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_SIGNAL_TO_APP", "SIGINT")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "SIGINT", cfg.Shutdown.SignalToApp)
}

func TestLoadFromEnv_HealthProbeInterval(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_HEALTH_PROBE_INTERVAL", "10s")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, cfg.Health.ProbeInterval)
}

func TestLoadFromEnv_InvalidHealthProbeInterval(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_HEALTH_PROBE_INTERVAL", "invalid")
	defer os.Clearenv()

	_, err := LoadFromEnv()
	assert.Error(t, err)
}

func TestLoadFromEnv_MetricsEnabled(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_METRICS_ENABLED", "true")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.True(t, cfg.Metrics.Enabled)
}

func TestLoadFromEnv_MetricsPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_METRICS_PORT", "9999")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, uint16(9999), cfg.Metrics.Port)
}

func TestLoadFromEnv_MetricsPath(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_METRICS_PATH", "/custom-metrics")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, "/custom-metrics", cfg.Metrics.Path)
}

func TestLoadFromEnv_InvalidMetricsPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_METRICS_PORT", "invalid")
	defer os.Clearenv()

	_, err := LoadFromEnv()
	assert.Error(t, err)
}

func TestLoadFromEnv_DrainSteadyStateWait(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "10s")
	defer os.Clearenv()

	cfg, err := LoadFromEnv()
	assert.NoError(t, err)
	assert.Equal(t, 10*time.Second, cfg.Shutdown.DrainSteadyStateWait)
}

func TestLoadFromEnv_InvalidDrainSteadyStateWait(t *testing.T) {
	os.Clearenv()
	os.Setenv("ZEROHALT_DRAIN_STEADY_STATE_WAIT", "invalid")
	defer os.Clearenv()

	_, err := LoadFromEnv()
	assert.Error(t, err)
}
