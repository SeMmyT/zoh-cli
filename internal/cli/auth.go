package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/SeMmyT/zohcli/internal/auth"
	"github.com/SeMmyT/zohcli/internal/config"
	"github.com/SeMmyT/zohcli/internal/output"
	"github.com/SeMmyT/zohcli/internal/secrets"
)

// AuthLoginCmd implements the auth login command
type AuthLoginCmd struct {
	Manual bool `help:"Manual paste mode (no browser)" short:"m"`
}

// Run executes the login command
func (cmd *AuthLoginCmd) Run(cfg *config.Config, fp *FormatterProvider, globals *Globals) error {
	// Validate required config
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return &output.CLIError{
			Message: "Client ID and Client Secret required.\n\n" +
				"Run: zoh config set client_id YOUR_CLIENT_ID\n" +
				"Run: zoh config set client_secret YOUR_CLIENT_SECRET\n\n" +
				"Get credentials at: https://api-console.zoho.com/",
			ExitCode: output.ExitConfigError,
		}
	}

	// Note: region is already set in cfg via BeforeApply hook (global --region flag > config > "us")

	// Initialize secrets store and token cache
	store, err := secrets.NewStore()
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize secrets store: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	tokenCache, err := auth.NewTokenCache(cfg, store)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize token cache: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// Execute login flow
	ctx := context.Background()
	var token *oauth2.Token

	if cmd.Manual {
		token, err = auth.ManualLogin(ctx, cfg)
	} else {
		token, err = auth.InteractiveLogin(ctx, cfg)
	}

	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Login failed: %v", err),
			ExitCode: output.ExitAuth,
		}
	}

	// Save tokens
	if err := tokenCache.SaveInitialTokens(token); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to save tokens: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// Persist region to config if it was set via global flag
	if globals.Region != "" {
		cfg.Region = globals.Region
		if err := cfg.Save(); err != nil {
			// Non-fatal - log warning
			fmt.Fprintf(os.Stderr, "Warning: failed to save region to config: %v\n", err)
		}
	}

	// Output success
	fmt.Fprintf(os.Stderr, "✓ Authenticated successfully\n")
	fmt.Fprintf(os.Stderr, "Region: %s\n", cfg.Region)
	fmt.Fprintf(os.Stderr, "Token expires: %s\n", token.Expiry.Format(time.RFC3339))

	// Detect storage backend
	storageType := "keyring"
	if secrets.IsWSL() || secrets.IsHeadless() {
		storageType = "encrypted file"
	}
	fmt.Fprintf(os.Stderr, "Credentials stored in %s\n", storageType)

	return nil
}

// AuthLogoutCmd implements the auth logout command
type AuthLogoutCmd struct {
	All bool `help:"Remove all stored accounts" short:"a"`
}

// Run executes the logout command
func (cmd *AuthLogoutCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	store, err := secrets.NewStore()
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize secrets store: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	if cmd.All {
		// Clear tokens for all regions
		regions := []string{"us", "eu", "in", "au", "jp", "ca", "sa", "uk"}
		for _, region := range regions {
			// Create temp config for this region
			tempCfg := &config.Config{
				Region:       region,
				ClientID:     cfg.ClientID,
				ClientSecret: cfg.ClientSecret,
			}
			tokenCache, err := auth.NewTokenCache(tempCfg, store)
			if err != nil {
				continue // Skip regions with errors
			}
			tokenCache.ClearTokens() // Ignore errors for missing tokens
		}
		fmt.Fprintf(os.Stderr, "Logged out all accounts\n")
	} else {
		// Clear tokens for current region
		tokenCache, err := auth.NewTokenCache(cfg, store)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to initialize token cache: %v", err),
				ExitCode: output.ExitGeneral,
			}
		}

		if err := tokenCache.ClearTokens(); err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to clear tokens: %v", err),
				ExitCode: output.ExitGeneral,
			}
		}

		fmt.Fprintf(os.Stderr, "Logged out from region: %s\n", cfg.Region)
	}

	fmt.Fprintf(os.Stderr, "Credentials removed\n")
	return nil
}

// AuthListCmd implements the auth list command
type AuthListCmd struct {
	Check bool `help:"Validate stored tokens are still valid" short:"c"`
}

// Run executes the list command
func (cmd *AuthListCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	store, err := secrets.NewStore()
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize secrets store: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// List all stored refresh tokens
	items, err := store.List()
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to list stored credentials: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// Parse refresh token keys to extract regions
	type accountInfo struct {
		Region string
		Active bool
		Valid  string // "yes", "no", "unknown" (if not checked)
		Expiry string
	}

	var accounts []accountInfo

	for _, item := range items {
		if strings.HasPrefix(item, "refresh_token_") {
			region := strings.TrimPrefix(item, "refresh_token_")

			account := accountInfo{
				Region: region,
				Active: region == cfg.Region,
				Valid:  "unknown",
				Expiry: "n/a",
			}

			// Optionally check if token is valid
			if cmd.Check {
				tempCfg := &config.Config{
					Region:       region,
					ClientID:     cfg.ClientID,
					ClientSecret: cfg.ClientSecret,
				}
				tokenCache, err := auth.NewTokenCache(tempCfg, store)
				if err != nil {
					account.Valid = "error"
				} else {
					token, err := tokenCache.Token()
					if err != nil {
						account.Valid = "no"
					} else {
						account.Valid = "yes"
						account.Expiry = token.Expiry.Format(time.RFC3339)
					}
				}
			}

			accounts = append(accounts, account)
		}
	}

	if len(accounts) == 0 {
		fmt.Fprintf(os.Stderr, "No stored accounts found\n")
		fmt.Fprintf(os.Stderr, "Run 'zoh auth login' to authenticate\n")
		return nil
	}

	// Output as list using PrintList
	cols := []output.Column{
		{Name: "Region", Key: "Region"},
		{Name: "Status", Key: "Status"},
	}
	if cmd.Check {
		cols = append(cols,
			output.Column{Name: "Valid", Key: "Valid"},
			output.Column{Name: "Token Expiry", Key: "Expiry"},
		)
	}

	fp.Formatter.PrintList(accounts, cols)
	return nil
}
