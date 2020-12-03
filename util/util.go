package util

import (
	"os"
	"runtime"
)

// EnsureDirExists will create a directory if it doesn't exist.
func EnsureDirExists(dirPath string, fileMode os.FileMode) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, fileMode)
	}
	if runtime.GOOS != "windows" { // We skip the chmod step for Windows... Because we can't chmod.
		return nil
	}
	// This won't return an error if we're setting it to 0700 - which we do in init.go, but for consistency with the other
	// chmod operations we do, we remove it from the code path.
	return os.Chmod(dirPath, fileMode)
}

// EnsureFileExists will create a file if it doesn't exist.
func EnsureFileExists(filePath string, fileMode os.FileMode) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_, err := os.Create(filePath)
		if err != nil {
			return err
		}
		// Ensure it's set to 0600
	}
	return os.Chmod(filePath, fileMode)
}
