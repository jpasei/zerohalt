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
)

type Server struct {
	port   uint16
	path   string
	state  *State
	server *http.Server
}

func NewServer(port uint16, path string) *Server {
	s := &Server{
		port:  port,
		path:  path,
		state: NewState(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, s.healthHandler)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return s
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

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	state := s.GetState()

	w.Header().Set("Content-Type", "application/json")

	switch state {
	case StateHealthy:
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	case StateStarting:
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"starting"}`))
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
