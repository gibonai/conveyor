package init

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// createDirectories creates the necessary directories with proper permissions
func createDirectories(dataDir, configDir, certDir string) error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("unable to get current user: %w", err)
	}

	isRoot := currentUser.Uid == "0"

	// Create data directory
	if err := createDirectory(dataDir, 0755, isRoot); err != nil {
		return fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
	}

	// Create config directory
	if err := createDirectory(configDir, 0755, isRoot); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	// Create certificate directory with more restrictive permissions
	if err := createDirectory(certDir, 0700, isRoot); err != nil {
		return fmt.Errorf("failed to create certificate directory %s: %w", certDir, err)
	}

	return nil
}

// createDirectory creates a directory with specified permissions
func createDirectory(path string, perm os.FileMode, isRoot bool) error {
	// Check if directory already exists
	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path %s exists but is not a directory", path)
		}
		// Directory exists, update permissions if needed
		if info.Mode().Perm() != perm {
			if err := os.Chmod(path, perm); err != nil {
				return fmt.Errorf("failed to update permissions for %s: %w", path, err)
			}
		}
		return nil
	}

	// Create directory with proper permissions
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// For root installations, ensure proper ownership
	if isRoot {
		// Note: In a real implementation, you might want to set specific ownership
		// For now, we'll just ensure the permissions are correct
		if err := os.Chmod(path, perm); err != nil {
			return fmt.Errorf("failed to set permissions for %s: %w", path, err)
		}
	}

	return nil
}

// ensureParentDir ensures the parent directory of a file exists
func ensureParentDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory %s: %w", dir, err)
		}
	}
	return nil
}