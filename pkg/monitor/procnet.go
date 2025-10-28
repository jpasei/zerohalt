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

package monitor

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

type Connection struct {
	LocalAddr  string
	LocalPort  uint16
	RemoteAddr string
	RemotePort uint16
	State      TCPState
	UID        uint32
}

var parseProcNetTCP = parseProcNetTCPImpl

func parseProcNetTCPImpl(path string) ([]Connection, error) {
	file, err := os.Open(path)
	if err != nil {
		slog.Error("Failed to open proc net file", "path", path, "error", err)
		return nil, err
	}
	defer file.Close()

	var conns []Connection
	scanner := bufio.NewScanner(file)

	scanner.Scan()

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 10 {
			slog.Debug("Skipping malformed line in proc net file", "path", path, "line_number", lineNum, "field_count", len(fields))
			continue
		}

		localAddr, localPort := parseAddress(fields[1])
		remoteAddr, remotePort := parseAddress(fields[2])

		state, _ := strconv.ParseUint(fields[3], 16, 8)
		uid, _ := strconv.ParseUint(fields[7], 10, 32)

		conn := Connection{
			LocalAddr:  localAddr,
			LocalPort:  localPort,
			RemoteAddr: remoteAddr,
			RemotePort: remotePort,
			State:      TCPState(state),
			UID:        uint32(uid),
		}

		conns = append(conns, conn)
	}

	if err := scanner.Err(); err != nil {
		slog.Error("Error reading proc net file", "path", path, "error", err)
		return conns, err
	}

	slog.Debug("Parsed proc net file", "path", path, "connection_count", len(conns))
	return conns, nil
}

func parseAddress(addr string) (string, uint16) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return "", 0
	}

	port, _ := strconv.ParseUint(parts[1], 16, 16)

	ipHex := parts[0]
	ip := parseIPv4Hex(ipHex)

	return ip, uint16(port)
}

func parseIPv4Hex(hexIP string) string {
	if len(hexIP) != 8 {
		return ""
	}

	bytes, err := hex.DecodeString(hexIP)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%d.%d.%d.%d", bytes[3], bytes[2], bytes[1], bytes[0])
}
