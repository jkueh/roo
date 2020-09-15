package config

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jkueh/roo/util"
	"gopkg.in/yaml.v2"
)

// Verbose - Package variable for extra logging
var Verbose bool

// Debug - Package variable for extra logging
var Debug bool

// Config represents the config file.
type Config struct {
	DefaultProfile string       `yaml:"default_profile"`
	MFASerial      string       `yaml:"mfa_serial"`
	Roles          []RoleConfig `yaml:"roles"`
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

	// Search roles by ARN first.
	for _, roleConfig := range c.Roles {
		if roleConfig.ARN == searchString {
			return &roleConfig
		}
	}

	// Then search by name
	for _, roleConfig := range c.Roles {
		if roleConfig.Name == searchString {
			return &roleConfig
		}
	}

	// Then search by alias.
	for _, roleConfig := range c.Roles {
		for _, alias := range roleConfig.Aliases {
			if strings.ToLower(alias) == strings.ToLower(searchString) {
				return &roleConfig
			}
		}
	}
	return &RoleConfig{}
}

// bootstrapConfig will generate a generic config file, and exit.
func bootstrapConfig(filePath string) {
	exampleConfigYAML, err := yaml.Marshal(Config{
		MFASerial: "arn:aws:iam::000000000000:mfa/your_mfa_serial",
		Roles: []RoleConfig{
			{
				Name:      "one_of_your_accounts",
				IsDefault: true,
				ARN:       "arn:aws:iam::000000000000:role/DeleteOnly",
				Aliases:   []string{"delete", "deleteprod"},
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

// ListRoles outputs a list of configured roles.
func (c *Config) ListRoles() {
	var defaultCount int
	if len(c.Roles) <= 0 {
		fmt.Println("It looks like you haven't got any roles configured!")
		return
	}
	for _, role := range c.Roles {
		if role.IsDefault {
			defaultCount++
		}
		fmt.Println("ARN:", role.ARN)
		fmt.Println("Name:", role.Name)
		if len(role.Aliases) > 0 {
			fmt.Println("Aliases:")
			for _, alias := range role.Aliases {
				fmt.Println("  - " + alias)
			}
		}
		fmt.Println()
	}
	if defaultCount > 1 {
		fmt.Println("Hey, it looks like you've got more than one role flagged as default!")
		fmt.Println("Well use the first one in the list if a role isn't specified via the -role argument.")
	}
}

// GetDefaultRole returns the first role flagged as default.
func (c *Config) GetDefaultRole() *RoleConfig {
	for _, role := range c.Roles {
		if role.IsDefault {
			return &role
		}
	}
	return &RoleConfig{}
}
