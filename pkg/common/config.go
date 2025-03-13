package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config represents the application configuration
type Config struct {
	// Server configuration
	ServerPort int
	ServerDir  string

	// Client configuration
	DefaultUser     string
	DefaultPassword string
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ServerPort:      2121,
		ServerDir:       ".",
		DefaultUser:     "anonymous",
		DefaultPassword: "guest@",
	}
}

// LoadConfig loads the configuration from environment variables
func LoadConfig() *Config {
	config := DefaultConfig()

	// Load server configuration
	if port := os.Getenv("ULTRAFTP_SERVER_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			config.ServerPort = p
		}
	}

	if dir := os.Getenv("ULTRAFTP_SERVER_DIR"); dir != "" {
		config.ServerDir = dir
	}

	// Load client configuration
	if user := os.Getenv("ULTRAFTP_DEFAULT_USER"); user != "" {
		config.DefaultUser = user
	}

	if pass := os.Getenv("ULTRAFTP_DEFAULT_PASSWORD"); pass != "" {
		config.DefaultPassword = pass
	}

	return config
}

// ValidateServerConfig validates the server configuration
func (c *Config) ValidateServerConfig() error {
	// Validate port
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return fmt.Errorf("invalid port: %d", c.ServerPort)
	}

	// Validate directory
	absDir, err := filepath.Abs(c.ServerDir)
	if err != nil {
		return fmt.Errorf("invalid directory: %s", c.ServerDir)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("cannot access directory: %s", absDir)
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", absDir)
	}

	return nil
}
