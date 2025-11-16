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
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jpasei/zerohalt/pkg/config"
	"github.com/jpasei/zerohalt/pkg/health"
	"github.com/jpasei/zerohalt/pkg/metrics"
	"github.com/jpasei/zerohalt/pkg/monitor"
	"github.com/jpasei/zerohalt/pkg/process"
	"github.com/jpasei/zerohalt/pkg/shutdown"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

type ConfigAdapter struct {
	*config.Config
}

func (c *ConfigAdapter) GetAppCommand() []string {
	return c.App.Command
}

func (c *ConfigAdapter) GetAppPort() uint16 {
	return c.App.Port
}

func (c *ConfigAdapter) GetAdditionalPorts() []uint16 {
	return c.App.AdditionalPorts
}

func (c *ConfigAdapter) GetHealthPort() uint16 {
	return c.Health.Port
}

func (c *ConfigAdapter) GetHealthPath() string {
	return c.Health.Path
}

func (c *ConfigAdapter) GetShutdownConfig() process.ShutdownConfig {
	return &ShutdownConfigAdapter{&c.Shutdown}
}

func (c *ConfigAdapter) GetSignalConfig() process.SignalConfig {
	return process.SignalConfig{
		PassThroughSignals: c.Signal.PassThroughSignals,
		ShutdownSignals:    c.Signal.ShutdownSignals,
	}
}

func (c *ConfigAdapter) GetConnectionCheckInterval() interface{} {
	return c.Shutdown.ConnectionCheckInterval
}

func (c *ConfigAdapter) GetAppStartupTimeout() time.Duration {
	return c.App.StartupTimeout
}

func (c *ConfigAdapter) GetHealthProbeInterval() time.Duration {
	return c.Health.ProbeInterval
}

type ShutdownConfigAdapter struct {
	*config.ShutdownConfig
}

func (s *ShutdownConfigAdapter) GetDrainTimeout() interface{} {
	return s.DrainTimeout
}

func (s *ShutdownConfigAdapter) GetShutdownTimeout() interface{} {
	return s.ShutdownTimeout
}

func (s *ShutdownConfigAdapter) GetSignalToApp() string {
	return s.SignalToApp
}

func (s *ShutdownConfigAdapter) GetForceKillAfterTimeout() bool {
	return s.ForceKillAfterTimeout
}

type HealthServerAdapter struct {
	*health.Server
}

func (h *HealthServerAdapter) SetState(state int) {
	h.Server.SetState(health.HealthState(state))
}

func (h *HealthServerAdapter) GetState() int {
	return int(h.Server.GetState())
}

func (h *HealthServerAdapter) WaitForAppHealthy(timeout time.Duration, interval time.Duration) bool {
	return h.Server.WaitForAppHealthy(timeout, interval)
}

type MonitorAdapter struct {
	*monitor.Monitor
}

func (m *MonitorAdapter) WaitForZeroConnections(timeout interface{}) error {
	return m.Monitor.WaitForZeroConnections(timeout.(time.Duration))
}

var osExit = os.Exit

func setupLogger(level string) {
	var logLevel slog.Level

	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	slog.SetDefault(slog.New(handler))
}

func setupMetrics(cfg *config.Config, healthServer *HealthServerAdapter) {
	metricsOnSamePort := cfg.Metrics.Port == cfg.Health.Port

	if metricsOnSamePort {
		enableMetricsOnHealthServer(cfg, healthServer)
		return
	}

	startSeparateMetricsServer(cfg)
	startUptimeTracker()
}

func enableMetricsOnHealthServer(cfg *config.Config, healthServer *HealthServerAdapter) {
	healthServer.Server.EnableMetrics(cfg.Metrics.Path)
	slog.Info("Metrics enabled on health server", "path", cfg.Metrics.Path, "port", cfg.Health.Port)
	startUptimeTracker()
}

func startSeparateMetricsServer(cfg *config.Config) {
	mux := http.NewServeMux()
	mux.Handle(cfg.Metrics.Path, metrics.Handler())

	metricsAddr := fmt.Sprintf(":%d", cfg.Metrics.Port)
	metricsServer := &http.Server{
		Addr:    metricsAddr,
		Handler: mux,
	}

	go runMetricsServer(metricsServer, cfg)
}

func runMetricsServer(server *http.Server, cfg *config.Config) {
	slog.Info("Starting metrics server", "path", cfg.Metrics.Path, "port", cfg.Metrics.Port)
	err := server.ListenAndServe()

	isServerClosed := err == http.ErrServerClosed
	if isServerClosed {
		return
	}

	if err != nil {
		slog.Error("Metrics server error", "error", err)
	}
}

func startUptimeTracker() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			metrics.Uptime.Inc()
		}
	}()
}

func run(args []string) int {
	if len(args) > 1 && args[1] == "--version" {
		fmt.Printf("zerohalt version %s (commit: %s, built: %s)\n", Version, Commit, BuildTime)
		return 0
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		setupLogger("error")
		slog.Error("Configuration error", "error", err)
		return 1
	}

	setupLogger(cfg.Logging.Level)

	slog.Info("Starting Zerohalt", "version", Version)
	slog.Debug("Configuration loaded", "app_port", cfg.App.Port, "health_port", cfg.Health.Port)

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: zerohalt <command> [args...]")
		fmt.Fprintln(os.Stderr, "Example: zerohalt nginx -g 'daemon off;'")
		return 1
	}

	cfg.App.Command = args[1:]

	slog.Info("Application command", "command", cfg.App.Command)

	var healthServer *HealthServerAdapter
	if cfg.Health.Mode == config.HealthModeAppDependent {
		appChecker := health.NewAppHealthChecker(cfg.App.HealthURL, cfg.Health.ProbeTimeout)
		healthServer = &HealthServerAdapter{
			Server: health.NewServerWithAppChecker(cfg.Health.Port, cfg.Health.Path, appChecker),
		}
		slog.Info("Health server created in app-dependent mode", "app_health_url", cfg.App.HealthURL)
	} else {
		healthServer = &HealthServerAdapter{
			Server: health.NewServer(cfg.Health.Port, cfg.Health.Path),
		}
		slog.Info("Health server created in standalone mode")
	}

	if cfg.Metrics.Enabled {
		setupMetrics(cfg, healthServer)
	}

	ports := []uint16{cfg.App.Port}
	ports = append(ports, cfg.App.AdditionalPorts...)

	connMonitor := &MonitorAdapter{
		Monitor: monitor.NewMonitor(ports, cfg.Shutdown.ConnectionCheckInterval),
	}
	connMonitor.Monitor.Start()
	slog.Info("Connection monitoring started", "ports", ports, "interval", cfg.Shutdown.ConnectionCheckInterval)

	configAdapter := &ConfigAdapter{Config: cfg}
	manager := process.NewManager(configAdapter)

	shutdownCoord := shutdown.NewCoordinator(
		&shutdown.ShutdownConfig{
			DrainTimeout:          cfg.Shutdown.DrainTimeout,
			ShutdownTimeout:       cfg.Shutdown.ShutdownTimeout,
			SignalToApp:           cfg.Shutdown.SignalToApp,
			ForceKillAfterTimeout: cfg.Shutdown.ForceKillAfterTimeout,
		},
		healthServer,
		connMonitor,
		nil,
	)

	if err := manager.Run(healthServer, connMonitor, shutdownCoord); err != nil {
		slog.Error("Manager error", "error", err)
		return 1
	}

	slog.Info("Process manager shutting down")
	return 0
}

func main() {
	osExit(run(os.Args))
}
