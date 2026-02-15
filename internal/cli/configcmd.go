package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/SeMmyT/zoh/internal/config"
	"github.com/SeMmyT/zoh/internal/output"
)

// ConfigGetCmd implements config get command
type ConfigGetCmd struct {
	Key string `arg:"" help:"Config key to get (e.g., region, client_id)"`
}

// Run executes the get command
func (cmd *ConfigGetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	value, err := cfg.Get(cmd.Key)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Unknown config key: %s", cmd.Key),
			ExitCode: output.ExitNotFound,
		}
	}

	// Print value via formatter (respects --output flag)
	fp.Formatter.Print(value)
	return nil
}

// ConfigSetCmd implements config set command
type ConfigSetCmd struct {
	Key   string `arg:"" help:"Config key to set"`
	Value string `arg:"" help:"Value to set"`
}

// Run executes the set command
func (cmd *ConfigSetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	// Validate key exists
	if _, err := cfg.Get(cmd.Key); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Unknown config key: %s", cmd.Key),
			ExitCode: output.ExitUsage,
		}
	}

	// Special validation for region
	if cmd.Key == "region" {
		validRegions := []string{"us", "eu", "in", "au", "jp", "ca", "sa", "uk"}
		valid := false
		for _, r := range validRegions {
			if cmd.Value == r {
				valid = true
				break
			}
		}
		if !valid {
			return &output.CLIError{
				Message:  fmt.Sprintf("Invalid region: %s. Valid regions: %s", cmd.Value, strings.Join(validRegions, ", ")),
				ExitCode: output.ExitUsage,
			}
		}
	}

	// Hint for client_secret
	if cmd.Key == "client_secret" {
		fmt.Fprintf(os.Stderr, "Note: client_secret is stored in config file. For better security, consider using the secrets store.\n")
	}

	// Set and save
	if err := cfg.Set(cmd.Key, cmd.Value); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to set config: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	fmt.Fprintf(os.Stderr, "Set %s = %s\n", cmd.Key, cmd.Value)
	return nil
}

// ConfigUnsetCmd implements config unset command
type ConfigUnsetCmd struct {
	Key string `arg:"" help:"Config key to remove"`
}

// Run executes the unset command
func (cmd *ConfigUnsetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	// Validate key exists
	if _, err := cfg.Get(cmd.Key); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Unknown config key: %s", cmd.Key),
			ExitCode: output.ExitUsage,
		}
	}

	if err := cfg.Unset(cmd.Key); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to unset config: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	fmt.Fprintf(os.Stderr, "Unset %s\n", cmd.Key)
	return nil
}

// ConfigListConfigCmd implements config list command
type ConfigListConfigCmd struct{}

// Run executes the list command
func (cmd *ConfigListConfigCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	// Build list of config key-value pairs
	type ConfigItem struct {
		Key   string
		Value string
	}

	items := []ConfigItem{
		{Key: "region", Value: cfg.Region},
		{Key: "client_id", Value: cfg.ClientID},
		{Key: "client_secret", Value: maskSecret(cfg.ClientSecret)},
		{Key: "org_id", Value: cfg.OrgID},
		{Key: "account_id", Value: cfg.AccountID},
		{Key: "default_output", Value: cfg.DefaultOutput},
	}

	cols := []output.Column{
		{Name: "Key", Key: "Key"},
		{Name: "Value", Key: "Value"},
	}

	fp.Formatter.PrintList(items, cols)
	return nil
}

// maskSecret masks sensitive values, showing only last 4 characters
func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}

// ConfigPathCmd implements config path command
type ConfigPathCmd struct{}

// Run executes the path command
func (cmd *ConfigPathCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	path := config.ConfigPath()

	// Print path via formatter (respects --output flag)
	fp.Formatter.Print(path)

	// Print existence hint to stderr
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "(file does not exist yet - will be created on first write)\n")
	} else {
		fmt.Fprintf(os.Stderr, "(file exists)\n")
	}

	return nil
}
