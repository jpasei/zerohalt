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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jpasei/zerohalt/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestNewAppHealthChecker(t *testing.T) {
	healthURL := "http://localhost:8080/health"
	timeout := 5 * time.Second

	checker := NewAppHealthChecker(healthURL, timeout)

	assert.NotNil(t, checker)
	assert.Equal(t, healthURL, checker.healthURL)
	assert.Equal(t, timeout, checker.timeout)
}

func TestAppHealthChecker_Check(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody string
		serverDelay  time.Duration
		timeout      time.Duration
		want         bool
	}{
		{"healthy status 200", http.StatusOK, `{"status":"ok"}`, 0, 5 * time.Second, true},
		{"healthy status 204", http.StatusNoContent, "", 0, 5 * time.Second, true},
		{"unhealthy status 500", http.StatusInternalServerError, `{"status":"error"}`, 0, 5 * time.Second, false},
		{"unhealthy status 503", http.StatusServiceUnavailable, `{"status":"starting"}`, 0, 5 * time.Second, false},
		{"timeout", http.StatusOK, `{"status":"ok"}`, 200 * time.Millisecond, 100 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.serverDelay > 0 {
					time.Sleep(tt.serverDelay)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			checker := NewAppHealthChecker(server.URL, tt.timeout)
			got := checker.Check()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppHealthChecker_CheckWithInvalidURL(t *testing.T) {
	checker := NewAppHealthChecker("http://invalid-host-that-does-not-exist:9999/health", 1*time.Second)
	got := checker.Check()
	assert.False(t, got)
}

func TestAppHealthChecker_CheckWithEmptyURL(t *testing.T) {
	checker := NewAppHealthChecker("", 1*time.Second)
	got := checker.Check()
	assert.False(t, got)
}

func TestAppHealthChecker_CheckWithMalformedURL(t *testing.T) {
	checker := NewAppHealthChecker("http://example.com\x00/health", 1*time.Second)
	got := checker.Check()
	assert.False(t, got)
}

func TestAppHealthChecker_WaitForHealthy(t *testing.T) {
	tests := []struct {
		name           string
		healthyAfter   time.Duration
		startupTimeout time.Duration
		checkInterval  time.Duration
		want           bool
	}{
		{"becomes healthy immediately", 0, 5 * time.Second, 100 * time.Millisecond, true},
		{"becomes healthy after 300ms", 300 * time.Millisecond, 2 * time.Second, 100 * time.Millisecond, true},
		{"never becomes healthy", 10 * time.Second, 500 * time.Millisecond, 100 * time.Millisecond, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startTime := time.Now()
			healthy := false

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if time.Since(startTime) >= tt.healthyAfter {
					healthy = true
				}

				if healthy {
					w.WriteHeader(http.StatusOK)
				} else {
					w.WriteHeader(http.StatusServiceUnavailable)
				}
			}))
			defer server.Close()

			checker := NewAppHealthChecker(server.URL, 1*time.Second)
			got := checker.WaitForHealthy(tt.startupTimeout, tt.checkInterval)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppHealthChecker_WaitForHealthyWithInvalidURL(t *testing.T) {
	checker := NewAppHealthChecker("http://invalid-host:9999/health", 1*time.Second)
	got := checker.WaitForHealthy(1*time.Second, 100*time.Millisecond)
	assert.False(t, got)
}

func TestAppHealthChecker_Check_SetsMetricsWhenHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	checker := NewAppHealthChecker(server.URL, 5*time.Second)
	result := checker.Check()

	assert.True(t, result)
	assert.Equal(t, float64(1), testutil.ToFloat64(metrics.HealthApp))
}

func TestAppHealthChecker_Check_SetsMetricsWhenUnhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unhealthy"}`))
	}))
	defer server.Close()

	checker := NewAppHealthChecker(server.URL, 5*time.Second)
	result := checker.Check()

	assert.False(t, result)
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.HealthApp))
}

func TestAppHealthChecker_Check_SetsMetricsToZeroOnError(t *testing.T) {
	checker := NewAppHealthChecker("http://invalid-host-that-does-not-exist:9999/health", 1*time.Second)
	result := checker.Check()

	assert.False(t, result)
	assert.Equal(t, float64(0), testutil.ToFloat64(metrics.HealthApp))
}
