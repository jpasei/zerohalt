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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCommandHealthChecker(t *testing.T) {
	cmd := []string{"echo", "test"}
	timeout := 5 * time.Second

	checker := NewCommandHealthChecker(cmd, timeout)

	assert.NotNil(t, checker)
	assert.Len(t, checker.command, 2)
	assert.Equal(t, timeout, checker.timeout)
}

func TestCommandHealthChecker_Check(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		timeout time.Duration
		want    bool
	}{
		{"no command", []string{}, 5 * time.Second, false},
		{"true command", []string{"true"}, 5 * time.Second, true},
		{"false command", []string{"false"}, 5 * time.Second, false},
		{"timeout command", []string{"sleep", "10"}, 100 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCommandHealthChecker(tt.command, tt.timeout)
			got := checker.Check()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCommandHealthChecker_CheckWithDetails(t *testing.T) {
	tests := []struct {
		name         string
		command      []string
		timeout      time.Duration
		wantHealthy  bool
		wantExitCode int
		wantErr      bool
	}{
		{"no command", []string{}, 5 * time.Second, false, -1, true},
		{"true command", []string{"true"}, 5 * time.Second, true, 0, false},
		{"false command", []string{"false"}, 5 * time.Second, false, 1, true},
		{"invalid command", []string{"/nonexistent/command"}, 5 * time.Second, false, -1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCommandHealthChecker(tt.command, tt.timeout)
			gotHealthy, gotExitCode, gotErr := checker.CheckWithDetails()

			assert.Equal(t, tt.wantHealthy, gotHealthy)
			assert.Equal(t, tt.wantExitCode, gotExitCode)
			assert.Equal(t, tt.wantErr, gotErr != nil)
		})
	}
}
