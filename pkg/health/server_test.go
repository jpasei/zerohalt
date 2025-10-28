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
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")

	assert.NotNil(t, s)
	assert.Equal(t, port, s.port)
	assert.Equal(t, "/health", s.path)
	assert.NotNil(t, s.state)
}

func TestServer_SetState(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")

	s.SetState(StateHealthy)
	assert.Equal(t, StateHealthy, s.GetState())

	s.SetState(StateDraining)
	assert.Equal(t, StateDraining, s.GetState())
}

func TestServer_healthHandler_Healthy(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")
	s.SetState(StateHealthy)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"status":"healthy"}`, w.Body.String())
}

func TestServer_healthHandler_Starting(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")
	s.SetState(StateStarting)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"starting"}`, w.Body.String())
}

func TestServer_healthHandler_Draining(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")
	s.SetState(StateDraining)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"draining"}`, w.Body.String())
}

func TestServer_healthHandler_Terminating(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")
	s.SetState(StateTerminating)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"terminating"}`, w.Body.String())
}

func TestServer_healthHandler_Unknown(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")
	s.state.Set(HealthState(99))

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `{"status":"unknown"}`, w.Body.String())
}

func TestServer_Start_Shutdown(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")

	err := s.Start()
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestServer_Start_ErrorCase(t *testing.T) {
	port := getAvailablePort()
	s1 := NewServer(port, "/health")
	s2 := NewServer(port, "/health")

	err := s1.Start()
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	err = s2.Start()
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s1.Shutdown(ctx)
	s2.Shutdown(ctx)
}
