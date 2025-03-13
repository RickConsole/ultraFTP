package common

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureDirectoryExists checks if a directory exists and creates it if it doesn't
func EnsureDirectoryExists(path string) error {
	// Check if the directory exists
	info, err := os.Stat(path)
	if err == nil {
		// Path exists, check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", path)
		}
		return nil
	}

	// If the error is not that the file doesn't exist, return it
	if !os.IsNotExist(err) {
		return err
	}

	// Create the directory
	return os.MkdirAll(path, 0755)
}

// ResolvePath resolves a path relative to a base directory
func ResolvePath(basePath, relativePath string) (string, error) {
	// If the relative path is absolute, return it directly
	if filepath.IsAbs(relativePath) {
		return relativePath, nil
	}

	// Otherwise, join it with the base path
	return filepath.Join(basePath, relativePath), nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirectoryExists checks if a directory exists
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
