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
	"sync"

	"github.com/jpasei/zerohalt/pkg/metrics"
)

type HealthState int

const (
	StateStarting HealthState = iota
	StateHealthy
	StateDraining
	StateTerminating
)

func (s HealthState) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateHealthy:
		return "healthy"
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
	return &State{
		current: StateStarting,
	}
}

func (s *State) Set(state HealthState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = state
	metrics.State.Set(float64(state))
}

func (s *State) Get() HealthState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}
