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

package metrics

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetrics_StateInitialized(t *testing.T) {
	assert.NotNil(t, State)
}

func TestMetrics_UptimeInitialized(t *testing.T) {
	assert.NotNil(t, Uptime)
}

func TestMetrics_AppUptimeInitialized(t *testing.T) {
	assert.NotNil(t, AppUptime)
}

func TestMetrics_ActiveConnectionsInitialized(t *testing.T) {
	assert.NotNil(t, ActiveConnections)
}

func TestMetrics_DrainPhaseActiveInitialized(t *testing.T) {
	assert.NotNil(t, DrainPhaseActive)
}

func TestMetrics_DrainDurationInitialized(t *testing.T) {
	assert.NotNil(t, DrainDuration)
}

func TestMetrics_HealthRequestsInitialized(t *testing.T) {
	assert.NotNil(t, HealthRequests)
}

func TestMetrics_HealthRequestDurationInitialized(t *testing.T) {
	assert.NotNil(t, HealthRequestDuration)
}

func TestMetrics_HealthAppInitialized(t *testing.T) {
	assert.NotNil(t, HealthApp)
}

func TestMetrics_SignalsReceivedInitialized(t *testing.T) {
	assert.NotNil(t, SignalsReceived)
}

func TestMetrics_SignalsForwardedInitialized(t *testing.T) {
	assert.NotNil(t, SignalsForwarded)
}

func TestMetrics_StateGauge(t *testing.T) {
	State.Set(2)

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_state 2")
}

func TestMetrics_UptimeCounter(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_uptime_seconds")
}

func TestMetrics_AppUptimeCounter(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_app_uptime_seconds")
}

func TestMetrics_ActiveConnectionsGauge(t *testing.T) {
	ActiveConnections.Set(42)

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_active_connections 42")
}

func TestMetrics_DrainPhaseActiveGauge(t *testing.T) {
	DrainPhaseActive.Set(1)

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_drain_phase_active 1")
}

func TestMetrics_DrainDurationGauge(t *testing.T) {
	DrainDuration.Set(30.5)

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_drain_duration_seconds 30.5")
}

func TestMetrics_HealthRequestsCounter(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_health_requests_total")
}

func TestMetrics_HealthRequestDurationGauge(t *testing.T) {
	HealthRequestDuration.Set(1.2)

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_health_request_duration_ms 1.2")
}

func TestMetrics_HealthAppGauge(t *testing.T) {
	HealthApp.Set(1)

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_health_app 1")
}

func TestMetrics_SignalsReceivedCounter(t *testing.T) {
	SignalsReceived.WithLabelValues("SIGTERM").Inc()

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_signals_received_total")
	assert.Contains(t, string(body), "SIGTERM")
}

func TestMetrics_SignalsForwardedCounter(t *testing.T) {
	SignalsForwarded.WithLabelValues("SIGUSR1").Inc()

	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_signals_forwarded_total")
	assert.Contains(t, string(body), "SIGUSR1")
}

func TestHandler_ReturnsStatusOK(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Result().StatusCode)
}

func TestHandler_ContainsStateMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_state")
}

func TestHandler_ContainsUptimeMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_uptime_seconds")
}

func TestHandler_ContainsAppUptimeMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_app_uptime_seconds")
}

func TestHandler_ContainsActiveConnectionsMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_active_connections")
}

func TestHandler_ContainsDrainPhaseActiveMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_drain_phase_active")
}

func TestHandler_ContainsDrainDurationMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_drain_duration_seconds")
}

func TestHandler_ContainsHealthRequestsMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_health_requests_total")
}

func TestHandler_ContainsHealthRequestDurationMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_health_request_duration_ms")
}

func TestHandler_ContainsHealthAppMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_health_app")
}

func TestHandler_ContainsSignalsReceivedMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_signals_received_total")
}

func TestHandler_ContainsSignalsForwardedMetric(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), "zerohalt_signals_forwarded_total")
}

func TestHandler_NoHelpComments(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.NotContains(t, string(body), "# HELP")
}

func TestHandler_NoTypeComments(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.NotContains(t, string(body), "# TYPE")
}

func TestHandler_ContentType(t *testing.T) {
	handler := Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Contains(t, w.Result().Header.Get("Content-Type"), "text/plain")
}

func TestResponseRecorder_Write(t *testing.T) {
	rec := &responseRecorder{
		ResponseWriter: httptest.NewRecorder(),
		body:          &bytes.Buffer{},
	}

	data := []byte("test data")
	n, err := rec.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, "test data", rec.body.String())
}
