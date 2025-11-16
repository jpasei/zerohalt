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
	"errors"
	"log/slog"
	"os"
	"syscall"
	"time"

	"github.com/jpasei/zerohalt/pkg/health"
	"github.com/jpasei/zerohalt/pkg/metrics"
	"github.com/jpasei/zerohalt/pkg/process"
)

var (
	ErrShutdownTimeout = errors.New("shutdown timeout reached")
)

type HealthServer interface {
	SetState(state health.HealthState)
}

type ConnectionMonitor interface {
	WaitForZeroConnections(timeout interface{}) error
}

type ShutdownConfig struct {
	DrainTimeout          time.Duration
	ShutdownTimeout       time.Duration
	SignalToApp           string
	ForceKillAfterTimeout bool
}

type Coordinator struct {
	config       *ShutdownConfig
	healthServer HealthServer
	connMonitor  ConnectionMonitor
	appProcess   *os.Process
}

func NewCoordinator(
	cfg *ShutdownConfig,
	healthServer HealthServer,
	connMonitor ConnectionMonitor,
	appProcess *os.Process,
) *Coordinator {
	return &Coordinator{
		config:       cfg,
		healthServer: healthServer,
		connMonitor:  connMonitor,
		appProcess:   appProcess,
	}
}

func (c *Coordinator) SetAppProcess(appProcess *os.Process) {
	c.appProcess = appProcess
}

func (c *Coordinator) InitiateShutdown(sig os.Signal) error {
	slog.Info("Received signal, starting graceful shutdown", "signal", sig.String())

	c.healthServer.SetState(health.StateDraining)
	metrics.HealthApp.Set(float64(health.StateDraining))

	slog.Info("Health check now returning 503")

	err := c.connMonitor.WaitForZeroConnections(c.config.DrainTimeout)
	if err != nil {
		slog.Warn("Connection drain timeout", "error", err)
	} else {
		slog.Info("All connections drained")
	}

	if c.appProcess == nil {
		slog.Info("No application process to signal")
		return nil
	}

	signal := c.getSignalForApp(sig)
	if err := c.appProcess.Signal(signal); err != nil {
		slog.Error("Error sending signal to app", "error", err)
	} else {
		slog.Info("Sent signal to application", "signal", signal.String(), "pid", c.appProcess.Pid)
	}

	done := make(chan error, 1)
	go func() {
		_, err := c.appProcess.Wait()
		if err != nil {
			errMsg := err.Error()
			if errMsg == "waitid: no child processes" || errMsg == "wait: no child processes" {
				done <- nil
				return
			}
		}
		done <- err
	}()

	select {
	case err := <-done:
		slog.Info("Application exited cleanly")
		return err
	case <-time.After(c.config.ShutdownTimeout):
		if c.config.ForceKillAfterTimeout {
			c.appProcess.Signal(syscall.SIGKILL)
			slog.Warn("Sent SIGKILL after timeout")
		}
		return ErrShutdownTimeout
	}
}

func (c *Coordinator) getSignalForApp(receivedSignal os.Signal) os.Signal {
	signalToAppIsEmpty := c.config.SignalToApp == ""

	if signalToAppIsEmpty {
		return receivedSignal
	}

	signal := process.ParseSignal(c.config.SignalToApp)
	if signal == nil {
		return receivedSignal
	}

	return signal
}
