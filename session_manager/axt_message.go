package session_manager

// Reference: https://github.com/aliyun/cloud-assistant-starter/tree/master/src/main/resources/static/components/session

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"strings"
)

// BigLong wraps *big.Int to simulate the Long type from JavaScript.
type BigLong struct {
	Value *big.Int
}

// NewBigLong creates a new BigLong from an int64.
func NewBigLong(i int64) *BigLong {
	return &BigLong{Value: big.NewInt(i)}
}

// ToInt64 converts the internal *big.Int value back to an int64.
func (l *BigLong) ToInt64() int64 {
	return l.Value.Int64()
}

// SetInt64 sets the internal *big.Int value from an int64.
func (l *BigLong) SetInt64(i int64) {
	l.Value.SetInt64(i)
}

// MessageType represents the message types for the cloud assistant session management WebSocket.
type MessageType uint32

const (
	Input  MessageType = 0 // Client->Server only
	Output MessageType = 1 // Server->Client only
	Resize MessageType = 2 // Client->Server only
	Close  MessageType = 3 // Client->Server only
	Open   MessageType = 4 // Client->Server only
	Status MessageType = 5 // Server->Client only
	Sync   MessageType = 6 // Bidirectional for state synchronization
)

// ChannelState represents the state of the channel/session.
type ChannelState uint32

const (
	Initial ChannelState = 0
	Opening ChannelState = 1
	Aborted ChannelState = 2
	Opened  ChannelState = 3
	Closing ChannelState = 4
	Closed  ChannelState = 5
	Exited  ChannelState = 6
	Stream  ChannelState = 7
	Frozen  ChannelState = 8
)

// AxtMessage represents the data structure for the WebSocket message.
type AxtMessage struct {
	MsgType    MessageType
	Version    string
	InstanceID string
	ChannelID  string
	Timestamp  *BigLong
	InputSeq   uint32
	OutputSeq  uint32
	MsgLength  uint16
	Encoding   uint8
	Reserved   uint8
	Payload    []byte
}

// DecodeMessage deserializes a byte slice into an AxtMessage instance.
func DecodeMessage(data []byte) (*AxtMessage, error) {
	buf := bytes.NewReader(data)

	var msg AxtMessage

	if err := binary.Read(buf, binary.LittleEndian, &msg.MsgType); err != nil {
		return nil, err
	}

	versionBytes := make([]byte, 4)
	if _, err := buf.Read(versionBytes); err != nil {
		return nil, err
	}
	msg.Version = string(bytes.TrimRight(versionBytes, "\x00")) // Remove null padding if any

	var len1, len2 uint8
	if err := binary.Read(buf, binary.LittleEndian, &len1); err != nil {
		return nil, err
	}
	channelIDBytes := make([]byte, len1)
	if _, err := buf.Read(channelIDBytes); err != nil && len1 > 0 {
		return nil, err
	}
	msg.ChannelID = string(channelIDBytes)

	if err := binary.Read(buf, binary.LittleEndian, &len2); err != nil {
		return nil, err
	}
	instanceIDBytes := make([]byte, len2)
	if _, err := buf.Read(instanceIDBytes); err != nil && len2 > 0 {
		return nil, err
	}
	msg.InstanceID = string(instanceIDBytes)

	// Read 8-byte timestamp as uint64 and convert to BigLong
	timestampU64 := uint64(0)
	if err := binary.Read(buf, binary.LittleEndian, &timestampU64); err != nil {
		return nil, err
	}
	msg.Timestamp = NewBigLong(int64(timestampU64))

	if err := binary.Read(buf, binary.LittleEndian, &msg.InputSeq); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &msg.OutputSeq); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &msg.MsgLength); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &msg.Encoding); err != nil {
		return nil, err
	}
	if err := binary.Read(buf, binary.LittleEndian, &msg.Reserved); err != nil {
		return nil, err
	}

	_, err := buf.ReadByte()
	if err != nil && err.Error() != "EOF" { // EOF is expected if payload is empty
		return nil, err
	}
	// The payload is the remaining part of the original data slice.
	// We need to calculate its offset. Total header size is 30 bytes.
	// The payload starts after reading the fixed-size header and variable strings.
	payloadStartOffset := 4 + 4 + 1 + int(len1) + 1 + int(len2) + 8 + 4 + 4 + 2 + 1 + 1
	if len(data) < payloadStartOffset {
		return nil, fmt.Errorf("invalid data length")
	}
	msg.Payload = data[payloadStartOffset:]

	return &msg, nil
}

