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
		if sig := ParseSignal(sigName); sig != nil {
			h.passThroughSignals[sig] = true
		}
	}

	for _, sigName := range config.ShutdownSignals {
		if sig := ParseSignal(sigName); sig != nil {
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
		h.forwardSignalToApp(sig)
		return ActionPassThrough

	case sig == syscall.SIGCHLD:
		return ActionReapZombies

	default:
		slog.Warn("Received unexpected signal", "signal", sig.String())
		return ActionIgnore
	}
}

func (h *SignalHandler) forwardSignalToApp(sig os.Signal) {
	err := h.appProcess.Signal(sig)

	if err != nil {
		slog.Error("Failed to forward signal to app", "signal", sig.String(), "error", err)
		return
	}

	metrics.SignalsForwarded.WithLabelValues(sig.String()).Inc()
	slog.Info("Forwarded signal to application", "signal", sig.String(), "pid", h.appProcess.Pid)
}

var signalMap = map[string]os.Signal{
	"SIGHUP":   syscall.SIGHUP,
	"SIGINT":   syscall.SIGINT,
	"SIGTERM":  syscall.SIGTERM,
	"SIGUSR1":  syscall.SIGUSR1,
	"SIGUSR2":  syscall.SIGUSR2,
	"SIGWINCH": syscall.SIGWINCH,
	"SIGQUIT":  syscall.SIGQUIT,
}

func ParseSignal(name string) os.Signal {
	return signalMap[name]
}
