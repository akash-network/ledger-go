/*******************************************************************************
*   (c) 2018 ZondaX GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
********************************************************************************/

package ledger_go

import (
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
)

const (
	MinPacketSize = 3
	TagValue      = 0x05
)

const (
	ErrMsgPacketSize       = "packet size must be at least 3"
	ErrMsgInvalidChannel   = "invalid channel"
	ErrMsgInvalidTag       = "invalid tag"
	ErrMsgWrongSequenceIdx = "wrong sequenceIdx"
)

var (
	ErrPacketSize       = errors.New(ErrMsgPacketSize)
	ErrInvalidChannel   = errors.New(ErrMsgInvalidChannel)
	ErrInvalidTag       = errors.New(ErrMsgInvalidTag)
	ErrWrongSequenceIdx = errors.New(ErrMsgWrongSequenceIdx)
)

var (
	codec = binary.BigEndian
)

// ErrorMessage returns a human-readable error message for a given APDU error code.
func ErrorMessage(errorCode uint16) string {
	switch errorCode {
	// FIXME: Code and description don't match for 0x6982 and 0x6983 based on
	// apdu spec: https://www.eftlab.co.uk/index.php/site-map/knowledge-base/118-apdu-response-list

	case 0x6400:
		return "[APDU_CODE_EXECUTION_ERROR] No information given (NV-Ram not changed)"
	case 0x6700:
		return "[APDU_CODE_WRONG_LENGTH] Wrong length"
	case 0x6982:
		return "[APDU_CODE_EMPTY_BUFFER] Security condition not satisfied"
	case 0x6983:
		return "[APDU_CODE_OUTPUT_BUFFER_TOO_SMALL] Authentication method blocked"
	case 0x6984:
		return "[APDU_CODE_DATA_INVALID] Referenced data reversibly blocked (invalidated)"
	case 0x6985:
		return "[APDU_CODE_CONDITIONS_NOT_SATISFIED] Conditions of use not satisfied"
	case 0x6986:
		return "[APDU_CODE_COMMAND_NOT_ALLOWED] Command not allowed / User Rejected (no current EF)"
	case 0x6A80:
		return "[APDU_CODE_BAD_KEY_HANDLE] The parameters in the data field are incorrect"
	case 0x6B00:
		return "[APDU_CODE_INVALID_P1P2] Wrong parameter(s) P1-P2"
	case 0x6D00:
		return "[APDU_CODE_INS_NOT_SUPPORTED] Instruction code not supported or invalid"
	case 0x6E00:
		return "[APDU_CODE_CLA_NOT_SUPPORTED] CLA not supported"
	case 0x6E01:
		return "[APDU_CODE_APP_NOT_OPEN] Ledger Connected but Chain Specific App Not Open"
	case 0x6F00:
		return "APDU_CODE_UNKNOWN"
	case 0x6F01:
		return "APDU_CODE_SIGN_VERIFY_ERROR"
	default:
		return fmt.Sprintf("Error code: %04x", errorCode)
	}
}

func SerializePacket(
	channel uint16,
	command []byte,
	packetSize int,
	sequenceIdx uint16,
) ([]byte, int, error) {
	if packetSize < 3 {
		return nil, 0, ErrPacketSize
	}

	buffer := make([]byte, packetSize)

	offset := 0

	// Insert channel (2 bytes)
	codec.PutUint16(buffer, channel)
	offset += 2

	// Insert tag (1 byte)
	buffer[offset] = 0x05
	offset++

	// Insert sequenceIdx (2 bytes)
	codec.PutUint16(buffer[offset:], sequenceIdx)
	offset += 2

	// Only insert the total size of the command in the first package
	if sequenceIdx == 0 {
		// Insert sequenceIdx (2 bytes)
		commandLength := uint16(len(command))
		codec.PutUint16(buffer[offset:], commandLength)
		offset += 2
	}

	offset = copy(buffer[offset:], command)

	return buffer, offset, nil
}

func DeserializePacket(
	channel uint16,
	packet []byte,
	sequenceIdx uint16,
) ([]byte, uint16, bool, error) {
	const minFirstPacketSize = 7
	const minPacketSize = 5
	const tag = 0x05

	if (sequenceIdx == 0 && len(packet) < minFirstPacketSize) || (sequenceIdx > 0 && len(packet) < minPacketSize) {
		return nil, 0, false, errors.New("cannot deserialize the packet. header information is missing")
	}

	if codec.Uint16(packet) != channel {
		return nil, 0, false, fmt.Errorf("%w: expected %d, got %d", ErrInvalidChannel, channel, codec.Uint16(packet))
	}

	offset := 2

	if packet[offset] != tag {
		return nil, 0, false, fmt.Errorf("invalid tag. expected %d, got %d", tag, packet[offset])
	}

	offset++

	foundSequenceIdx := codec.Uint16(packet[offset:])
	isSequenceZero := foundSequenceIdx == 0

	if foundSequenceIdx != sequenceIdx {
		return nil, 0, isSequenceZero, fmt.Errorf("wrong sequenceIdx: expected %d, got %d", sequenceIdx, foundSequenceIdx)
	}

	offset += 2

	var totalResponseLength uint16
	if sequenceIdx == 0 {
		totalResponseLength = codec.Uint16(packet[offset:])
		offset += 2
	}

	return packet[offset:], totalResponseLength, isSequenceZero, nil
}

// WrapCommandAPDU turns the command into a sequence of 64 byte packets
func WrapCommandAPDU(channel uint16, command []byte, packetSize int) ([]byte, error) {
	var totalResult []byte
	var sequenceIdx uint16

	for len(command) > 0 {
		packet, offset, err := SerializePacket(channel, command, packetSize, sequenceIdx)
		if err != nil {
			return nil, err
		}

		command = command[offset:]
		totalResult = append(totalResult, packet...)
		sequenceIdx++
	}

	return totalResult, nil
}

// UnwrapResponseAPDU parses a response of 64 byte packets into the real data
func UnwrapResponseAPDU(channel uint16, pipe <-chan []byte, packetSize int) ([]byte, error) {
	var sequenceIdx uint16
	var totalResult []byte
	var totalSize uint16
	var foundZeroSequence bool

	for buffer := range pipe {
		result, responseSize, isSequenceZero, err := DeserializePacket(channel, buffer, sequenceIdx)
		if err != nil {
			return nil, err
		}

		// Recover from a known error condition:
		// * Discard messages left over from previous exchange until isSequenceZero == true
		if !foundZeroSequence && !isSequenceZero {
			continue
		}
		foundZeroSequence = true

		// Initialize totalSize
		if totalSize == 0 {
			totalSize = responseSize
		}

		totalResult = append(totalResult, result...)
		sequenceIdx++

		if len(totalResult) >= int(totalSize) {
			break
		}
	}

	// Remove trailing zeros
	return totalResult[:totalSize], nil
}
