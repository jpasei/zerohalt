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
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jpasei/zerohalt/pkg/metrics"
)

type SignalAction int

const (
	ActionIgnore SignalAction = iota
	ActionPassThrough
	ActionShutdown
	ActionReapZombies
)

type SignalConfig struct {
	PassThroughSignals []string
	ShutdownSignals    []string
}

type SignalHandler struct {
	passThroughSignals map[os.Signal]bool
	shutdownSignals    map[os.Signal]bool
	appProcess         *os.Process
}

func NewSignalHandler(config *SignalConfig, appProcess *os.Process) *SignalHandler {
	h := &SignalHandler{
		passThroughSignals: make(map[os.Signal]bool),
		shutdownSignals:    make(map[os.Signal]bool),
		appProcess:         appProcess,
	}

	for _, sigName := range config.PassThroughSignals {
		if sig := parseSignal(sigName); sig != nil {
			h.passThroughSignals[sig] = true
		}
	}

	for _, sigName := range config.ShutdownSignals {
		if sig := parseSignal(sigName); sig != nil {
			h.shutdownSignals[sig] = true
		}
	}

	return h
}

func (h *SignalHandler) Setup() chan os.Signal {
	shutdownChan := make(chan os.Signal, 1)

	allSignals := make([]os.Signal, 0)

	for sig := range h.passThroughSignals {
		allSignals = append(allSignals, sig)
	}

	for sig := range h.shutdownSignals {
		allSignals = append(allSignals, sig)
	}

	allSignals = append(allSignals, syscall.SIGCHLD)

	signal.Notify(shutdownChan, allSignals...)

	return shutdownChan
}

func (h *SignalHandler) Handle(sig os.Signal) SignalAction {
	metrics.SignalsReceived.WithLabelValues(sig.String()).Inc()

	switch {
	case h.shutdownSignals[sig]:
		return ActionShutdown

	case h.passThroughSignals[sig]:
		if err := h.appProcess.Signal(sig); err != nil {
			slog.Error("Failed to forward signal to app", "signal", sig.String(), "error", err)
		} else {
			metrics.SignalsForwarded.WithLabelValues(sig.String()).Inc()
			slog.Info("Forwarded signal to application", "signal", sig.String(), "pid", h.appProcess.Pid)
		}
		return ActionPassThrough

	case sig == syscall.SIGCHLD:
		return ActionReapZombies

	default:
		slog.Warn("Received unexpected signal", "signal", sig.String())
		return ActionIgnore
	}
}

func parseSignal(name string) os.Signal {
	switch name {
	case "SIGHUP":
		return syscall.SIGHUP
	case "SIGINT":
		return syscall.SIGINT
	case "SIGTERM":
		return syscall.SIGTERM
	case "SIGUSR1":
		return syscall.SIGUSR1
	case "SIGUSR2":
		return syscall.SIGUSR2
	case "SIGWINCH":
		return syscall.SIGWINCH
	case "SIGQUIT":
		return syscall.SIGQUIT
	default:
		return nil
	}
}
