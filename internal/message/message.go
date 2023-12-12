// Package indy-mqtt/internal/message implements Message for messages sent to a
// switch, and AckMessage for reponses returned
package message

import (
	"encoding/json"
	"fmt"
	"time"

	"indy-mqtt/internal/util"
)

// Header holds the MQTT message header.
type Header struct {
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
}

// Message holds the MQTT message header and content.
type Message struct {
	Header  Header      `json:"header"`
	Content interface{} `json:"content"`
}

// ControlContent is the message content for the control command,
// to turn a switch on and off.
type ControlContent struct {
	SwitchOn bool `json:"switch_on"`
}

// ConfigContent is the message content for the config command,
// to configure a switch.
type ConfigContent struct {
	Settings map[string]interface{} `json:"settings"`
}

// RestartContent is the message content for the restart and reset commands, to
// restart and reset a switch.
type RestartContent struct {
	Reset bool `json:"reset"`
}

// Empty is the message content for commands that don't need to send content.
type EmptyContent struct {
}

// NewMessage returns a new Message, with its header and `content`.
func NewMessage(clientID string, content interface{}) *Message {
	// Generate the message ID
	messageID := fmt.Sprintf("%s-%s", clientID, util.GenerateHexSuffix())

	// Create the header
	now := time.Now()
	timestamp := now.Format(time.RFC3339)
	header := Header{MessageID: messageID, Timestamp: timestamp}

	// Create messageJSON
	message := Message{Header: header, Content: content}

	return &message
}

// AckMessage is for the ACK returned from a switch to acknowledge a command
type AckMessage struct {
	ID         string          `json:"id"`
	StatusCode int             `json:"status_code"`
	Message    string          `json:"message"`
	Content    json.RawMessage `json:"content"`
}
