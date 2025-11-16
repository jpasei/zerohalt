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

package monitor

import (
	"errors"
	"log/slog"
	"time"

	"github.com/jpasei/zerohalt/pkg/metrics"
)

var (
	ErrDrainTimeout = errors.New("connection drain timeout reached")
)

type Monitor struct {
	ports           []uint16
	interval        time.Duration
	steadyStateWait time.Duration
}

func NewMonitor(ports []uint16, interval time.Duration) *Monitor {
	return &Monitor{
		ports:           ports,
		interval:        interval,
		steadyStateWait: 0,
	}
}

func (m *Monitor) SetSteadyStateWait(wait time.Duration) {
	m.steadyStateWait = wait
}

func (m *Monitor) Start() {
	go m.runMonitoringLoop()
}

func (m *Monitor) runMonitoringLoop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for range ticker.C {
		m.CountActiveConnections()
	}
}

func (m *Monitor) CountActiveConnections() (int, error) {
	tcpConns, err := parseProcNetTCP("/proc/net/tcp")
	if err != nil {
		slog.Error("Failed to parse /proc/net/tcp", "error", err)
		return 0, err
	}

	tcp6Conns, err := parseProcNetTCP("/proc/net/tcp6")
	if err != nil {
		slog.Error("Failed to parse /proc/net/tcp6", "error", err)
		return 0, err
	}

	allConns := append(tcpConns, tcp6Conns...)

	count := 0
	for _, conn := range allConns {
		isMonitored := m.isMonitoredPort(conn.LocalPort)
		isActive := m.isActiveState(conn.State)

		if isMonitored && isActive {
			count++
		}
	}

	metrics.ActiveConnections.Set(float64(count))
	slog.Debug("Active connections counted", "count", count, "monitored_ports", m.ports)

	return count, nil
}

func (m *Monitor) WaitForZeroConnections(timeout time.Duration) error {
	start := time.Now()
	metrics.DrainPhaseActive.Set(1)
	defer func() {
		metrics.DrainPhaseActive.Set(0)
		metrics.DrainDuration.Set(time.Since(start).Seconds())
	}()

	deadline := time.Now().Add(timeout)
	slog.Info("Waiting for connections to drain", "timeout", timeout, "check_interval", m.interval, "steady_state_wait", m.steadyStateWait)

	count, err := m.CountActiveConnections()
	if err != nil {
		slog.Error("Error counting active connections", "error", err)
		return err
	}

	steadyStateEnabled := m.steadyStateWait > 0

	if count == 0 {
		shouldWaitForSteadyState := steadyStateEnabled

		if shouldWaitForSteadyState {
			return m.waitForSteadyState(deadline)
		}

		slog.Info("All connections drained successfully")
		return nil
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			count, err := m.CountActiveConnections()
			if err != nil {
				slog.Error("Error counting active connections", "error", err)
				return err
			}

			if count == 0 {
				shouldWaitForSteadyState := steadyStateEnabled

				if shouldWaitForSteadyState {
					return m.waitForSteadyState(deadline)
				}

				slog.Info("All connections drained successfully")
				return nil
			}

			slog.Debug("Connections still active, continuing to wait", "active_count", count)

			if time.Now().After(deadline) {
				slog.Warn("Connection drain timeout exceeded", "active_count", count, "timeout", timeout)
				return ErrDrainTimeout
			}
		}
	}
}

func (m *Monitor) waitForSteadyState(deadline time.Time) error {
	steadyStateCheckInterval := 50 * time.Millisecond
	slog.Info("Connections reached zero, starting steady state wait", "wait_duration", m.steadyStateWait, "check_interval", steadyStateCheckInterval)
	steadyStateDeadline := time.Now().Add(m.steadyStateWait)

	ticker := time.NewTicker(steadyStateCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			isAfterOverallDeadline := time.Now().After(deadline)

			if isAfterOverallDeadline {
				slog.Warn("Overall drain timeout exceeded during steady state wait")
				return ErrDrainTimeout
			}

			count, err := m.CountActiveConnections()
			if err != nil {
				slog.Error("Error counting active connections during steady state wait", "error", err)
				return err
			}

			if count > 0 {
				slog.Info("Connections increased during steady state wait, resetting timer", "active_count", count)
				return m.WaitForZeroConnections(time.Until(deadline))
			}

			isAfterSteadyStateDeadline := time.Now().After(steadyStateDeadline)

			if isAfterSteadyStateDeadline {
				slog.Info("Steady state wait completed, all connections drained successfully")
				return nil
			}

			slog.Debug("Steady state wait in progress, connections still at zero")
		}
	}
}

func (m *Monitor) isMonitoredPort(port uint16) bool {
	for _, p := range m.ports {
		if p == port {
			return true
		}
	}
	return false
}

func (m *Monitor) isActiveState(state TCPState) bool {
	switch state {
	case StateEstablished:
		return true
	case StateSynSent:
		return true
	case StateSynRecv:
		return true
	case StateFinWait1:
		return true
	case StateFinWait2:
		return true
	case StateCloseWait:
		return true
	case StateClosing:
		return true
	case StateLastAck:
		return true
	default:
		return false
	}
}
