// Package indy-mqtt/internal/config loads config files.
package config

import (
	"encoding/json"
	"os"

	"indy-mqtt/internal/util"
)

// configRegular holds config values read from the file config.json.
type configRegular struct {
	Hostname *string `json:"hostname"`
	Port     *int    `json:"port"`
}

// configSecrets holds config values read from the file config-secrets.json.
type configSecrets struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
}

// Config holds all config values.
type Config struct {
	configRegular
	configSecrets
}

// checkFields checks that the fields in `config` are set.
func (config configRegular) checkFields(path string) {
	if config.Hostname == nil {
		util.ERROR.Fatalf("hostname not found in '%s'", path)
	}
	if config.Port == nil {
		util.ERROR.Fatalf("port not found in '%s'", path)
	}
}

// checkFields checks that the fields in `config` are set.
func (config configSecrets) checkFields(path string) {
	if config.Username == nil {
		util.ERROR.Fatalf("username not found in '%s'", path)
	}
	if config.Password == nil {
		util.ERROR.Fatalf("password not found in '%s'", path)
	}
}

// LoadConfig reads and parses the JSON config file at `path` and returns the
// results in `dest`.
func loadConfig(path string, dest any) {
	// Read config file
	bytes, err := os.ReadFile(path)
	if err != nil {
		util.ERROR.Fatalf("Failed to read config file '%s': %v", path, err)
	}

	// Parse config file
	err = json.Unmarshal(bytes, &dest)
	if err != nil {
		util.ERROR.Fatalf("Failed to unmarshal config file '%s': %v", path, err)
	}
}

// LoadConfig returns a Config struct that contains the config values read from
// the files config.json and config-secrets.json.
func LoadConfig() *Config {
	// Load config.json
	var config1 configRegular
	path := "internal/config/config.json"
	loadConfig(path, &config1)
	config1.checkFields(path)

	// Load config-secrets.json
	var config2 configSecrets
	path = "internal/config/config-secrets.json"
	loadConfig(path, &config2)
	config2.checkFields(path)

	return &Config{configRegular: config1, configSecrets: config2}
}
