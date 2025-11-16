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
	"bufio"
	"bytes"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Custom registry - only our metrics, no Go runtime metrics
	registry = prometheus.NewRegistry()

	// Process Manager Metrics
	State = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "zerohalt_state",
		Help: "Current state (0=starting, 1=healthy, 2=draining, 3=terminating)",
	})

	Uptime = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "zerohalt_uptime_seconds",
		Help: "Time since process manager started",
	})

	AppUptime = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "zerohalt_app_uptime_seconds",
		Help: "Time since application started",
	})

	// Connection Metrics
	ActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "zerohalt_active_connections",
		Help: "Current active connections on monitored ports",
	})

	DrainPhaseActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "zerohalt_drain_phase_active",
		Help: "1 if currently draining, 0 otherwise",
	})

	DrainDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "zerohalt_drain_duration_seconds",
		Help: "Time spent draining connections",
	})

	// Health Check Metrics
	HealthRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "zerohalt_health_requests_total",
		Help: "Total health check requests",
	})

	HealthRequestDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "zerohalt_health_request_duration_ms",
		Help: "Health check latency in milliseconds",
	})

	HealthApp = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "zerohalt_health_app",
		Help: "Application health state (0=starting, 1=healthy, 2=unhealthy, 3=draining, 4=terminating)",
	})

	// Signal Metrics
	SignalsReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zerohalt_signals_received_total",
			Help: "Signals received by process manager",
		},
		[]string{"signal"},
	)

	SignalsForwarded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zerohalt_signals_forwarded_total",
			Help: "Signals forwarded to application",
		},
		[]string{"signal"},
	)
)

func init() {
	// Register only our custom metrics
	registry.MustRegister(State)
	registry.MustRegister(Uptime)
	registry.MustRegister(AppUptime)
	registry.MustRegister(ActiveConnections)
	registry.MustRegister(DrainPhaseActive)
	registry.MustRegister(DrainDuration)
	registry.MustRegister(HealthRequests)
	registry.MustRegister(HealthRequestDuration)
	registry.MustRegister(HealthApp)
	registry.MustRegister(SignalsReceived)
	registry.MustRegister(SignalsForwarded)

	// Initialize signal metrics with zero values so they always appear
	SignalsReceived.WithLabelValues("SIGTERM").Add(0)
	SignalsForwarded.WithLabelValues("SIGTERM").Add(0)
}

// Handler returns the Prometheus HTTP handler with ONLY our custom metrics
func Handler() http.Handler {
	baseHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{
			ResponseWriter: w,
			body:           &bytes.Buffer{},
		}

		baseHandler.ServeHTTP(rec, r)

		scanner := bufio.NewScanner(rec.body)
		filtered := &bytes.Buffer{}

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "# HELP") && !strings.HasPrefix(line, "# TYPE") {
				filtered.WriteString(line)
				filtered.WriteString("\n")
			}
		}

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Write(filtered.Bytes())
	})
}

type responseRecorder struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	return r.body.Write(b)
}
