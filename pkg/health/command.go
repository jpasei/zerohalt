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
	"os/exec"
	"time"
)

type CommandHealthChecker struct {
	command []string
	timeout time.Duration
}

func NewCommandHealthChecker(command []string, timeout time.Duration) *CommandHealthChecker {
	return &CommandHealthChecker{
		command: command,
		timeout: timeout,
	}
}

func (c *CommandHealthChecker) Check() bool {
	if len(c.command) == 0 {
		slog.Warn("Health check command is empty")
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.command[0], c.command[1:]...)

	err := cmd.Run()

	if err != nil {
		slog.Error("Health check command failed", "command", c.command, "error", err)
		return false
	}

	slog.Debug("Health check command succeeded", "command", c.command)
	return true
}

func (c *CommandHealthChecker) CheckWithDetails() (bool, int, error) {
	if len(c.command) == 0 {
		slog.Warn("Health check command is empty")
		return false, -1, fmt.Errorf("no command configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.command[0], c.command[1:]...)

	err := cmd.Run()

	if err == nil {
		slog.Debug("Health check command succeeded with details", "command", c.command, "exit_code", 0)
		return true, 0, nil
	}

	if exitError, ok := err.(*exec.ExitError); ok {
		slog.Error("Health check command failed with exit code", "command", c.command, "exit_code", exitError.ExitCode(), "error", err)
		return false, exitError.ExitCode(), err
	}

	slog.Error("Health check command failed", "command", c.command, "error", err)
	return false, -1, err
}
