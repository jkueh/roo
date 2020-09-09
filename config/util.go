package config

import (
	"encoding/gob"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
)

// LoadCredentialsFromFile - Attempts to load existing credentials from file.
func LoadCredentialsFromFile(filePath string) (*credentials.Credentials, error) {
	// Attempt to read the file - Nothing wrong if it doesn't exist.
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return &credentials.Credentials{}, nil
	}

	var readCredentials credentials.Credentials
	err := ReadGobFromFile(filePath, &readCredentials)
	return &readCredentials, err
}

// WriteGobToFile - Writes a gob-encoded interface to file.
func WriteGobToFile(filePath string, object interface{}) error {
	file, err := os.Create(filePath)
	if err == nil {
		encoder := gob.NewEncoder(file)
		encoder.Encode(object)
	}
	file.Close()
	return err
}

// ReadGobFromFile - Reads a gob-encoded interface from file.
func ReadGobFromFile(filePath string, object interface{}) error {
	file, err := os.Open(filePath)
	if err == nil {
		decoder := gob.NewDecoder(file)
		err = decoder.Decode(object)
	}
	file.Close()
	return err
}

// CredentialsNeedRefresh - Calculates whether tokens need to be refreshed, based on existence, or if time.Now() is
// within refreshWindowMinutes of expiry.
func CredentialsNeedRefresh(creds *credentials.Credentials, refreshWindowMinutes int) bool {
	// The first thing we need to do is check that we have valid Credentials data to avoid a segfault.
	// Surprisingly, we can do that with creds.IsExpired().
	if creds.IsExpired() {
		return true
	}

	credExpiryTime, err := creds.ExpiresAt()
	if err != nil {
		log.Println("Unable to get credential expiry time:", err, "- Will refresh credentials.")
		return true
	}

	refreshTime := credExpiryTime.Add(
		-time.Minute * time.Duration(refreshWindowMinutes),
	)

	// If they're due to be refreshed... Should probably refresh them.
	if time.Now().After(refreshTime) {
		return true
	}

	// ... Otherwise we assume that they're fine.
	return false
}
