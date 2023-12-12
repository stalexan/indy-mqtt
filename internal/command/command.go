// Package indy-mqtt/internal/command command implements Command, for commands
// the user creates.
package command

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"indy-mqtt/internal/message"
)

// Command holds the contents of a command: what MQTT broker and topic to
// publish to, what to publish, what results are expected, and do with them.
type Command struct {
	Host          string           // Name of device
	Topic         string           // MQTT topic
	QOS           byte             // MQTT QOS
	Message       *message.Message // Payload to publish
	IsAckExpected bool             // Whether an ACK response is expected
	AckHandler    AckHandler       // Handles ACK content
}

// HandleAck calls the AckHandler if there is one, passing it the `content`
// from the ACK.
func (command Command) HandleAck(content []byte) error {
	if command.AckHandler != nil {
		return command.AckHandler.HandleAck(content)
	}
	return nil
}

// AckHandler is implemented for Commands that need to perform some action with
// the contents of an ACK.
type AckHandler interface {
	HandleAck(content []byte) error
}

// GetStatusAckHandler implements AckHandler for the get status command.
type GetStatusAckHandler struct {
	All bool // Whether to print all status fields or just a subset.
}

// createStrFromJsonObj returns a one-line string representation of `obj`,
// where obj is a JSON object with keys that can be parsed as integers (e.g.
// suntimes).
func createStrFromJsonObj(obj json.RawMessage) (string, error) {
	// Is this an object?
	if obj[0] != '{' {
		return "", fmt.Errorf("object not found")
	}

	// Parse JSON
	var data map[string]json.RawMessage
	if err := json.Unmarshal(obj, &data); err != nil {
		return "", fmt.Errorf("error unmarshaling: %v", err)
	}

	// Sort keys
	keys := make([]int, 0, len(data))
	for key := range data {
		var keyInt int
		var err error
		if keyInt, err = strconv.Atoi(key); err != nil {
			return "", fmt.Errorf("integer not found for key '%s'", key)
		}
		keys = append(keys, keyInt)
	}
	sort.Ints(keys)

	// Build string
	var builder strings.Builder
	for i, keyInt := range keys {
		if i > 0 {
			builder.WriteString(", ")
		}
		key := strconv.Itoa(keyInt)
		value := data[key]
		builder.WriteString(fmt.Sprintf("%s: %s", key, value))
	}

	return builder.String(), nil
}

// HandleAck handles the ACK content for the get status command, by printing the
// status returned with the ACK.
func (handler GetStatusAckHandler) HandleAck(content []byte) error {
	// Which attributes to print?
	var attrs []string
	if handler.All {
		attrs = []string{"device", "firmware", "date", "is_on", "sunrise",
			"sunset", "offset", "next_action", "next_action_time", "suntimes"}
	} else {
		attrs = []string{"date", "is_on", "sunrise", "sunset", "offset",
			"next_action", "next_action_time"}
	}

	// Unmarshal the JSON into a map
	var data map[string]json.RawMessage
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("unable to parse ACK JSON content '%s': %v", string(content), err)
	}

	// Print status
	for _, attr := range attrs {
		if val, ok := data[attr]; ok {
			var valStr string
			if val[0] == '{' {
				// val is a JSON object (currently just suntimes)
				var err error
				valStr, err = createStrFromJsonObj(val)
				if err != nil {
					return err
				}
			} else {
				// val is not a JSON object
				valStr = string(val)
				valStr = strings.Trim(valStr, "\"")
			}
			fmt.Printf("%s: %s\n", attr, valStr)
		}
	}

	return nil
}

// NewCommand creates a new Command based on the command line `args` provided by the user.
func NewCommand(clientID string, args []string) (*Command, error) {
	// What host?
	if len(args) == 0 {
		return nil, fmt.Errorf("no host specified")
	}
	host := args[0]
	args = args[1:]

	// What command is this?
	if len(args) == 0 {
		return nil, fmt.Errorf("no command specified")
	}
	cmdStr := args[0]
	args = args[1:]

	// Create command
	var cmd *Command
	var err error
	switch cmdStr {
	case "switch":
		cmd, err = NewControlCommand(clientID, host, args)
		if err != nil {
			return nil, err
		}
	case "config":
		cmd, err = NewConfigCommand(clientID, host, args)
		if err != nil {
			return nil, err
		}
	case "status":
		cmd, err = NewGetStatusCommand(clientID, host, args)
		if err != nil {
			return nil, err
		}
	case "restart":
		cmd, err = NewRestartCommand(clientID, host, args)
		if err != nil {
			return nil, err
		}
	case "reset":
		cmd, err = NewResetCommand(clientID, host, args)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unrecognized command %s", cmdStr)
	}

	return cmd, nil
}

// NewControlCommand creates a control command, to turn switch `host` on or off.
func NewControlCommand(clientID string, host string, args []string) (*Command, error) {
	// On or off?
	if len(args) == 0 {
		return nil, fmt.Errorf("switch command is missing the on/off parameter")
	}
	switchOnStr := args[0]
	args = args[1:]
	if switchOnStr != "on" && switchOnStr != "off" {
		return nil, fmt.Errorf("switch command is expecting on or off instead of %s", switchOnStr)
	}
	switchOn := switchOnStr == "on"

	// Are there any unexpected arguments?
	if len(args) != 0 {
		return nil, fmt.Errorf("unexpected arguments for switch command")
	}

	// Create control command
	topic := fmt.Sprintf("indy-switch/%s/control", host)
	msg := message.NewMessage(clientID, message.ControlContent{SwitchOn: switchOn})
	cmd := &Command{Host: host, Topic: topic, QOS: 2, Message: msg, IsAckExpected: true}

	return cmd, nil
}

