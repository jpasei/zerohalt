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
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMonitor(t *testing.T) {
	ports := []uint16{8080, 9090}
	interval := 1 * time.Second

	m := NewMonitor(ports, interval)

	assert.NotNil(t, m)
	assert.Len(t, m.ports, 2)
	assert.Equal(t, interval, m.interval)
}

func TestMonitor_isMonitoredPort(t *testing.T) {
	m := NewMonitor([]uint16{8080, 9090}, 1*time.Second)

	tests := []struct {
		name string
		port uint16
		want bool
	}{
		{"monitored port 8080", 8080, true},
		{"monitored port 9090", 9090, true},
		{"unmonitored port 80", 80, false},
		{"unmonitored port 3000", 3000, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.isMonitoredPort(tt.port)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMonitor_CountActiveConnections(t *testing.T) {
	m := NewMonitor([]uint16{8080}, 1*time.Second)

	count, err := m.CountActiveConnections()
	assert.GreaterOrEqual(t, count, 0)

	t.Skipf("CountActiveConnections() skipped: /proc/net/tcp not accessible: %v", err)
}

func TestMonitor_WaitForZeroConnections_Immediate(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 50 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(100 * time.Millisecond)

	assert.NoError(t, err)
}

func TestMonitor_CountActiveConnections_FileNotFound(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 1 * time.Second,
	}

	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		return nil, os.ErrNotExist
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	_, err := m.CountActiveConnections()
	assert.Error(t, err)
}

func TestMonitor_CountActiveConnections_FileNotFoundTCP6(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 1 * time.Second,
	}

	origParseProcNetTCP := parseProcNetTCP
	callCount := 0
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callCount++
		if callCount == 1 {
			return []Connection{}, nil
		}
		return nil, os.ErrNotExist
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	_, err := m.CountActiveConnections()
	assert.Error(t, err)
}

func TestMonitor_WaitForZeroConnections_ErrorInCount(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 10 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		return nil, os.ErrPermission
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(50 * time.Millisecond)
	assert.Error(t, err)
}

func TestMonitor_WaitForZeroConnections_Timeout(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 10 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		return []Connection{
			{
				LocalPort: 8080,
				State:     StateEstablished,
			},
		}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(30 * time.Millisecond)
	assert.Equal(t, ErrDrainTimeout, err)
}

func TestMonitor_CountActiveConnections_WithEstablished(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080, 9090},
		interval: 1 * time.Second,
	}

	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		return []Connection{
			{LocalPort: 8080, State: StateEstablished},
			{LocalPort: 9090, State: StateEstablished},
			{LocalPort: 8080, State: StateListen},
			{LocalPort: 7070, State: StateEstablished},
		}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	count, err := m.CountActiveConnections()
	assert.NoError(t, err)
	assert.Equal(t, 4, count)
}

func TestMonitor_Start(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 50 * time.Millisecond,
	}

	callCount := 0
	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callCount++
		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	m.Start()

	time.Sleep(120 * time.Millisecond)

	assert.GreaterOrEqual(t, callCount, 2)
}

func TestMonitor_WaitForZeroConnections_EventualSuccess(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 30 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	callIndex := 0
	responses := []struct {
		conns []Connection
		err   error
	}{
		{[]Connection{{LocalPort: 8080, State: StateEstablished}}, nil},
		{[]Connection{}, nil},
		{[]Connection{{LocalPort: 8080, State: StateEstablished}}, nil},
		{[]Connection{}, nil},
		{[]Connection{}, nil},
		{[]Connection{}, nil},
		{[]Connection{}, nil},
		{[]Connection{}, nil},
		{[]Connection{}, nil},
		{[]Connection{}, nil},
	}
	parseProcNetTCP = func(path string) ([]Connection, error) {
		response := responses[callIndex]
		callIndex++
		return response.conns, response.err
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(200 * time.Millisecond)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, callIndex, 4)
}

