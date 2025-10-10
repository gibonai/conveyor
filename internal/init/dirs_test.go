package init

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateDirectories(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-dirs-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	currentUser, err := user.Current()
	require.NoError(t, err)
	isRoot := currentUser.Uid == "0"

	tests := []struct {
		name    string
		dataDir string
		configDir string
		certDir string
		setup   func() // Function to set up test conditions
		wantErr bool
	}{
		{
			name:      "create new directories",
			dataDir:   filepath.Join(tmpDir, "data"),
			configDir: filepath.Join(tmpDir, "config"),
			certDir:   filepath.Join(tmpDir, "config", "certs"),
			setup:     func() {}, // No setup needed
		},
		{
			name:      "directories already exist",
			dataDir:   filepath.Join(tmpDir, "existing-data"),
			configDir: filepath.Join(tmpDir, "existing-config"),
			certDir:   filepath.Join(tmpDir, "existing-config", "certs"),
			setup: func() {
				// Create directories beforehand
				os.MkdirAll(filepath.Join(tmpDir, "existing-data"), 0755)
				os.MkdirAll(filepath.Join(tmpDir, "existing-config"), 0755)
				os.MkdirAll(filepath.Join(tmpDir, "existing-config", "certs"), 0700)
			},
		},
		{
			name:      "file exists where directory should be",
			dataDir:   filepath.Join(tmpDir, "file-conflict-data"),
			configDir: filepath.Join(tmpDir, "file-conflict-config"),
			certDir:   filepath.Join(tmpDir, "file-conflict-config", "certs"),
			setup: func() {
				// Create a file where directory should be
				err := os.WriteFile(filepath.Join(tmpDir, "file-conflict-data"), []byte("conflict"), 0644)
				require.NoError(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test conditions
			tt.setup()

			err := createDirectories(tt.dataDir, tt.configDir, tt.certDir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify directories were created
			assert.DirExists(t, tt.dataDir)
			assert.DirExists(t, tt.configDir)
			assert.DirExists(t, tt.certDir)

			// Verify permissions
			dataInfo, err := os.Stat(tt.dataDir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0755), dataInfo.Mode().Perm())

			configInfo, err := os.Stat(tt.configDir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0755), configInfo.Mode().Perm())

			certInfo, err := os.Stat(tt.certDir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0700), certInfo.Mode().Perm())

			// Clean up test directory
			if !tt.wantErr {
				os.RemoveAll(tt.dataDir)
				os.RemoveAll(tt.configDir)
			}
		})
	}

	// Skip root-specific test if not running as root
	if !isRoot {
		t.Log("Skipping root-specific directory tests (not running as root)")
	}
}

func TestCreateDirectory(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-create-dir-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	currentUser, err := user.Current()
	require.NoError(t, err)
	isRoot := currentUser.Uid == "0"

	tests := []struct {
		name    string
		path    string
		perm    os.FileMode
		setup   func(string) // Function to set up test conditions
		wantErr bool
	}{
		{
			name: "create new directory",
			path: filepath.Join(tmpDir, "new-dir"),
			perm: 0755,
			setup: func(string) {}, // No setup needed
		},
		{
			name: "create nested directories",
			path: filepath.Join(tmpDir, "nested", "deep", "directory"),
			perm: 0700,
			setup: func(string) {}, // No setup needed
		},
		{
			name: "directory already exists with correct permissions",
			path: filepath.Join(tmpDir, "existing-correct"),
			perm: 0755,
			setup: func(path string) {
				err := os.MkdirAll(path, 0755)
				require.NoError(t, err)
			},
		},
		{
			name: "directory exists with wrong permissions - should update",
			path: filepath.Join(tmpDir, "existing-wrong-perm"),
			perm: 0700,
			setup: func(path string) {
				err := os.MkdirAll(path, 0755)
				require.NoError(t, err)
			},
		},
		{
			name: "file exists at path - should fail",
			path: filepath.Join(tmpDir, "file-at-path"),
			perm: 0755,
			setup: func(path string) {
				err := os.WriteFile(path, []byte("content"), 0644)
				require.NoError(t, err)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test conditions
			tt.setup(tt.path)

			err := createDirectory(tt.path, tt.perm, isRoot)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify directory exists
			assert.DirExists(t, tt.path)

			// Verify permissions
			info, err := os.Stat(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.perm, info.Mode().Perm())
		})
	}
}

func TestEnsureParentDir(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "conveyor-parent-dir-test-")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		filePath string
		setup    func() // Function to set up test conditions
		wantErr  bool
	}{
		{
			name:     "create parent directories for nested file",
			filePath: filepath.Join(tmpDir, "nested", "deep", "file.txt"),
			setup:    func() {}, // No setup needed
		},
		{
			name:     "parent directory already exists",
			filePath: filepath.Join(tmpDir, "existing", "file.txt"),
			setup: func() {
				err := os.MkdirAll(filepath.Join(tmpDir, "existing"), 0755)
				require.NoError(t, err)
			},
		},
		{
			name:     "file in root of temp directory",
			filePath: filepath.Join(tmpDir, "root-file.txt"),
			setup:    func() {}, // Parent already exists (tmpDir)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test conditions
			tt.setup()

			err := ensureParentDir(tt.filePath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify parent directory exists
			parentDir := filepath.Dir(tt.filePath)
			assert.DirExists(t, parentDir)

			// Verify parent directory permissions
			info, err := os.Stat(parentDir)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
		})
	}
}