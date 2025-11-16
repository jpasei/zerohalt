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
	"fmt"
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

func TestServer_healthHandler_Unhealthy(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")
	s.SetState(StateUnhealthy)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"unhealthy"}`, w.Body.String())
}

func TestServer_healthHandler_Unhealthy_BecomesHealthy(t *testing.T) {
	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer appServer.Close()

	port := getAvailablePort()
	appChecker := NewAppHealthChecker(appServer.URL, 5*time.Second)
	s := NewServerWithAppChecker(port, "/health", appChecker)
	s.SetState(StateUnhealthy)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"status":"healthy"}`, w.Body.String())
	assert.Equal(t, StateHealthy, s.GetState(), "State should transition from Unhealthy to Healthy when app becomes healthy")
}

func TestServer_healthHandler_Unhealthy_RemainsUnhealthy(t *testing.T) {
	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unavailable"}`))
	}))
	defer appServer.Close()

	port := getAvailablePort()
	appChecker := NewAppHealthChecker(appServer.URL, 5*time.Second)
	s := NewServerWithAppChecker(port, "/health", appChecker)
	s.SetState(StateUnhealthy)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"unhealthy"}`, w.Body.String())
	assert.Equal(t, StateUnhealthy, s.GetState(), "State should remain Unhealthy when app is still unhealthy")
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

func TestNewServerWithAppHealthChecker(t *testing.T) {
	port := getAvailablePort()
	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer appServer.Close()

	appChecker := NewAppHealthChecker(appServer.URL, 1*time.Second)
	s := NewServerWithAppChecker(port, "/health", appChecker)

	assert.NotNil(t, s)
	assert.Equal(t, port, s.port)
	assert.Equal(t, "/health", s.path)
	assert.NotNil(t, s.appChecker)
}

func TestServer_healthHandler_AppDependent_AppHealthy(t *testing.T) {
	port := getAvailablePort()

	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer appServer.Close()

	appChecker := NewAppHealthChecker(appServer.URL, 1*time.Second)
	s := NewServerWithAppChecker(port, "/health", appChecker)
	s.SetState(StateHealthy)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"status":"healthy"}`, w.Body.String())
}

func TestServer_healthHandler_AppDependent_AppUnhealthy(t *testing.T) {
	port := getAvailablePort()

	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer appServer.Close()

	appChecker := NewAppHealthChecker(appServer.URL, 1*time.Second)
	s := NewServerWithAppChecker(port, "/health", appChecker)
	s.SetState(StateHealthy)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"unhealthy"}`, w.Body.String())
}

func TestServer_healthHandler_AppDependent_StateStarting(t *testing.T) {
	port := getAvailablePort()

	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer appServer.Close()

	appChecker := NewAppHealthChecker(appServer.URL, 1*time.Second)
	s := NewServerWithAppChecker(port, "/health", appChecker)
	s.SetState(StateStarting)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"starting"}`, w.Body.String())
}

func TestServer_healthHandler_AppDependent_StateDraining(t *testing.T) {
	port := getAvailablePort()

	appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer appServer.Close()

	appChecker := NewAppHealthChecker(appServer.URL, 1*time.Second)
	s := NewServerWithAppChecker(port, "/health", appChecker)
	s.SetState(StateDraining)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	s.healthHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, `{"status":"draining"}`, w.Body.String())
}

func TestServer_WaitForAppHealthy(t *testing.T) {
	tests := []struct {
		name            string
		appHealthyAfter time.Duration
		startupTimeout  time.Duration
		checkInterval   time.Duration
		want            bool
	}{
		{"app becomes healthy immediately", 0, 2 * time.Second, 100 * time.Millisecond, true},
		{"app becomes healthy after 300ms", 300 * time.Millisecond, 2 * time.Second, 100 * time.Millisecond, true},
		{"app never becomes healthy", 10 * time.Second, 500 * time.Millisecond, 100 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := getAvailablePort()
			startTime := time.Now()
			appHealthy := false

			appServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if time.Since(startTime) >= tt.appHealthyAfter {
					appHealthy = true
				}

				if appHealthy {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			}))
			defer appServer.Close()

			appChecker := NewAppHealthChecker(appServer.URL, 1*time.Second)
			s := NewServerWithAppChecker(port, "/health", appChecker)

			got := s.WaitForAppHealthy(tt.startupTimeout, tt.checkInterval)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestServer_WaitForAppHealthy_NoAppChecker(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")

	got := s.WaitForAppHealthy(1*time.Second, 100*time.Millisecond)
	assert.True(t, got)
}

func TestServer_EnableMetrics(t *testing.T) {
	port := getAvailablePort()
	s := NewServer(port, "/health")

	s.EnableMetrics("/metrics")

	err := s.Start()
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = s.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestServer_AppDependentMode_RealIntegration_AppDown(t *testing.T) {
	port := getAvailablePort()

	appChecker := NewAppHealthChecker("http://localhost:9999/nonexistent", 500*time.Millisecond)
	s := NewServerWithAppChecker(port, "/health", appChecker)

	err := s.Start()
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	s.SetState(StateHealthy)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", port))
	assert.NoError(t, err)
	defer resp.Body.Close()

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	t.Logf("Response status: %d, body: %s", resp.StatusCode, string(body[:n]))

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode, "Health check should return 503 when app is down even if state is Healthy")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.Shutdown(ctx)
}