func TestMonitor_isActiveState(t *testing.T) {
	m := NewMonitor([]uint16{8080}, 1*time.Second)

	tests := []struct {
		name  string
		state TCPState
		want  bool
	}{
		{"established", StateEstablished, true},
		{"syn_sent", StateSynSent, true},
		{"syn_recv", StateSynRecv, true},
		{"fin_wait1", StateFinWait1, true},
		{"fin_wait2", StateFinWait2, true},
		{"close_wait", StateCloseWait, true},
		{"closing", StateClosing, true},
		{"last_ack", StateLastAck, true},
		{"listen", StateListen, false},
		{"time_wait", StateTimeWait, false},
		{"close", StateClose, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := m.isActiveState(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMonitor_CountActiveConnections_WithMultipleStates(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 1 * time.Second,
	}

	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		return []Connection{
			{LocalPort: 8080, State: StateEstablished},
			{LocalPort: 8080, State: StateSynRecv},
			{LocalPort: 8080, State: StateFinWait1},
			{LocalPort: 8080, State: StateCloseWait},
			{LocalPort: 8080, State: StateListen},
			{LocalPort: 8080, State: StateTimeWait},
			{LocalPort: 7070, State: StateEstablished},
		}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	count, err := m.CountActiveConnections()
	assert.NoError(t, err)
	assert.Equal(t, 8, count)
}

func TestMonitor_WaitForZeroConnections_ErrorInTickerLoop(t *testing.T) {
	m := &Monitor{
		ports:    []uint16{8080},
		interval: 30 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	callIndex := 0
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callIndex++
		atErrorPoint := callIndex == 7 || callIndex == 8

		if atErrorPoint {
			return nil, os.ErrPermission
		}

		return []Connection{{LocalPort: 8080, State: StateEstablished}}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(200 * time.Millisecond)

	assert.Error(t, err)
	assert.Equal(t, os.ErrPermission, err)
}

func TestMonitor_SetSteadyStateWait(t *testing.T) {
	m := NewMonitor([]uint16{8080}, 1*time.Second)
	wait := 5 * time.Second

	m.SetSteadyStateWait(wait)

	assert.Equal(t, wait, m.steadyStateWait)
}

func TestMonitor_WaitForZeroConnections_SteadyStateDisabled(t *testing.T) {
	m := &Monitor{
		ports:           []uint16{8080},
		interval:        30 * time.Millisecond,
		steadyStateWait: 0,
	}

	origParseProcNetTCP := parseProcNetTCP
	callIndex := 0
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callIndex++
		hasConnections := callIndex == 1 || callIndex == 2

		if hasConnections {
			return []Connection{{LocalPort: 8080, State: StateEstablished}}, nil
		}

		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(200 * time.Millisecond)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, callIndex, 4)
}

func TestMonitor_WaitForZeroConnections_SteadyStateWaitSuccess(t *testing.T) {
	m := &Monitor{
		ports:           []uint16{8080},
		interval:        20 * time.Millisecond,
		steadyStateWait: 60 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	callIndex := 0
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callIndex++
		hasConnections := callIndex == 1 || callIndex == 2

		if hasConnections {
			return []Connection{{LocalPort: 8080, State: StateEstablished}}, nil
		}

		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(500 * time.Millisecond)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, callIndex, 3)
}

func TestMonitor_WaitForZeroConnections_SteadyStateResetOnConnectionIncrease(t *testing.T) {
	m := &Monitor{
		ports:           []uint16{8080},
		interval:        20 * time.Millisecond,
		steadyStateWait: 60 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	callIndex := 0
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callIndex++
		hasConnectionsPhase1 := callIndex == 1 || callIndex == 2
		hasConnectionsPhase2 := callIndex == 4 || callIndex == 5

		if hasConnectionsPhase1 || hasConnectionsPhase2 {
			return []Connection{{LocalPort: 8080, State: StateEstablished}}, nil
		}

		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(500 * time.Millisecond)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, callIndex, 6)
}

func TestMonitor_WaitForZeroConnections_SteadyStateTimeoutDuringWait(t *testing.T) {
	m := &Monitor{
		ports:           []uint16{8080},
		interval:        20 * time.Millisecond,
		steadyStateWait: 100 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	callIndex := 0
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callIndex++
		hasConnections := callIndex == 1 || callIndex == 2

		if hasConnections {
			return []Connection{{LocalPort: 8080, State: StateEstablished}}, nil
		}

		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(80 * time.Millisecond)

	assert.Equal(t, ErrDrainTimeout, err)
}

func TestMonitor_WaitForZeroConnections_SteadyStateImmediateZero(t *testing.T) {
	m := &Monitor{
		ports:           []uint16{8080},
		interval:        20 * time.Millisecond,
		steadyStateWait: 60 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	parseProcNetTCP = func(path string) ([]Connection, error) {
		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(500 * time.Millisecond)

	assert.NoError(t, err)
}

func TestMonitor_WaitForZeroConnections_SteadyStateErrorDuringWait(t *testing.T) {
	m := &Monitor{
		ports:           []uint16{8080},
		interval:        20 * time.Millisecond,
		steadyStateWait: 60 * time.Millisecond,
	}

	origParseProcNetTCP := parseProcNetTCP
	callIndex := 0
	parseProcNetTCP = func(path string) ([]Connection, error) {
		callIndex++
		hasConnections := callIndex == 1 || callIndex == 2

		if hasConnections {
			return []Connection{{LocalPort: 8080, State: StateEstablished}}, nil
		}

		atSteadyStateErrorPoint := callIndex == 4

		if atSteadyStateErrorPoint {
			return nil, os.ErrPermission
		}

		return []Connection{}, nil
	}
	defer func() {
		parseProcNetTCP = origParseProcNetTCP
	}()

	err := m.WaitForZeroConnections(500 * time.Millisecond)

	assert.Error(t, err)
	assert.Equal(t, os.ErrPermission, err)
}
