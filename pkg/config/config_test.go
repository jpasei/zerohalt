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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig_NotNil(t *testing.T) {
	cfg := DefaultConfig()
	assert.NotNil(t, cfg)
}

func TestDefaultConfig_AppPort(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, uint16(8080), cfg.App.Port)
}

func TestDefaultConfig_HealthPort(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, uint16(8888), cfg.Health.Port)
}

func TestDefaultConfig_HealthPath(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "/health", cfg.Health.Path)
}

func TestDefaultConfig_HealthMode(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, HealthModeStandalone, cfg.Health.Mode)
}

func TestDefaultConfig_DrainTimeout(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 60*time.Second, cfg.Shutdown.DrainTimeout)
}

func TestDefaultConfig_ShutdownTimeout(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, 30*time.Second, cfg.Shutdown.ShutdownTimeout)
}

func TestDefaultConfig_SignalToApp(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "SIGTERM", cfg.Shutdown.SignalToApp)
}

func TestDefaultConfig_ForceKillAfterTimeout(t *testing.T) {
	cfg := DefaultConfig()
	assert.True(t, cfg.Shutdown.ForceKillAfterTimeout)
}

func TestDefaultConfig_LoggingLevel(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestDefaultConfig_LoggingIncludeTimestamp(t *testing.T) {
	cfg := DefaultConfig()
	assert.True(t, cfg.Logging.IncludeTimestamp)
}

func TestDefaultConfig_ShutdownSignalsLength(t *testing.T) {
	cfg := DefaultConfig()
	assert.Len(t, cfg.Signal.ShutdownSignals, 3)
}

func TestDefaultSignalConfig_PassThroughSignalsLength(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Len(t, cfg.PassThroughSignals, 4)
}

func TestDefaultSignalConfig_PassThroughSignalsContainsSIGHUP(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Contains(t, cfg.PassThroughSignals, "SIGHUP")
}

func TestDefaultSignalConfig_PassThroughSignalsContainsSIGUSR1(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Contains(t, cfg.PassThroughSignals, "SIGUSR1")
}

func TestDefaultSignalConfig_PassThroughSignalsContainsSIGUSR2(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Contains(t, cfg.PassThroughSignals, "SIGUSR2")
}

func TestDefaultSignalConfig_PassThroughSignalsContainsSIGWINCH(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Contains(t, cfg.PassThroughSignals, "SIGWINCH")
}

func TestDefaultSignalConfig_ShutdownSignalsLength(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Len(t, cfg.ShutdownSignals, 3)
}

func TestDefaultSignalConfig_ShutdownSignalFirst(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Equal(t, "SIGTERM", cfg.ShutdownSignals[0])
}

func TestDefaultSignalConfig_ShutdownSignalSecond(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Equal(t, "SIGINT", cfg.ShutdownSignals[1])
}

func TestDefaultSignalConfig_ShutdownSignalThird(t *testing.T) {
	cfg := DefaultSignalConfig()
	assert.Equal(t, "SIGQUIT", cfg.ShutdownSignals[2])
}

func TestHealthModeStandalone(t *testing.T) {
	got := string(HealthModeStandalone)
	assert.Equal(t, "standalone", got)
}

func TestHealthModeAppDependent(t *testing.T) {
	got := string(HealthModeAppDependent)
	assert.Equal(t, "app-dependent", got)
}

func TestHealthModeHybrid(t *testing.T) {
	got := string(HealthModeHybrid)
	assert.Equal(t, "hybrid", got)
}

func TestHealthModeCommand(t *testing.T) {
	got := string(HealthModeCommand)
	assert.Equal(t, "command", got)
}
