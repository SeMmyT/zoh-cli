package cli

import (
	"bufio"
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

// SetupCmd implements the interactive setup wizard
type SetupCmd struct{}

// Run executes the setup wizard
func (cmd *SetupCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  zoh — Zoho CLI Setup\n")
	fmt.Fprintf(os.Stderr, "  ====================\n\n")

	// Step 1: Region
	fmt.Fprintf(os.Stderr, "  Step 1: Choose your Zoho data center\n\n")
	fmt.Fprintf(os.Stderr, "    us  United States    eu  Europe\n")
	fmt.Fprintf(os.Stderr, "    in  India            au  Australia\n")
	fmt.Fprintf(os.Stderr, "    jp  Japan            ca  Canada\n")
	fmt.Fprintf(os.Stderr, "    sa  Saudi Arabia     uk  United Kingdom\n\n")

	current := cfg.Region
	if current == "" {
		current = "us"
	}
	region := prompt(reader, fmt.Sprintf("  Region [%s]: ", current))
	if region == "" {
		region = current
	}
	if _, err := config.GetRegion(region); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid region: %s. Valid: us, eu, in, au, jp, ca, sa, uk", region),
			ExitCode: output.ExitUsage,
		}
	}

	// Step 2: API credentials
	regionCfg, _ := config.GetRegion(region)
	consoleURL := strings.Replace(regionCfg.AccountsServer, "accounts.", "api-console.", 1)

	fmt.Fprintf(os.Stderr, "\n  Step 2: Create a Zoho API client\n\n")
	fmt.Fprintf(os.Stderr, "    1. Go to %s\n", consoleURL)
	fmt.Fprintf(os.Stderr, "    2. Click \"Add Client\" > \"Server-based Applications\"\n")
	fmt.Fprintf(os.Stderr, "    3. Set the redirect URI to: http://localhost:8080/callback\n")
	fmt.Fprintf(os.Stderr, "    4. Copy the Client ID and Client Secret below\n\n")

	clientID := prompt(reader, "  Client ID: ")
	if clientID == "" && cfg.ClientID != "" {
		clientID = cfg.ClientID
		fmt.Fprintf(os.Stderr, "  (keeping existing)\n")
	}
	if clientID == "" {
		return &output.CLIError{
			Message:  "Client ID is required",
			ExitCode: output.ExitUsage,
		}
	}

	clientSecret := prompt(reader, "  Client Secret: ")
	if clientSecret == "" && cfg.ClientSecret != "" {
		clientSecret = cfg.ClientSecret
		fmt.Fprintf(os.Stderr, "  (keeping existing)\n")
	}
	if clientSecret == "" {
		return &output.CLIError{
			Message:  "Client Secret is required",
			ExitCode: output.ExitUsage,
		}
	}

	// Save config before login
	cfg.Region = region
	cfg.ClientID = clientID
	cfg.ClientSecret = clientSecret
	if err := cfg.Save(); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to save config: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// Step 3: Login
	fmt.Fprintf(os.Stderr, "\n  Step 3: Authenticate\n\n")

	answer := prompt(reader, "  Open browser to log in? [Y/n]: ")
	manual := strings.ToLower(answer) == "n"

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

	ctx := context.Background()
	var token *oauth2.Token

	if manual {
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

	if err := tokenCache.SaveInitialTokens(token); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to save tokens: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// Done
	storageType := "keyring"
	if secrets.IsWSL() || secrets.IsHeadless() {
		storageType = "encrypted file"
	}

	fmt.Fprintf(os.Stderr, "\n  Setup complete!\n\n")
	fmt.Fprintf(os.Stderr, "    Region:      %s\n", region)
	fmt.Fprintf(os.Stderr, "    Credentials: %s\n", storageType)
	fmt.Fprintf(os.Stderr, "    Expires:     %s\n", token.Expiry.Format(time.RFC3339))
	fmt.Fprintf(os.Stderr, "    Config:      %s\n\n", config.ConfigPath())
	fmt.Fprintf(os.Stderr, "  Try it out:\n\n")
	fmt.Fprintf(os.Stderr, "    zoh admin users list\n")
	fmt.Fprintf(os.Stderr, "    zoh mail messages list\n")
	fmt.Fprintf(os.Stderr, "    zoh mail messages search --unread\n\n")

	return nil
}

// prompt prints a prompt and reads a line of input
func prompt(reader *bufio.Reader, text string) string {
	fmt.Fprint(os.Stderr, text)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// NeedsSetup returns true if the CLI has not been configured yet
func NeedsSetup(cfg *config.Config) bool {
	return cfg.ClientID == "" || cfg.ClientSecret == ""
}

// PrintSetupHint prints a hint to run setup
func PrintSetupHint() {
	fmt.Fprintf(os.Stderr, "\n  zoh is not configured yet. Run:\n\n")
	fmt.Fprintf(os.Stderr, "    zoh setup\n\n")
}
