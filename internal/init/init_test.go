package init

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeterminePaths(t *testing.T) {
	// Save original user for cleanup
	originalUID := os.Getenv("FAKE_UID")
	defer func() {
		if originalUID != "" {
			os.Setenv("FAKE_UID", originalUID)
		} else {
			os.Unsetenv("FAKE_UID")
		}
	}()

	tests := []struct {
		name             string
		customConfigPath string
		isRoot           bool
		expectedPaths    func(currentUser *user.User) (string, string, string)
	}{
		{
			name:   "root user with default paths",
			isRoot: true,
			expectedPaths: func(currentUser *user.User) (string, string, string) {
				return "/var/lib/conveyor", "/etc/conveyor", "/etc/conveyor/certs"
			},
		},
		{
			name:   "regular user with default paths",
			isRoot: false,
			expectedPaths: func(currentUser *user.User) (string, string, string) {
				dataDir := filepath.Join(currentUser.HomeDir, ".local", "share", "conveyor")
				configDir := filepath.Join(currentUser.HomeDir, ".config", "conveyor")
				certDir := filepath.Join(configDir, "certs")
				return dataDir, configDir, certDir
			},
		},
		{
			name:             "custom config path",
			customConfigPath: "/custom/path/conveyor.yml",
			isRoot:           false,
			expectedPaths: func(currentUser *user.User) (string, string, string) {
				dataDir := filepath.Join(currentUser.HomeDir, ".local", "share", "conveyor")
				configDir := "/custom/path"
				certDir := "/custom/path/certs"
				return dataDir, configDir, certDir
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock user context if needed
			if tt.isRoot {
				// We can't easily mock user.Current(), so we'll test with actual user
				// and verify the logic works for different scenarios
				currentUser, err := user.Current()
				require.NoError(t, err)
				
				// Skip root test if not actually root
				if currentUser.Uid != "0" {
					t.Skip("Skipping root test when not running as root")
				}
			}

			dataDir, configDir, certDir, err := determinePaths(tt.customConfigPath)
			require.NoError(t, err)

			currentUser, err := user.Current()
			require.NoError(t, err)

			expectedDataDir, expectedConfigDir, expectedCertDir := tt.expectedPaths(currentUser)
			
			// For non-root users, we verify the paths follow the expected pattern
			if currentUser.Uid != "0" && !tt.isRoot {
				assert.Equal(t, expectedDataDir, dataDir)
				assert.Equal(t, expectedConfigDir, configDir)
				assert.Equal(t, expectedCertDir, certDir)
			}

			// Verify paths are absolute
			assert.True(t, filepath.IsAbs(dataDir))
			assert.True(t, filepath.IsAbs(configDir))
			assert.True(t, filepath.IsAbs(certDir))
		})
	}
}

func TestInitialize(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-init-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "conveyor.yml")

	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name: "successful initialization with defaults",
			opts: Options{
				ConfigPath:  configPath,
				Force:       true,
				AuthEnabled: false,
				APIPort:     8080,
				NATSPort:    4222,
			},
			wantErr: false,
		},
		{
			name: "successful initialization with auth enabled",
			opts: Options{
				ConfigPath:  configPath,
				Force:       true,
				AuthEnabled: true,
				APIPort:     9090,
				NATSPort:    4223,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.RemoveAll(tmpDir)
			os.MkdirAll(tmpDir, 0755)

			err := Initialize(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify config file was created
			assert.FileExists(t, configPath)

			// Verify certificate directory was created
			certDir := filepath.Join(filepath.Dir(configPath), "certs")
			assert.DirExists(t, certDir)

			// Verify certificates were generated
			assert.FileExists(t, filepath.Join(certDir, "ca.crt"))
			assert.FileExists(t, filepath.Join(certDir, "ca-key.pem"))
			assert.FileExists(t, filepath.Join(certDir, "server.crt"))
			assert.FileExists(t, filepath.Join(certDir, "server-key.pem"))
		})
	}
}

func TestInitializeWithExistingCerts(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-init-existing-certs-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create mock certificate files
	caPath := filepath.Join(tmpDir, "existing-ca.crt")
	keyPath := filepath.Join(tmpDir, "existing-key.pem")
	certPath := filepath.Join(tmpDir, "existing-cert.crt")

	// Write mock certificate content
	err = os.WriteFile(caPath, []byte("mock ca certificate"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyPath, []byte("mock private key"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(certPath, []byte("mock server certificate"), 0644)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, "conveyor.yml")

	opts := Options{
		ConfigPath:     configPath,
		Force:          true,
		AuthEnabled:    false,
		APIPort:        8080,
		NATSPort:       4222,
		CAPath:         caPath,
		PrivateKeyPath: keyPath,
		CertPath:       certPath,
	}

	err = Initialize(opts)
	require.NoError(t, err)

	// Verify config file was created
	assert.FileExists(t, configPath)

	// Verify certificates were copied
	certDir := filepath.Join(filepath.Dir(configPath), "certs")
	assert.FileExists(t, filepath.Join(certDir, "ca.crt"))
	assert.FileExists(t, filepath.Join(certDir, "server-key.pem"))
	assert.FileExists(t, filepath.Join(certDir, "server.crt"))

	// Verify content matches
	copiedCA, err := os.ReadFile(filepath.Join(certDir, "ca.crt"))
	require.NoError(t, err)
	assert.Equal(t, "mock ca certificate", string(copiedCA))
}