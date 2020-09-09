package config

// RoleConfig represents a single mapping of an account role to assume.
type RoleConfig struct {
	Name    string   `yaml:"name"`
	ARN     string   `yaml:"arn"`
	Aliases []string `yaml:"aliases"`
}
