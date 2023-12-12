// Package indy-mqtt/cmd/main implements the main() function for indy-mqtt.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"indy-mqtt/internal/command"
	"indy-mqtt/internal/config"
	"indy-mqtt/internal/message"
	"indy-mqtt/internal/util"
)

// version holds the gomarkwiki version, and is set at build time.
var version string

var connectionLost bool = false

const TIMEOUT = 30 * time.Second

func main() {
	// Parse command line
	binaryName := filepath.Base(os.Args[0])
	args := parseCommandLine(binaryName)

	// Configure logging
	util.ConfigureLogging()

	// Read config file
	config := config.LoadConfig()

	// Lookup hostname
	hostname, err := os.Hostname()
	if err != nil {
		util.ERROR.Fatalf("Unable to lookup hostname")
	}

	// Generate client ID
	clientID := fmt.Sprintf("%s-%s", hostname, binaryName)

	// Create command
	cmd, err := command.NewCommand(clientID, args)
	if err != nil {
		util.PrintFatalUsage(err.Error())
	}

	// Connect to MQTT broker
	ackCh := make(chan message.AckMessage)
	client, err := connect(config, clientID, cmd.IsAckExpected, cmd.Host, ackCh)
	if err != nil {
		util.ERROR.Fatalf("Unable to connect: %v", err)
	}

	// Publish message
	var messageBytes []byte
	messageBytes, err = json.Marshal(cmd.Message)
	if err != nil {
		util.ERROR.Fatalf("Error marshaling message: %v", err)
	}
	if util.Verbose {
		util.INFO.Printf("Publishing to topic '%s'", cmd.Topic)
		prettyJSON := marshalToJSONString(cmd.Message)
		util.INFO.Printf("Message:\n%s", prettyJSON)
	}
	token := client.Publish(cmd.Topic, cmd.QOS, false, messageBytes)

	// Create a channel to listen for interrupt signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	defer close(interrupt)

	// Wait for the publish to complete, or an interrupt signal
	publishSuccess := false
	select {
	case <-token.Done():
		if token.Error() != nil {
			util.ERROR.Printf("Failed to publish: %v", token.Error())
		} else {
			util.INFO.Printf("Message published successfully")
			publishSuccess = true
		}
	case <-interrupt:
		fmt.Println("Interrupt signal received. Exiting...")
	}

	// Watch for ack
	if cmd.IsAckExpected && publishSuccess {
		acked := make(chan struct{})
		go func() {
			for ack := range ackCh {
				// Is this the expected ack?
				if ack.ID == cmd.Message.Header.MessageID {
					const STATUS_CODE_OK = 200
					if ack.StatusCode == STATUS_CODE_OK {
						util.INFO.Printf("Message was successfully acknowledged")

						// Print ACK message
						if len(ack.Message) > 0 {
							fmt.Println(ack.Message)
						}

						// Handle ACK
						if err := cmd.HandleAck(ack.Content); err != nil {
							util.ERROR.Printf("Failed to handle ack: %v", err)
						}
					} else {
						util.ERROR.Printf("ACK error code %d: %s", ack.StatusCode, ack.Message)
					}
					close(acked) // Signal that the ack was received
					return
				}
			}
		}()

		// Watch for ACK or interrupt signal
		util.INFO.Printf("Watching for ACK")
		select {
		case <-acked:
			// Ack was received
		case <-time.After(TIMEOUT):
			util.ERROR.Printf("Timed out while waiting for ACK")
		case <-interrupt:
			fmt.Println("Interrupt signal received. Exiting...")
		}
	}

	// Disconnect from the broker
	const DISCONNECT_WAIT = 250 // Milliseconds
	client.Disconnect(DISCONNECT_WAIT)
	util.INFO.Printf("Disconnected from broker")
}

// marshalToJSONString returns the JSON encoding for source.
func marshalToJSONString(source any) string {
	jsonBytes, err := json.MarshalIndent(source, "", "    ")
	if err != nil {
		util.ERROR.Fatalf("Unable to format JSON: %v", err)
	}
	return string(jsonBytes)
}

// prettifyJSON returns a prettified version of source, with items (name-value
// pairs and list elements) on their own lines and indented.
func prettifyJSON(source string) string {
	var buffer bytes.Buffer
	err := json.Indent(&buffer, []byte(source), "", "    ")
	if err != nil {
		util.ERROR.Printf("Unable to prettify '%s': %v", source, err)
		return ""
	}
	return buffer.String()
}