// NewConfigCommand creates a config command, to configure switch `host`.
func NewConfigCommand(clientID string, host string, args []string) (*Command, error) {
	// Parse setting
	settings := make(map[string]interface{})
	if len(args) == 0 {
		return nil, fmt.Errorf("config command is missing setting")
	}
	settingName := args[0]
	args = args[1:]
	switch settingName {
	case "timezone":
		// What timezone?
		if len(args) == 0 {
			return nil, fmt.Errorf("timezone missing")
		}
		timezone := args[0]
		args = args[1:]
		settings[settingName] = timezone
	case "offset":
		// What offset?
		if len(args) == 0 {
			return nil, fmt.Errorf("offset missing")
		}
		offsetStr := args[0]
		args = args[1:]
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset <= 0 {
			return nil, fmt.Errorf("offset needs to be a postive integer")
		}
		settings[settingName] = offset
	case "suntimes":
		// What file?
		if len(args) == 0 {
			return nil, fmt.Errorf("file name missing")
		}
		filename := args[0]
		args = args[1:]

		// Parse file
		suntimes, err := readSuntimes(filename)
		if err != nil {
			return nil, err
		}
		settings[settingName] = suntimes
	default:
		return nil, fmt.Errorf("unrecognized setting %s", settingName)
	}

	// Are there any unexpected arguments?
	if len(args) != 0 {
		return nil, fmt.Errorf("unexpected arguments for config command")
	}

	// Create config command
	topic := fmt.Sprintf("indy-switch/%s/config", host)
	msg := message.NewMessage(clientID, message.ConfigContent{Settings: settings})
	cmd := &Command{Host: host, Topic: topic, QOS: 2, Message: msg, IsAckExpected: true}

	return cmd, nil
}

// NewGetStatusCommand creates a get status command, to get the status of switch `host`.
func NewGetStatusCommand(clientID string, host string, args []string) (*Command, error) {
	// Print all status fields, or just a subset?
	all := false
	if len(args) == 1 {
		allStr := args[0]
		args = args[1:]
		if allStr != "all" {
			return nil, fmt.Errorf("status command is expecting all instead of %s", allStr)
		}
		all = true
	}

	// Are there any unexpected arguments?
	if len(args) != 0 {
		return nil, fmt.Errorf("unexpected arguments for status command")
	}

	// Create command
	topic := fmt.Sprintf("indy-switch/%s/status/get", host)
	msg := message.NewMessage(clientID, message.EmptyContent{})
	cmd := &Command{Host: host, Topic: topic, QOS: 2, Message: msg, IsAckExpected: true, AckHandler: GetStatusAckHandler{All: all}}

	return cmd, nil
}

// NewRestartCommand creates a restart command, to restart switch `host`.
func NewRestartCommand(clientID string, host string, args []string) (*Command, error) {
	// Are there any unexpected arguments?
	if len(args) != 0 {
		return nil, fmt.Errorf("unexpected arguments for restart command")
	}

	// Create command
	topic := fmt.Sprintf("indy-switch/%s/restart", host)
	msg := message.NewMessage(clientID, message.RestartContent{Reset: false})
	cmd := &Command{Host: host, Topic: topic, QOS: 2, Message: msg, IsAckExpected: false}

	return cmd, nil
}

// NewResetCommand creates a reset command, to reset switch `host`.
func NewResetCommand(clientID string, host string, args []string) (*Command, error) {
	// Are there any unexpected arguments?
	if len(args) != 0 {
		return nil, fmt.Errorf("unexpected arguments for reset command")
	}

	// Create command
	topic := fmt.Sprintf("indy-switch/%s/restart", host)
	msg := message.NewMessage(clientID, message.RestartContent{Reset: true})
	cmd := &Command{Host: host, Topic: topic, QOS: 2, Message: msg, IsAckExpected: false}

	return cmd, nil
}

// readSuntimes reads and parses the JSON suntimes file `filename`, and returns
// the results. The expected format for the JSON is the same as that used by
// indy-switch. For example:
//
//	{
//	  "1":  ["6:53 AM", "6:03 PM"],
//	  "2":  ["6:46 AM", "6:20 PM"],
//	  "3":  ["6:26 AM", "6:29 PM"],
//	  "4":  ["6:02 AM", "6:36 PM"],
//	  "5":  ["5:45 AM", "6:45 PM"],
//	  "6":  ["5:43 AM", "6:56 PM"],
//	  "7":  ["5:51 AM", "6:58 PM"],
//	  "8":  ["6:01 AM", "6:45 PM"],
//	  "9":  ["6:07 AM", "6:20 PM"],
//	  "10": ["6:12 AM", "5:57 PM"],
//	  "11": ["6:25 AM", "5:42 PM"],
//	  "12": ["6:42 AM", "5:46 PM"]
//	}
func readSuntimes(filename string) (*map[int][2]string, error) {
	// Open file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open '%s': %v", filename, err)
	}
	defer file.Close()

	// Parse file
	var suntimes map[int][2]string
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&suntimes); err != nil {
		return nil, fmt.Errorf("unable to parse JSON: %v", err)
	}

	return &suntimes, nil
}
