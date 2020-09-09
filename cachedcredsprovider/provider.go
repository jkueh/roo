package cachedcredsprovider

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/sts"
)

var refreshWindowSeconds int
var cacheDir string

func init() {
	// Set a default refresh window
	if refreshWindowSeconds == 0 {
		// Default refreshWindow - 90 seconds prior to expiry.
		refreshWindowSeconds = 90
	}

	// Set a default cache dir.
}

// CachedCredProvider is the custom credential provider that we use in the credential provider chain when creating
// a new AWS SDK session.
type CachedCredProvider struct {
	cachedCredentials CachedCredentials
	cacheFilePath     string
	credentials.Provider
}

// New - Returns an instance of CachedCredProvider, requiring a cache file path.
func New(filePath string) *CachedCredProvider {
	newProvider := &CachedCredProvider{
		cacheFilePath: filePath,
	}

	// Should also take this opportunity to load the credentials from disk.
	newProvider.loadFromDisk()

	return newProvider
}

// Retrieve - Interface function that returns the credential values.
func (p *CachedCredProvider) Retrieve() (credentials.Value, error) {
	err := p.loadFromDisk()

	return p.cachedCredentials.Values, err
}

// IsExpired - Interface function to determine if credentials need to be refreshed.
func (p *CachedCredProvider) IsExpired() bool {
	earlyExpiryTime := p.cachedCredentials.ExpiresAt.Add(-time.Second * time.Duration(refreshWindowSeconds))
	if time.Now().After(earlyExpiryTime) {
		return true
	}
	return false
}

func (p *CachedCredProvider) loadFromDisk() error {

	// Attempt to read the file - Nothing wrong if it doesn't exist.
	if _, err := os.Stat(p.cacheFilePath); os.IsNotExist(err) {
		return err
	}
	var allegedCachedCredentials CachedCredentials
	err := readCachedCredentialsFromFile(p.cacheFilePath, &allegedCachedCredentials)
	if err != nil {
		return err
	}

	// Set the credentials struct in the provider
	p.cachedCredentials = allegedCachedCredentials

	return nil
}

// WriteNewCredentialsFromSTS - Will transform the STS credentials struct to a CachedCredentials struct then overwrite
// current values, and write it all to disk.
func (p *CachedCredProvider) WriteNewCredentialsFromSTS(c *sts.Credentials, filePath string) error {
	timeNow := time.Now()
	latestValidTime := c.Expiration.Add(-time.Second * time.Duration(refreshWindowSeconds))
	if timeNow.After(latestValidTime) {
		log.Println("WARNING: New credentials to write to disk expire within the refresh window.")
		log.Println("Time Now:             ", timeNow)
		log.Println("Latest Valid Time:    ", latestValidTime)
		log.Println("Credential Expiration:", c.Expiration)
	}

	p.cachedCredentials.ExpiresAt = *c.Expiration
	p.cachedCredentials.Values = credentials.Value{
		AccessKeyID:     *c.AccessKeyId,
		SecretAccessKey: *c.SecretAccessKey,
		SessionToken:    *c.SessionToken,
	}

	credentialJSON, err := json.Marshal(p.cachedCredentials)
	if err != nil {
		log.Println("WARNING: An error occurred while trying to marshal credentials into the JSON format:", err)
	}

	// Write to the cacheFile
	err = ioutil.WriteFile(filePath, credentialJSON, 0600)
	if err != nil {
		log.Println("WARNING: An error occurred while trying to write credentials to file:", err)
		log.Println("File path:", filePath)
	}

	return err
}
