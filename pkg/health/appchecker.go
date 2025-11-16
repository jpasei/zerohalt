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
	"log/slog"
	"net/http"
	"time"

	"github.com/jpasei/zerohalt/pkg/metrics"
)

type AppHealthChecker struct {
	healthURL string
	timeout   time.Duration
	client    *http.Client
}

func NewAppHealthChecker(healthURL string, timeout time.Duration) *AppHealthChecker {
	return &AppHealthChecker{
		healthURL: healthURL,
		timeout:   timeout,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (a *AppHealthChecker) Check() bool {
	if a.healthURL == "" {
		slog.Warn("Health URL is empty")
		metrics.HealthApp.Set(0)
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.healthURL, nil)
	if err != nil {
		slog.Error("Failed to create health check request", "url", a.healthURL, "error", err)
		metrics.HealthApp.Set(0)
		return false
	}

	resp, err := a.client.Do(req)
	if err != nil {
		slog.Error("Health check request failed", "url", a.healthURL, "error", err)
		metrics.HealthApp.Set(0)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		slog.Debug("App health check succeeded", "url", a.healthURL, "status", resp.StatusCode)
		metrics.HealthApp.Set(1)
		return true
	}

	slog.Error("App health check failed", "url", a.healthURL, "status", resp.StatusCode)
	metrics.HealthApp.Set(0)
	return false
}

func (a *AppHealthChecker) WaitForHealthy(startupTimeout time.Duration, checkInterval time.Duration) bool {
	slog.Info("Waiting for application to become healthy", "url", a.healthURL, "timeout", startupTimeout)

	deadline := time.Now().Add(startupTimeout)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		if a.Check() {
			slog.Info("Application is healthy", "url", a.healthURL)
			return true
		}

		if time.Now().After(deadline) {
			slog.Error("Application startup timeout exceeded", "url", a.healthURL, "timeout", startupTimeout)
			return false
		}

		select {
		case <-ticker.C:
			continue
		}
	}
}