// EncodeMessage serializes an AxtMessage instance into a byte slice.
func EncodeMessage(message *AxtMessage) ([]byte, error) {
	len1 := len(message.ChannelID)
	len2 := len(message.InstanceID)
	len3 := len(message.Payload)
	totalSize := 30 + len1 + len2 + len3

	buf := new(bytes.Buffer)
	buf.Grow(totalSize) // Pre-allocate to optimize memory allocation

	_ = binary.Write(buf, binary.LittleEndian, uint32(message.MsgType))
	buf.WriteString(strings.Repeat("\x00", 4)) // Ensure 4-byte field, pad with nulls if needed
	buf.WriteString(message.Version)
	buf.Truncate(buf.Len() - (4 - len(message.Version))) // Truncate back if padded

	_ = binary.Write(buf, binary.LittleEndian, uint8(len1))
	if len1 > 0 {
		buf.WriteString(message.ChannelID)
	}

	_ = binary.Write(buf, binary.LittleEndian, uint8(len2))
	if len2 > 0 {
		buf.WriteString(message.InstanceID)
	}

	// Write the timestamp as a uint64
	timestampU64 := uint64(message.Timestamp.ToInt64())
	_ = binary.Write(buf, binary.LittleEndian, timestampU64)

	_ = binary.Write(buf, binary.LittleEndian, message.InputSeq)
	_ = binary.Write(buf, binary.LittleEndian, message.OutputSeq)

	_ = binary.Write(buf, binary.LittleEndian, message.MsgLength)
	_ = binary.Write(buf, binary.LittleEndian, message.Encoding)
	_ = binary.Write(buf, binary.LittleEndian, message.Reserved)
	buf.Write(message.Payload)

	return buf.Bytes(), nil
}

// Display returns a human-readable string for a given ChannelState.
func Display(state ChannelState) string {
	switch state {
	case Initial:
		return "Initial"
	case Opening:
		return "Opening"
	case Opened:
		return "Opened"
	case Aborted:
		return "Aborted"
	case Closing:
		return "Closing"
	case Closed:
		return "Closed"
	case Exited:
		return "Exit"
	case Frozen:
		return "Frozen"
	case Stream:
		return "Stream"
	default:
		return fmt.Sprintf("%d", state)
	}
}

// Print formats an AxtMessage for display based on its type.
func Print(message *AxtMessage, text bool) string {
	getString := func(payload []byte, text bool) string {
		if text {
			return string(payload)
		}
		return fmt.Sprintf("%x", payload) // Use %x for hex representation
	}

	switch message.MsgType {
	case Input:
		input := getString(message.Payload, text)
		return fmt.Sprintf("[Input %d:%d (%d)'%s']", message.InputSeq, message.OutputSeq, message.MsgLength, input)
	case Output:
		output := getString(message.Payload, text)
		return fmt.Sprintf("[Output %d:%d (%d)'%s']", message.InputSeq, message.OutputSeq, message.MsgLength, output)
	case Resize:
		if len(message.Payload) == 4 {
			size := fmt.Sprintf("%x", message.Payload) // Use %x for hex representation
			return fmt.Sprintf("[Resize %d:%d (%d)%s]", message.InputSeq, message.OutputSeq, message.MsgType, size)
		} else {
			jsonStr := string(message.Payload)
			return fmt.Sprintf("[Resize %d:%d (%d)%s]", message.InputSeq, message.OutputSeq, message.MsgLength, jsonStr)
		}
	case Status:
		buf := bytes.NewReader(message.Payload)
		var statusByte uint8
		if err := binary.Read(buf, binary.LittleEndian, &statusByte); err != nil {
			return fmt.Sprintf("[Status Parse Error: %v]", err)
		}
		status := Display(ChannelState(statusByte))
		if status == "Stream" {
			var limit uint32
			if err := binary.Read(buf, binary.LittleEndian, &limit); err != nil {
				return fmt.Sprintf("[Status Parse Limit Error: %v]", err)
			}
			return fmt.Sprintf("[Status %d:%d (%s)%d]", message.InputSeq, message.OutputSeq, status, limit)
		} else {
			// The rest of the payload is the prompt string
			promptBytes, _ := io.ReadAll(buf) // io is imported implicitly in full Go code
			prompt := string(promptBytes)
			return fmt.Sprintf("[Status %d:%d (%s)%s]", message.InputSeq, message.OutputSeq, status, prompt)
		}
	case Sync:
		return fmt.Sprintf("[Sync %d:%d]", message.InputSeq, message.OutputSeq)
	default:
		data := getString(message.Payload, text)
		return fmt.Sprintf("[%d %d:%d (%d)%s]", message.MsgType, message.InputSeq, message.OutputSeq, message.MsgLength, data)
	}
}
