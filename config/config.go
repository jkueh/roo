package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// Config represents the config file.
type Config struct {
	MFASerial string       `yaml:"mfa_serial"`
	Roles     []RoleConfig `yaml:"roles"`
}

// New - Returns a hydrated instance of Config from configFile.
func New(filePath string) *Config {
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatalln(fmt.Sprintf(
			"An error occurred while trying to open the config file '%s': %s", filePath, err,
		))
	}
	defer configFile.Close()

	// Attempt to read the file
	configBytes, err := ioutil.ReadAll(configFile)
	if err != nil {
		log.Fatalln(fmt.Sprintf(
			"An error occurred while trying to read the config file '%s': %s", filePath, err,
		))
	}

	var config Config
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatalln(fmt.Sprintf("An error occurred while trying to unmarshal config file '%s': %s", filePath, err))
	}

	return &config
}

// GetRole Returns a RoleConfig.
func (c *Config) GetRole(searchString string) *RoleConfig {
	// Search precedence: ARN, Name, then aliases as ordered.

	var foundByARN *RoleConfig
	var foundByName *RoleConfig
	var foundByAlias *RoleConfig

	// This feels like I'm optimising for O(n) where n isn't very large, but let's not jump to conclusions about how many
	// roles people realistically have here...
	for _, roleConfig := range c.Roles {

		// If we've found one by name or ARN, we don't need to do any more processing.
		if foundByARN != nil || foundByName != nil {
			break
		}

		if roleConfig.ARN == searchString {
			foundByARN = &roleConfig
		}

		// If there's a direct name match, we can set the variable, then exit early.
		if roleConfig.Name == searchString {
			foundByName = &roleConfig
			break
		}

		// If we haven't found one by name or by alias, check the current roleConfig against the aliases.
		if foundByName == nil && foundByAlias == nil {
			for _, alias := range roleConfig.Aliases {
				if alias == searchString {
					foundByAlias = &roleConfig
				}
			}
		}
	}

	if foundByARN != nil {
		return foundByARN
	}
	if foundByName != nil {
		return foundByName
	}
	if foundByAlias != nil {
		return foundByAlias
	}
	return &RoleConfig{}
}
