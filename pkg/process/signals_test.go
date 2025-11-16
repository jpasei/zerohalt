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

package process

import (
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignalAction_Constants(t *testing.T) {
	tests := []struct {
		name   string
		action SignalAction
		value  SignalAction
	}{
		{"ActionIgnore", ActionIgnore, 0},
		{"ActionPassThrough", ActionPassThrough, 1},
		{"ActionShutdown", ActionShutdown, 2},
		{"ActionReapZombies", ActionReapZombies, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.value, tt.action)
		})
	}
}

func TestParseSignal_AllSignals(t *testing.T) {
	tests := []struct {
		name string
		sig  string
		want os.Signal
	}{
		{"SIGHUP", "SIGHUP", syscall.SIGHUP},
		{"SIGINT", "SIGINT", syscall.SIGINT},
		{"SIGTERM", "SIGTERM", syscall.SIGTERM},
		{"SIGUSR1", "SIGUSR1", syscall.SIGUSR1},
		{"SIGUSR2", "SIGUSR2", syscall.SIGUSR2},
		{"SIGWINCH", "SIGWINCH", syscall.SIGWINCH},
		{"SIGQUIT", "SIGQUIT", syscall.SIGQUIT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSignal(tt.sig)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSignal_Invalid(t *testing.T) {
	tests := []string{"INVALID", "SIGKILL", "", "random"}

	for _, sig := range tests {
		t.Run(sig, func(t *testing.T) {
			got := ParseSignal(sig)
			assert.Nil(t, got)
		})
	}
}

func TestNewSignalHandler(t *testing.T) {
	config := &SignalConfig{
		PassThroughSignals: []string{"SIGHUP"},
		ShutdownSignals:    []string{"SIGTERM", "SIGINT"},
	}

	handler := NewSignalHandler(config, &os.Process{Pid: 123})

	assert.NotNil(t, handler)
	assert.Len(t, handler.passThroughSignals, 1)
	assert.Len(t, handler.shutdownSignals, 2)
}

func TestSignalHandler_Setup(t *testing.T) {
	config := &SignalConfig{
		PassThroughSignals: []string{"SIGHUP"},
		ShutdownSignals:    []string{"SIGTERM"},
	}

	handler := NewSignalHandler(config, &os.Process{Pid: 123})
	sigChan := handler.Setup()

	assert.NotNil(t, sigChan)
}

func TestSignalHandler_Handle_Shutdown(t *testing.T) {
	config := &SignalConfig{
		ShutdownSignals: []string{"SIGTERM"},
	}

	handler := NewSignalHandler(config, &os.Process{Pid: 123})
	action := handler.Handle(syscall.SIGTERM)

	assert.Equal(t, ActionShutdown, action)
}

func TestSignalHandler_Handle_ReapZombies(t *testing.T) {
	config := &SignalConfig{}
	handler := NewSignalHandler(config, &os.Process{Pid: 123})

	action := handler.Handle(syscall.SIGCHLD)

	assert.Equal(t, ActionReapZombies, action)
}

func TestSignalHandler_Handle_Ignore(t *testing.T) {
	config := &SignalConfig{}
	handler := NewSignalHandler(config, &os.Process{Pid: 123})

	action := handler.Handle(syscall.SIGUSR1)

	assert.Equal(t, ActionIgnore, action)
}

func TestSignalHandler_Handle_PassThrough_Success(t *testing.T) {
	config := &SignalConfig{
		PassThroughSignals: []string{"SIGHUP"},
	}

	currentProc, err := os.FindProcess(os.Getpid())
	assert.NoError(t, err)

	handler := NewSignalHandler(config, currentProc)
	action := handler.Handle(syscall.SIGHUP)

	assert.Equal(t, ActionPassThrough, action)
}

func TestSignalHandler_Handle_PassThrough_Error(t *testing.T) {
	config := &SignalConfig{
		PassThroughSignals: []string{"SIGHUP"},
	}

	invalidProc := &os.Process{Pid: 999999}

	handler := NewSignalHandler(config, invalidProc)
	action := handler.Handle(syscall.SIGHUP)

	assert.Equal(t, ActionPassThrough, action)
}
