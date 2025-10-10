package init

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCertificates(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-certs-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name    string
		force   bool
		setup   func(string) // Function to set up test conditions
		wantErr bool
	}{
		{
			name:  "successful certificate generation",
			force: false,
			setup: func(string) {}, // No setup needed
		},
		{
			name:  "successful certificate generation with force",
			force: true,
			setup: func(certDir string) {
				// Create existing certificate to test force overwrite
				os.WriteFile(filepath.Join(certDir, "ca.crt"), []byte("existing"), 0644)
			},
		},
		{
			name:  "fail when certificates exist without force",
			force: false,
			setup: func(certDir string) {
				// Create existing certificate
				os.WriteFile(filepath.Join(certDir, "ca.crt"), []byte("existing"), 0644)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			certDir := filepath.Join(tmpDir, tt.name)
			err := os.MkdirAll(certDir, 0755)
			require.NoError(t, err)

			// Setup test conditions
			tt.setup(certDir)

			err = generateCertificates(certDir, tt.force)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify all certificate files were created
			caKeyPath := filepath.Join(certDir, "ca-key.pem")
			caCertPath := filepath.Join(certDir, "ca.crt")
			serverKeyPath := filepath.Join(certDir, "server-key.pem")
			serverCertPath := filepath.Join(certDir, "server.crt")

			assert.FileExists(t, caKeyPath)
			assert.FileExists(t, caCertPath)
			assert.FileExists(t, serverKeyPath)
			assert.FileExists(t, serverCertPath)

			// Verify file permissions
			caKeyInfo, err := os.Stat(caKeyPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), caKeyInfo.Mode().Perm())

			serverKeyInfo, err := os.Stat(serverKeyPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0600), serverKeyInfo.Mode().Perm())

			caCertInfo, err := os.Stat(caCertPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0644), caCertInfo.Mode().Perm())

			serverCertInfo, err := os.Stat(serverCertPath)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0644), serverCertInfo.Mode().Perm())

			// Verify certificate content
			t.Run("verify CA certificate", func(t *testing.T) {
				caCertData, err := os.ReadFile(caCertPath)
				require.NoError(t, err)

				block, _ := pem.Decode(caCertData)
				require.NotNil(t, block)
				assert.Equal(t, "CERTIFICATE", block.Type)

				cert, err := x509.ParseCertificate(block.Bytes)
				require.NoError(t, err)
				assert.Equal(t, "Conveyor CI Root CA", cert.Subject.CommonName)
				assert.True(t, cert.IsCA)
			})

			t.Run("verify server certificate", func(t *testing.T) {
				serverCertData, err := os.ReadFile(serverCertPath)
				require.NoError(t, err)

				block, _ := pem.Decode(serverCertData)
				require.NotNil(t, block)
				assert.Equal(t, "CERTIFICATE", block.Type)

				cert, err := x509.ParseCertificate(block.Bytes)
				require.NoError(t, err)
				assert.Equal(t, "Conveyor CI Server", cert.Subject.CommonName)
				assert.False(t, cert.IsCA)

				// Check Subject Alternative Names
				assert.Contains(t, cert.DNSNames, "localhost")
				assert.Contains(t, cert.DNSNames, "conveyor")
				assert.Contains(t, cert.DNSNames, "conveyor.local")
			})

			// Clean up for next test
			os.RemoveAll(certDir)
		})
	}
}

func TestCopyExistingCertificates(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-copy-certs-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create source certificate files
	srcDir := filepath.Join(tmpDir, "src")
	err = os.MkdirAll(srcDir, 0755)
	require.NoError(t, err)

	caPath := filepath.Join(srcDir, "ca.crt")
	keyPath := filepath.Join(srcDir, "server.key")
	certPath := filepath.Join(srcDir, "server.crt")

	err = os.WriteFile(caPath, []byte("test ca certificate"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(keyPath, []byte("test private key"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(certPath, []byte("test server certificate"), 0644)
	require.NoError(t, err)

	// Create destination directory
	destDir := filepath.Join(tmpDir, "dest")
	err = os.MkdirAll(destDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name    string
		caPath  string
		keyPath string
		certPath string
		wantErr bool
	}{
		{
			name:     "successful copy",
			caPath:   caPath,
			keyPath:  keyPath,
			certPath: certPath,
		},
		{
			name:     "missing CA file",
			caPath:   filepath.Join(srcDir, "nonexistent-ca.crt"),
			keyPath:  keyPath,
			certPath: certPath,
			wantErr:  true,
		},
		{
			name:     "missing key file",
			caPath:   caPath,
			keyPath:  filepath.Join(srcDir, "nonexistent-key.pem"),
			certPath: certPath,
			wantErr:  true,
		},
		{
			name:     "missing cert file",
			caPath:   caPath,
			keyPath:  keyPath,
			certPath: filepath.Join(srcDir, "nonexistent-cert.crt"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDestDir := filepath.Join(destDir, tt.name)
			err := os.MkdirAll(testDestDir, 0755)
			require.NoError(t, err)

			err = copyExistingCertificates(tt.caPath, tt.keyPath, tt.certPath, testDestDir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify files were copied
			assert.FileExists(t, filepath.Join(testDestDir, "ca.crt"))
			assert.FileExists(t, filepath.Join(testDestDir, "server-key.pem"))
			assert.FileExists(t, filepath.Join(testDestDir, "server.crt"))

			// Verify content
			copiedCA, err := os.ReadFile(filepath.Join(testDestDir, "ca.crt"))
			require.NoError(t, err)
			assert.Equal(t, "test ca certificate", string(copiedCA))

			copiedKey, err := os.ReadFile(filepath.Join(testDestDir, "server-key.pem"))
			require.NoError(t, err)
			assert.Equal(t, "test private key", string(copiedKey))

			copiedCert, err := os.ReadFile(filepath.Join(testDestDir, "server.crt"))
			require.NoError(t, err)
			assert.Equal(t, "test server certificate", string(copiedCert))
		})
	}
}

func TestFileExists(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-fileexists-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	existingFile := filepath.Join(tmpDir, "existing.txt")
	err = os.WriteFile(existingFile, []byte("test"), 0644)
	require.NoError(t, err)

	nonExistingFile := filepath.Join(tmpDir, "nonexisting.txt")

	assert.True(t, fileExists(existingFile))
	assert.False(t, fileExists(nonExistingFile))
}