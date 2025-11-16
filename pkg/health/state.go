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

package health

import (
	"log/slog"
	"sync"

	"github.com/jpasei/zerohalt/pkg/metrics"
)

type HealthState int

const (
	StateStarting HealthState = iota
	StateHealthy
	StateUnhealthy
	StateDraining
	StateTerminating
)

func (s HealthState) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateHealthy:
		return "healthy"
	case StateUnhealthy:
		return "unhealthy"
	case StateDraining:
		return "draining"
	case StateTerminating:
		return "terminating"
	default:
		return "unknown"
	}
}

type State struct {
	current HealthState
	mu      sync.RWMutex
}

func NewState() *State {
	metrics.State.Set(float64(StateStarting))
	return &State{
		current: StateStarting,
	}
}

func (s *State) Set(state HealthState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Debug("State transition requested", "from", s.current.String(), "to", state.String())

	currentIsTerminating := s.current == StateTerminating
	if currentIsTerminating {
		slog.Debug("Blocked: cannot transition from Terminating")
		return
	}

	currentIsDraining := s.current == StateDraining
	targetIsTerminating := state == StateTerminating
	allowedDrainingTransition := currentIsDraining && targetIsTerminating

	if currentIsDraining && !allowedDrainingTransition {
		slog.Debug("Blocked: cannot transition from Draining except to Terminating")
		return
	}

	s.current = state
	metrics.State.Set(float64(state))
	slog.Debug("State transition successful", "new_state", state.String())
}

func (s *State) Get() HealthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}
