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

type TCPState uint8

const (
	StateEstablished TCPState = 0x01
	StateSynSent     TCPState = 0x02
	StateSynRecv     TCPState = 0x03
	StateFinWait1    TCPState = 0x04
	StateFinWait2    TCPState = 0x05
	StateTimeWait    TCPState = 0x06
	StateClose       TCPState = 0x07
	StateCloseWait   TCPState = 0x08
	StateLastAck     TCPState = 0x09
	StateListen      TCPState = 0x0A
	StateClosing     TCPState = 0x0B
)

func (s TCPState) String() string {
	switch s {
	case StateEstablished:
		return "ESTABLISHED"
	case StateSynSent:
		return "SYN_SENT"
	case StateSynRecv:
		return "SYN_RECV"
	case StateFinWait1:
		return "FIN_WAIT1"
	case StateFinWait2:
		return "FIN_WAIT2"
	case StateTimeWait:
		return "TIME_WAIT"
	case StateClose:
		return "CLOSE"
	case StateCloseWait:
		return "CLOSE_WAIT"
	case StateLastAck:
		return "LAST_ACK"
	case StateListen:
		return "LISTEN"
	case StateClosing:
		return "CLOSING"
	default:
		return "UNKNOWN"
	}
}
