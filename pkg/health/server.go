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
	"log/slog"
	"net/http"
	"time"

	"github.com/jpasei/zerohalt/pkg/metrics"
)

type Server struct {
	port       uint16
	path       string
	state      *State
	server     *http.Server
	appChecker *AppHealthChecker
}

func NewServer(port uint16, path string) *Server {
	return NewServerWithAppChecker(port, path, nil)
}

func NewServerWithAppChecker(port uint16, path string, appChecker *AppHealthChecker) *Server {
	s := &Server{
		port:       port,
		path:       path,
		state:      NewState(),
		appChecker: appChecker,
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, s.healthHandler)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return s
}

// EnableMetrics adds the metrics endpoint to the health server
func (s *Server) EnableMetrics(metricsPath string) {
	mux := s.server.Handler.(*http.ServeMux)
	mux.Handle(metricsPath, metrics.Handler())
	slog.Info("Metrics endpoint enabled", "path", metricsPath, "port", s.port)
}

func (s *Server) Start() error {
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Health server error", "error", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) SetState(state HealthState) {
	s.state.Set(state)
}

func (s *Server) GetState() HealthState {
	return s.state.Get()
}

func (s *Server) WaitForAppHealthy(startupTimeout time.Duration, checkInterval time.Duration) bool {
	if s.appChecker == nil {
		slog.Debug("No app health checker configured, skipping app health wait")
		return true
	}

	return s.appChecker.WaitForHealthy(startupTimeout, checkInterval)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		metrics.HealthRequests.Inc()
		duration := time.Since(start).Seconds() * 1000
		metrics.HealthRequestDuration.Set(duration)
	}()

	state := s.GetState()

	w.Header().Set("Content-Type", "application/json")

	switch state {
	case StateHealthy:
		s.handleHealthyState(w)
	case StateStarting:
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"starting"}`))
	case StateUnhealthy:
		s.handleUnhealthyState(w)
	case StateDraining:
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"draining"}`))
	case StateTerminating:
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"terminating"}`))
	default:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"status":"unknown"}`))
	}
}

func (s *Server) handleHealthyState(w http.ResponseWriter) {
	hasAppChecker := s.appChecker != nil

	if hasAppChecker {
		s.writeHealthyStateWithAppCheck(w)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *Server) writeHealthyStateWithAppCheck(w http.ResponseWriter) {
	appIsHealthy := s.appChecker.Check()

	if appIsHealthy {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
		return
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(`{"status":"unhealthy"}`))
}

func (s *Server) handleUnhealthyState(w http.ResponseWriter) {
	hasAppChecker := s.appChecker != nil

	if hasAppChecker {
		s.writeUnhealthyStateWithAppCheck(w)
		return
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(`{"status":"unhealthy"}`))
}

func (s *Server) writeUnhealthyStateWithAppCheck(w http.ResponseWriter) {
	appIsHealthy := s.appChecker.Check()

	if appIsHealthy {
		s.SetState(StateHealthy)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
		return
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	w.Write([]byte(`{"status":"unhealthy"}`))
}