// connect connects to the MQTT broker.
func connect(config *config.Config, clientID string, isAckExpected bool, host string, ackCh chan message.AckMessage) (mqtt.Client, error) {
	// Prepare connection options
	options := mqtt.NewClientOptions()
	brokerUrl := fmt.Sprintf("ssl://%s:%d", *config.Hostname, *config.Port)
	options.AddBroker(brokerUrl)
	options.SetClientID(clientID)
	options.SetUsername(*config.Username)
	options.SetPassword(*config.Password)
	options.SetOrderMatters(false) // Allow out of order messages
	options.ConnectRetry = false   // Don't retry initial connection if connection attempt fails
	options.AutoReconnect = true   // Reconnect if connection goes down
	options.PingTimeout = TIMEOUT
	options.ConnectTimeout = TIMEOUT
	options.WriteTimeout = TIMEOUT
	options.KeepAlive = 10 // Seconds. Send keepalive messages frequently to quickly detect network outages.

	// Handle connection events
	subscribed := make(chan struct{})
	ackTopic := fmt.Sprintf("indy-switch/%s/ack", host)
	options.OnConnect = func(client mqtt.Client) {
		if connectionLost {
			fmt.Println("Connection reestablished")
		} else {
			util.INFO.Printf("Connection established")
		}
		connectionLost = false

		// Subscribe to ack topic
		if isAckExpected {
			util.INFO.Printf("Subscribing to '%s'", ackTopic)
			const ACK_QOS = 1
			token := client.Subscribe(ackTopic, ACK_QOS, func(_ mqtt.Client, msg mqtt.Message) {
				// Display JSON received
				if util.Verbose {
					prettyJSON := prettifyJSON(string(msg.Payload()))
					util.INFO.Printf("ACK received:\n%s", prettyJSON)
				}

				// Unmarshal the ack
				var ack message.AckMessage
				err := json.Unmarshal(msg.Payload(), &ack)
				if err != nil {
					util.ERROR.Printf("ACK could not be parsed: %v", err)
					fmt.Fprintf(os.Stderr, "%s\n", msg.Payload())
					return
				}

				// Forward ack
				ackCh <- ack
			})
			go func() {
				<-token.Done()
				if token.Error() != nil {
					util.ERROR.Printf("Failed to subscribe to '%s': %v", ackTopic, token.Error())
				} else {
					util.INFO.Printf("Susbscribed to '%s'", ackTopic)
					close(subscribed) // Signal that subscribe has completed
				}
			}()
		}
	}
	options.OnConnectionLost = func(client mqtt.Client, err error) {
		util.WARNING.Printf("Connection lost: %v", err)
		connectionLost = true
	}
	options.OnReconnecting = func(client mqtt.Client, options *mqtt.ClientOptions) {
		fmt.Println("Attempting to reconnect")
	}

	// Connect to the broker
	util.INFO.Printf("Connecting to '%s' as user '%s' with client ID '%s'", brokerUrl, *config.Username, clientID)
	client := mqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	// Wait for subscribe to complete
	if isAckExpected {
		select {
		case <-subscribed:
			// Subscribe completed
		case <-time.After(TIMEOUT):
			util.ERROR.Printf("Timed out while waiting to subscribe to '%s'", ackTopic)
		}
	}

	return client, nil
}

// parseCommandLine parses the command line.
func parseCommandLine(binaryName string) []string {
	// Define command line flags.
	printHelp := flag.Bool("help", false, "Show help")
	printVersion := flag.Bool("version", false, "Print version information")
	flag.BoolVar(&util.Verbose, "verbose", false, "Print status messages")
	flag.BoolVar(&util.Debug, "debug", false, "Print debug messages")

	// Define custom usage message.
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [host] [command]\n\n", binaryName)
		fmt.Fprintf(os.Stderr, "Sends commands to the IndySwitch MQTT broker\n\n")
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nCommands:")
		fmt.Fprintln(os.Stderr, "  config timezone [timezone]")
		fmt.Fprintln(os.Stderr, "  config offset [offset]")
		fmt.Fprintln(os.Stderr, "  config suntimes [filename]")
		fmt.Fprintln(os.Stderr, "  status [all]")
		fmt.Fprintln(os.Stderr, "  restart")
		fmt.Fprintln(os.Stderr, "  reset")
		fmt.Fprintln(os.Stderr, "  switch [on|off]")
		fmt.Fprintln(os.Stderr, "\nExamples:")
		fmt.Fprintln(os.Stderr, "  indy-mqtt esp-vorona switch on")
		fmt.Fprintln(os.Stderr, "  indy-mqtt esp-vorona config timezone America/New_York")
		fmt.Fprintln(os.Stderr, "  indy-mqtt esp-vorona config offset 30")
		fmt.Fprintln(os.Stderr, "  indy-mqtt esp-vorona status all")
	}

	// Parse command line.
	flag.Parse()

	// Print help.
	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	// Print version.
	if *printVersion {
		fmt.Printf("%s %s compiled with %s on %s/%s\n",
			binaryName, version, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	return flag.Args()
}
