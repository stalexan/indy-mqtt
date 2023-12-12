// Package indy-mqtt/internal/util implements utility routines for printing messages, warnings, and errors.
package util

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"unicode"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var Verbose bool
var Debug bool

type Logger interface {
	Fatalf(format string, v ...interface{})
	Printf(format string, v ...interface{})
}

type NOOPLogger struct{}

func (NOOPLogger) Fatalf(format string, v ...interface{}) {}
func (NOOPLogger) Printf(format string, v ...interface{}) {}

// Loggers
var INFO Logger
var WARNING Logger
var ERROR Logger

// ConfigureLogging configures paho.mqtt logging and creates loggers for local logging.
func ConfigureLogging() {
	// Configure paho.mqtt logging
	const SUFFIX = "paho.mqtt"
	// const loggingFlags = log.Lmsgprefix
	const loggingFlags = log.Ldate | log.Ltime | log.Lmsgprefix
	if Verbose {
		mqtt.WARN = log.New(os.Stderr, fmt.Sprintf("WARNING (%s): ", SUFFIX), loggingFlags)
	}
	if Debug {
		mqtt.DEBUG = log.New(os.Stdout, fmt.Sprintf("DEBUG (%s): ", SUFFIX), loggingFlags)
	}
	mqtt.ERROR = log.New(os.Stderr, fmt.Sprintf("ERROR (%s): ", SUFFIX), loggingFlags)
	mqtt.CRITICAL = log.New(os.Stderr, fmt.Sprintf("CRITICAL (%s): ", SUFFIX), loggingFlags)

	// Configure local logging
	if Verbose {
		INFO = log.New(os.Stdout, "INFO: ", loggingFlags)
	} else {
		INFO = NOOPLogger{}
	}
	WARNING = log.New(os.Stderr, "WARNING: ", loggingFlags)
	ERROR = log.New(os.Stderr, "ERROR: ", loggingFlags)
}

// BoolAsStr returns a string representation of a bool.
func BoolAsStr(value bool) string {
	if value {
		return "true"
	} else {
		return "false"
	}
}

// capitalizeFirstLetter returns `input` with the first letter capitalized.
func capitalizeFirstLetter(input string) string {
	if input == "" {
		return input
	}

	// Convert the string to a rune slice
	runes := []rune(input)

	// Capitalize the first letter
	runes[0] = unicode.ToTitle(runes[0])

	// Convert the rune slice back to a string
	return string(runes)
}

// PrintFatalUsage prints an error message followed by the usage message and then exits.
func PrintFatalUsage(message string) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n\n", capitalizeFirstLetter(message))
	flag.Usage()
	os.Exit(0)
}

// GenerateHexSuffix returns a string of random hex numbers in the form ABCD-0123.
func GenerateHexSuffix() string {
	// Generate random data.
	const HEX_COUNT = 8
	const HEX_BOUND = 16
	hexDigits := make([]byte, HEX_COUNT)
	for i := 0; i < HEX_COUNT; i++ {
		hexDigits[i] = byte(rand.Intn(HEX_BOUND))
	}

	// Format data as "ABCD-EF01"
	return fmt.Sprintf(
		"%X%X%X%X-%X%X%X%X",
		hexDigits[0], hexDigits[1], hexDigits[2], hexDigits[3],
		hexDigits[4], hexDigits[5], hexDigits[6], hexDigits[7])
}
