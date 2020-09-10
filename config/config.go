package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/jkueh/roo/util"
	"gopkg.in/yaml.v2"
)

// Verbose - Package variable for extra logging
var Verbose bool

// Debug - Package variable for extra logging
var Debug bool

// Config represents the config file.
type Config struct {
	MFASerial string       `yaml:"mfa_serial"`
	Roles     []RoleConfig `yaml:"roles"`
}

// New - Returns a hydrated instance of Config from configFile.
func New(filePath string) *Config {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		bootstrapConfig(filePath)
	}

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

	var foundByName *RoleConfig
	var foundByAlias *RoleConfig

	// This feels like I'm optimising for O(n) where n isn't very large, but let's not jump to conclusions about how many
	// roles people realistically have here...
	for _, roleConfig := range c.Roles {

		if Debug {
			log.Println("Checking the following role config:", roleConfig)
		}

		if Debug {
			log.Println("Found by name:", foundByName)
			log.Println("Found by alias:", foundByAlias)
		}

		// If we've found one by name or ARN, we don't need to do any more processing.
		if foundByName != nil {
			break
		}

		if roleConfig.ARN == searchString {
			if Debug {
				log.Println("Found role by ARN:", roleConfig.ARN)
			}
			return &roleConfig
		}

		// If there's a direct name match, we can set the variable, then exit early.
		if roleConfig.Name == searchString {
			if Debug {
				log.Println("Found role by name:", roleConfig.Name)
			}
			foundByName = &roleConfig
		}

		// If we haven't found one by name or by alias, check the current roleConfig against the aliases.
		if foundByName == nil && foundByAlias == nil {
			if Debug {
				log.Println("Checking for an alias match:", roleConfig.ARN)
			}
			for _, alias := range roleConfig.Aliases {
				if alias == searchString {
					foundByAlias = &roleConfig
					if Debug {
						log.Println("Found role by alias:", alias, foundByAlias)
					}
					break
				}
			}
			// TODO: Work out why the heck this break statement fixes GitHub Issue #5.
			if foundByAlias != nil {
				break
			}
		}
		if Debug {
			fmt.Println()
		}
	}

	if Debug {
		log.Println("Found by name:", foundByName)
		log.Println("Found by alias:", foundByAlias)
	}

	if foundByName != nil {
		if Debug {
			log.Println("Returning match found by name:", foundByName)
		}
		return foundByName
	}
	if foundByAlias != nil {
		if Debug {
			log.Println("Returning match found by alias:", foundByAlias)
		}
		return foundByAlias
	}
	return &RoleConfig{}
}

// bootstrapConfig will generate a generic config file, and exit.
func bootstrapConfig(filePath string) {
	exampleConfigYAML, err := yaml.Marshal(Config{
		MFASerial: "arn:aws:iam::000000000000:mfa/your_mfa_serial",
		Roles: []RoleConfig{
			{
				Name:    "one_of_your_accounts",
				ARN:     "arn:aws:iam::000000000000:role/DeleteOnly",
				Aliases: []string{"delete", "deleteprod"},
			},
			{
				Name:    "another_one_of_your_accounts",
				ARN:     "arn:aws:iam::111111111111:role/ReadOnly",
				Aliases: []string{"readonly", "ro"},
			},
		},
	})
	if err != nil {
		log.Fatalln("Unable to marshal the example config struct into YAML.")
	}

	err = util.EnsureFileExists(filePath, 0600)
	if err != nil {
		log.Fatalln("Unable to create configFile:", filePath, err)
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY, os.ModeAppend)
	if err != nil {
		log.Fatalln("Unable to open config file to bootstrap:", err)
	}

	_, err = file.Write(exampleConfigYAML)
	if err != nil {
		log.Println("Unable to write example config to file:", filePath)
		log.Println(err)
	}

	fmt.Println("Hey there! I noticed you didn't have a configuration file, so I created one for you.")
	fmt.Println("You can find it at", filePath, "- You should probably modify it with the values you need!")

	file.Close()

	os.Exit(100)
}
