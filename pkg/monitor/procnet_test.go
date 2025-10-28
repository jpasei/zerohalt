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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProcNetTCP_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tcp")

	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	conns, err := parseProcNetTCP(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, conns, 1)

	conn := conns[0]
	assert.Equal(t, "127.0.0.1", conn.LocalAddr)
	assert.Equal(t, uint16(8080), conn.LocalPort)
	assert.Equal(t, StateListen, conn.State)
}

func TestParseProcNetTCP_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tcp")

	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	conns, err := parseProcNetTCP(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, conns, 0)
}

func TestParseProcNetTCP_InvalidFile(t *testing.T) {
	_, err := parseProcNetTCP("/nonexistent/file")
	assert.Error(t, err)
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		wantIP   string
		wantPort uint16
	}{
		{"localhost port 8080", "0100007F:1F90", "127.0.0.1", 8080},
		{"localhost port 80", "0100007F:0050", "127.0.0.1", 80},
		{"any address port 8888", "00000000:22B8", "0.0.0.0", 8888},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP, gotPort := parseAddress(tt.addr)
			assert.Equal(t, tt.wantIP, gotIP)
			assert.Equal(t, tt.wantPort, gotPort)
		})
	}
}

func TestParseAddress_Invalid(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{"no colon", "0100007F1F90"},
		{"empty", ""},
		{"only colon", ":"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, port := parseAddress(tt.addr)
			assert.Equal(t, "", ip)
			assert.Equal(t, uint16(0), port)
		})
	}
}

func TestParseIPv4Hex(t *testing.T) {
	tests := []struct {
		name   string
		hexIP  string
		wantIP string
	}{
		{"127.0.0.1", "0100007F", "127.0.0.1"},
		{"0.0.0.0", "00000000", "0.0.0.0"},
		{"192.168.1.1", "0101A8C0", "192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP := parseIPv4Hex(tt.hexIP)
			assert.Equal(t, tt.wantIP, gotIP)
		})
	}
}

func TestParseIPv4Hex_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		hexIP string
	}{
		{"wrong length", "0100"},
		{"invalid hex", "ZZZZZZZZ"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIP := parseIPv4Hex(tt.hexIP)
			assert.Equal(t, "", gotIP)
		})
	}
}

func TestParseProcNetTCP_ShortLine(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tcp")

	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0
   1: short line`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	conns, err := parseProcNetTCP(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, conns, 1)
}

func TestParseProcNetTCP_MultipleConnections(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tcp")

	content := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345 1 0000000000000000 100 0 0 10 0
   1: 0100007F:0050 00000000:0000 01 00000000:00000000 00:00000000 00000000     0        0 12346 1 0000000000000000 100 0 0 10 0
   2: 00000000:22B8 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12347 1 0000000000000000 100 0 0 10 0`

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	conns, err := parseProcNetTCP(tmpFile)
	assert.NoError(t, err)
	assert.Len(t, conns, 3)

	assert.Equal(t, "127.0.0.1", conns[0].LocalAddr)
	assert.Equal(t, uint16(8080), conns[0].LocalPort)

	assert.Equal(t, "127.0.0.1", conns[1].LocalAddr)
	assert.Equal(t, uint16(80), conns[1].LocalPort)

	assert.Equal(t, "0.0.0.0", conns[2].LocalAddr)
	assert.Equal(t, uint16(8888), conns[2].LocalPort)
}

func TestParseProcNetTCP_ScannerError(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "tcp_huge")

	header := "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode\n"

	longLine := "   0: 0100007F:1F90"
	for i := 0; i < 70000; i++ {
		longLine += " X"
	}
	longLine += "\n"

	content := header + longLine

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	conns, err := parseProcNetTCP(tmpFile)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token too long")
	assert.Empty(t, conns)
}
