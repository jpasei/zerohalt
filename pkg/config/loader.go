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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func LoadFromEnv() (*Config, error) {
	cfg := DefaultConfig()

	if port := os.Getenv("ZEROHALT_APP_PORT"); port != "" {
		parsed, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid ZEROHALT_APP_PORT: %w", err)
		}
		cfg.App.Port = uint16(parsed)
	}

	if healthURL := os.Getenv("ZEROHALT_APP_HEALTH_URL"); healthURL != "" {
		cfg.App.HealthURL = healthURL
	}

	if timeout := os.Getenv("ZEROHALT_APP_STARTUP_TIMEOUT"); timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid ZEROHALT_APP_STARTUP_TIMEOUT: %w", err)
		}
		cfg.App.StartupTimeout = parsed
	}

	if port := os.Getenv("ZEROHALT_HEALTH_PORT"); port != "" {
		parsed, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid ZEROHALT_HEALTH_PORT: %w", err)
		}
		cfg.Health.Port = uint16(parsed)
	}

	if path := os.Getenv("ZEROHALT_HEALTH_PATH"); path != "" {
		cfg.Health.Path = path
	}

	if mode := os.Getenv("ZEROHALT_HEALTH_MODE"); mode != "" {
		cfg.Health.Mode = HealthMode(mode)
	}

	if command := os.Getenv("ZEROHALT_HEALTH_COMMAND"); command != "" {
		cfg.Health.Command = strings.Fields(command)
	}

	if timeout := os.Getenv("ZEROHALT_DRAIN_TIMEOUT"); timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid ZEROHALT_DRAIN_TIMEOUT: %w", err)
		}
		cfg.Shutdown.DrainTimeout = parsed
	}

	if timeout := os.Getenv("ZEROHALT_SHUTDOWN_TIMEOUT"); timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid ZEROHALT_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.Shutdown.ShutdownTimeout = parsed
	}

	if signal := os.Getenv("ZEROHALT_SIGNAL_TO_APP"); signal != "" {
		cfg.Shutdown.SignalToApp = signal
	}

	if level := os.Getenv("ZEROHALT_LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}

	if passthrough := os.Getenv("ZEROHALT_PASSTHROUGH_SIGNALS"); passthrough != "" {
		cfg.Signal.PassThroughSignals = strings.Split(passthrough, ",")
	}

	if shutdown := os.Getenv("ZEROHALT_SHUTDOWN_SIGNALS"); shutdown != "" {
		cfg.Signal.ShutdownSignals = strings.Split(shutdown, ",")
	}

	return cfg, cfg.Validate()
}

func (c *Config) Validate() error {
	if c.Health.Port == 0 {
		return fmt.Errorf("health check port must be specified")
	}

	if c.Health.Path == "" {
		return fmt.Errorf("health check path must be specified")
	}

	validModes := map[HealthMode]bool{
		HealthModeStandalone:   true,
		HealthModeAppDependent: true,
		HealthModeHybrid:       true,
		HealthModeCommand:      true,
	}
	if !validModes[c.Health.Mode] {
		return fmt.Errorf("invalid health mode: %s", c.Health.Mode)
	}

	if c.Shutdown.DrainTimeout <= 0 {
		return fmt.Errorf("drain timeout must be positive")
	}

	if c.Shutdown.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}

	validSignals := map[string]bool{
		"SIGHUP":   true,
		"SIGINT":   true,
		"SIGTERM":  true,
		"SIGUSR1":  true,
		"SIGUSR2":  true,
		"SIGWINCH": true,
		"SIGQUIT":  true,
	}

	for _, sig := range c.Signal.PassThroughSignals {
		if !validSignals[sig] {
			return fmt.Errorf("invalid pass-through signal: %s", sig)
		}
	}

	for _, sig := range c.Signal.ShutdownSignals {
		if sig != "SIGTERM" && sig != "SIGINT" {
			return fmt.Errorf("invalid shutdown signal: %s", sig)
		}
	}

	if err := c.validateSignalConflicts(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateSignalConflicts() error {
	shutdownMap := make(map[string]bool)

	for _, sig := range c.Signal.ShutdownSignals {
		shutdownMap[sig] = true
	}

	for _, pt := range c.Signal.PassThroughSignals {
		if shutdownMap[pt] {
			return fmt.Errorf("signal %s cannot be both pass-through and shutdown signal", pt)
		}
	}

	return nil
}
