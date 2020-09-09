package util

import (
	"os"
)

// EnsureDirExists will create a directory if it doesn't exist.
func EnsureDirExists(dirPath string, fileMode os.FileMode) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, fileMode)
	}
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
