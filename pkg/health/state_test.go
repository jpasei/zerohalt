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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthState_String(t *testing.T) {
	tests := []struct {
		name  string
		state HealthState
		want  string
	}{
		{"StateStarting", StateStarting, "starting"},
		{"StateHealthy", StateHealthy, "healthy"},
		{"StateDraining", StateDraining, "draining"},
		{"StateTerminating", StateTerminating, "terminating"},
		{"StateUnknown", HealthState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewState(t *testing.T) {
	s := NewState()

	assert.NotNil(t, s)
	assert.Equal(t, StateStarting, s.current)
}

func TestState_Set(t *testing.T) {
	s := NewState()

	s.Set(StateHealthy)
	assert.Equal(t, StateHealthy, s.current)

	s.Set(StateDraining)
	assert.Equal(t, StateDraining, s.current)
}

func TestState_Get(t *testing.T) {
	s := NewState()

	got := s.Get()
	assert.Equal(t, StateStarting, got)

	s.Set(StateHealthy)
	got = s.Get()
	assert.Equal(t, StateHealthy, got)
}

func TestState_ConcurrentAccess(t *testing.T) {
	s := NewState()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			s.Set(StateHealthy)
		}()

		go func() {
			defer wg.Done()
			_ = s.Get()
		}()
	}

	wg.Wait()
}
