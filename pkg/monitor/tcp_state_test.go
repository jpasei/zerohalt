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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPState_String(t *testing.T) {
	tests := []struct {
		name  string
		state TCPState
		want  string
	}{
		{"StateEstablished", StateEstablished, "ESTABLISHED"},
		{"StateSynSent", StateSynSent, "SYN_SENT"},
		{"StateSynRecv", StateSynRecv, "SYN_RECV"},
		{"StateFinWait1", StateFinWait1, "FIN_WAIT1"},
		{"StateFinWait2", StateFinWait2, "FIN_WAIT2"},
		{"StateTimeWait", StateTimeWait, "TIME_WAIT"},
		{"StateClose", StateClose, "CLOSE"},
		{"StateCloseWait", StateCloseWait, "CLOSE_WAIT"},
		{"StateLastAck", StateLastAck, "LAST_ACK"},
		{"StateListen", StateListen, "LISTEN"},
		{"StateClosing", StateClosing, "CLOSING"},
		{"StateUnknown", TCPState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.state.String()
			assert.Equal(t, tt.want, got)
		})
	}
}
