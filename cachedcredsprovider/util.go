package cachedcredsprovider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func readCachedCredentialsFromFile(filePath string, creds *CachedCredentials) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("File " + filePath + " does not exist")
	}

	fileContents, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(fileContents, &creds)
	if err != nil {
		return err
	}

	return nil
}
