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
	"time"
)

type Config struct {
	App      AppConfig
	Health   HealthConfig
	Shutdown ShutdownConfig
	Logging  LoggingConfig
	Signal   SignalConfig
	Metrics  MetricsConfig
}

type AppConfig struct {
	Command         []string
	Port            uint16
	AdditionalPorts []uint16
	HealthURL       string
	StartupTimeout  time.Duration
}

type HealthConfig struct {
	Port           uint16
	Path           string
	Mode           HealthMode
	ProbeInterval  time.Duration
	ProbeTimeout   time.Duration
	Command        []string
	CommandTimeout time.Duration
}

type HealthMode string

const (
	HealthModeStandalone   HealthMode = "standalone"
	HealthModeAppDependent HealthMode = "app-dependent"
	HealthModeHybrid       HealthMode = "hybrid"
	HealthModeCommand      HealthMode = "command"
)

type ShutdownConfig struct {
	DrainTimeout            time.Duration
	ShutdownTimeout         time.Duration
	ConnectionCheckInterval time.Duration
	SignalToApp             string
	ForceKillAfterTimeout   bool
	DrainStrategy           string
	ConnectionIdleThreshold time.Duration
	MaxConnectionAge        time.Duration
}

type LoggingConfig struct {
	Level            string
	IncludeTimestamp bool
}

type SignalConfig struct {
	PassThroughSignals []string
	ShutdownSignals    []string
}

type MetricsConfig struct {
	Enabled bool
	Port    uint16
	Path    string
}

func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Port:           uint16(8080),
			HealthURL:      "http://localhost:8080/health",
			StartupTimeout: 30 * time.Second,
		},
		Health: HealthConfig{
			Port:           uint16(8888),
			Path:           "/health",
			Mode:           HealthModeStandalone,
			ProbeInterval:  5 * time.Second,
			ProbeTimeout:   2 * time.Second,
			Command:        []string{},
			CommandTimeout: 5 * time.Second,
		},
		Shutdown: ShutdownConfig{
			DrainTimeout:            60 * time.Second,
			ShutdownTimeout:         30 * time.Second,
			ConnectionCheckInterval: 1 * time.Second,
			SignalToApp:             "SIGTERM",
			ForceKillAfterTimeout:   true,
			DrainStrategy:           "connections",
			ConnectionIdleThreshold: 30 * time.Second,
			MaxConnectionAge:        0,
		},
		Logging: LoggingConfig{
			Level:            "info",
			IncludeTimestamp: true,
		},
		Signal: DefaultSignalConfig(),
		Metrics: MetricsConfig{
			Enabled: false,
			Port:    uint16(8888),
			Path:    "/metrics",
		},
	}
}

func DefaultSignalConfig() SignalConfig {
	return SignalConfig{
		PassThroughSignals: []string{"SIGHUP", "SIGUSR1", "SIGUSR2", "SIGWINCH"},
		ShutdownSignals:    []string{"SIGTERM", "SIGINT", "SIGQUIT"},
	}
}
