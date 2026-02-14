package cli

import (
	"github.com/alecthomas/kong"
	"github.com/semmy-space/zoh/internal/config"
	"github.com/semmy-space/zoh/internal/output"
)

// FormatterProvider wraps the formatter interface for Kong binding
type FormatterProvider struct {
	Formatter output.Formatter
}

// CLI is the root command structure
type CLI struct {
	Globals

	Auth    AuthCmd    `cmd:"" help:"Authentication commands"`
	Config  ConfigCmd  `cmd:"" help:"Configuration commands"`
	Admin   AdminCmd   `cmd:"" help:"Admin operations"`
	Version VersionCmd `cmd:"" help:"Show version information"`
}

// BeforeApply hook runs before any command execution
// It loads config, resolves region, creates formatter, and binds dependencies
func (c *CLI) BeforeApply(ctx *kong.Context) error {
	// Load config from XDG path (returns defaults if missing)
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Resolve region: CLI flag > config > "us" default
	region := c.Region
	if region == "" && cfg.Region != "" {
		region = cfg.Region
	}
	if region == "" {
		region = "us"
	}
	cfg.Region = region

	// Create output formatter
	formatter := &FormatterProvider{
		Formatter: output.New(c.ResolvedOutput()),
	}

	// Bind dependencies to kong context
	ctx.Bind(cfg)
	ctx.Bind(formatter)
	ctx.Bind(&c.Globals)

	return nil
}

// AuthCmd holds authentication subcommands
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Log in to Zoho account"`
	Logout AuthLogoutCmd `cmd:"" help:"Log out and remove stored credentials"`
	List   AuthListCmd   `cmd:"" help:"List stored accounts"`
}

// ConfigCmd holds configuration subcommands
type ConfigCmd struct {
	Get   ConfigGetCmd        `cmd:"" help:"Get a configuration value"`
	Set   ConfigSetCmd        `cmd:"" help:"Set a configuration value"`
	Unset ConfigUnsetCmd      `cmd:"" help:"Remove a configuration value"`
	List  ConfigListConfigCmd `cmd:"" name:"list" help:"List all configuration values"`
	Path  ConfigPathCmd       `cmd:"" help:"Show config file path"`
}

// AdminCmd holds admin subcommands
type AdminCmd struct {
	Users AdminUsersCmd `cmd:"" help:"User operations"`
}

// AdminUsersCmd holds user subcommands
type AdminUsersCmd struct {
	List AdminUsersListCmd `cmd:"" help:"List organization users"`
	Get  AdminUsersGetCmd  `cmd:"" help:"Get user details"`
}

// VersionCmd shows version information
type VersionCmd struct{}

func (cmd *VersionCmd) Run(ctx *kong.Context) error {
	version := ctx.Model.Vars()["version"]
	println("zoh version " + version)
	return nil
}
