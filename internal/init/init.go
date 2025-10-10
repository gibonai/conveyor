/*
Package init provides functionality for initializing Conveyor CI for the first time.
This includes creating configuration files, directories, and TLS certificates.
*/
package init

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// Options holds configuration for initializing Conveyor CI
type Options struct {
	ConfigPath     string // Path to config file (empty for default)
	Force          bool   // Overwrite existing files
	AuthEnabled    bool   // Enable authentication
	APIPort        int    // API server port
	NATSPort       int    // NATS server port
	CAPath         string // Path to existing CA certificate
	PrivateKeyPath string // Path to existing private key
	CertPath       string // Path to existing server certificate
}

// Initialize sets up Conveyor CI for the first time
func Initialize(opts Options) error {
	// Determine directories
	dataDir, configDir, certDir, err := determinePaths(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to determine paths: %w", err)
	}

	fmt.Printf("Data directory: %s\n", dataDir)
	fmt.Printf("Config directory: %s\n", configDir)
	fmt.Printf("Certificate directory: %s\n", certDir)

	// Create necessary directories
	if err := createDirectories(dataDir, configDir, certDir); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Generate or copy certificates
	if opts.CAPath != "" && opts.PrivateKeyPath != "" && opts.CertPath != "" {
		// Use existing certificates
		if err := copyExistingCertificates(opts.CAPath, opts.PrivateKeyPath, opts.CertPath, certDir); err != nil {
			return fmt.Errorf("failed to copy existing certificates: %w", err)
		}
		fmt.Println("✅ Using existing certificates")
	} else {
		// Generate new certificates
		if err := generateCertificates(certDir, opts.Force); err != nil {
			return fmt.Errorf("failed to generate certificates: %w", err)
		}
		fmt.Println("✅ Generated TLS certificates")
	}

	// Generate configuration file
	configPath := filepath.Join(configDir, "conveyor.yml")
	if opts.ConfigPath != "" {
		configPath = opts.ConfigPath
	}

	if err := generateConfigFile(configPath, certDir, opts, opts.Force); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}
	fmt.Printf("✅ Generated configuration file: %s\n", configPath)

	return nil
}

// determinePaths determines the appropriate paths for data, config, and certificates
func determinePaths(customConfigPath string) (dataDir, configDir, certDir string, err error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", "", "", fmt.Errorf("unable to get current user: %w", err)
	}

	if currentUser.Uid == "0" {
		// root/system user
		dataDir = "/var/lib/conveyor"
		configDir = "/etc/conveyor"
		certDir = "/etc/conveyor/certs"
	} else {
		// regular user
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			xdgConfig = filepath.Join(currentUser.HomeDir, ".config")
		}
		
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			xdgData = filepath.Join(currentUser.HomeDir, ".local", "share")
		}

		dataDir = filepath.Join(xdgData, "conveyor")
		configDir = filepath.Join(xdgConfig, "conveyor")
		certDir = filepath.Join(configDir, "certs")
	}

	// If custom config path is provided, use its directory for config and certs
	if customConfigPath != "" {
		configDir = filepath.Dir(customConfigPath)
		certDir = filepath.Join(configDir, "certs")
	}

	return dataDir, configDir, certDir, nil
}