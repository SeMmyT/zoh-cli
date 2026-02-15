package auth

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/SeMmyT/zohcli/internal/config"
	"github.com/SeMmyT/zohcli/internal/output"
	"github.com/SeMmyT/zohcli/pkg/browser"
	"golang.org/x/oauth2"
)

// newOAuth2Config creates an oauth2.Config for Zoho authentication.
func newOAuth2Config(cfg *config.Config, redirectURL string) (*oauth2.Config, error) {
	region, err := config.GetRegion(cfg.Region)
	if err != nil {
		return nil, err
	}

	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   region.AccountsServer + "/oauth/v2/auth",
			TokenURL:  region.AccountsServer + "/oauth/v2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: redirectURL,
		// Note: Scopes are NOT set here because Zoho uses comma-separated scopes,
		// but oauth2.Config uses space-separated. We'll add scopes manually to the auth URL.
	}, nil
}

// generateState generates a random state parameter for OAuth2 CSRF protection.
func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// InteractiveLogin performs an OAuth2 login flow using the browser.
// Opens the authorization URL in the default browser and starts a local callback server.
func InteractiveLogin(ctx context.Context, cfg *config.Config) (*oauth2.Token, error) {
	// Validate config
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, &output.CLIError{
			Message:  "Client ID and Client Secret required. Run: zoh config set client_id <id> && zoh config set client_secret <secret>",
			ExitCode: output.ExitConfigError,
		}
	}

	// Start callback server (port 0 = auto-select available port)
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	resultChan, callbackURL, shutdown := startCallbackServer(ctx, 0)
	defer shutdown()

	oauthCfg, err := newOAuth2Config(cfg, callbackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 config: %w", err)
	}

	state := generateState()

	// Build auth URL with Zoho-specific parameters
	// Zoho requires comma-separated scopes and access_type=offline for refresh token
	authURL := oauthCfg.Endpoint.AuthURL + "?" + url.Values{
		"client_id":     {oauthCfg.ClientID},
		"redirect_uri":  {oauthCfg.RedirectURL},
		"response_type": {"code"},
		"scope":         {ScopeString()}, // Comma-separated
		"state":         {state},
		"access_type":   {"offline"}, // Required for refresh token
		"prompt":        {"consent"},  // Force consent to ensure refresh token
	}.Encode()

	// Print auth URL as fallback in case browser doesn't open
	fmt.Fprintf(os.Stderr, "Opening browser for authentication...\n")
	fmt.Fprintf(os.Stderr, "If the browser doesn't open, visit this URL:\n%s\n\n", authURL)

	// Try to open browser
	if err := browser.Open(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open browser: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please visit the URL above manually.\n")
	}

	// Wait for callback
	var result callbackResult
	select {
	case result = <-resultChan:
	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timeout (5 minutes)")
	}

	if result.Error != "" {
		return nil, fmt.Errorf("authentication failed: %s", result.Error)
	}

	// Validate state parameter (CSRF protection)
	if result.State != state {
		return nil, fmt.Errorf("state mismatch (possible CSRF attack)")
	}

	// Exchange authorization code for tokens
	token, err := oauthCfg.Exchange(ctx, result.Code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	return token, nil
}

// ManualLogin performs an OAuth2 login flow by printing the auth URL and accepting a pasted redirect.
// This is useful for environments where the browser can't be opened automatically (SSH, headless, etc).
func ManualLogin(ctx context.Context, cfg *config.Config) (*oauth2.Token, error) {
	// Validate config
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, &output.CLIError{
			Message:  "Client ID and Client Secret required. Run: zoh config set client_id <id> && zoh config set client_secret <secret>",
			ExitCode: output.ExitConfigError,
		}
	}

	// Use fixed redirect URL for manual flow (user will paste the redirected URL)
	redirectURL := "http://localhost:8080/callback"

	oauthCfg, err := newOAuth2Config(cfg, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth2 config: %w", err)
	}

	state := generateState()

	// Build auth URL with Zoho-specific parameters
	authURL := oauthCfg.Endpoint.AuthURL + "?" + url.Values{
		"client_id":     {oauthCfg.ClientID},
		"redirect_uri":  {oauthCfg.RedirectURL},
		"response_type": {"code"},
		"scope":         {ScopeString()}, // Comma-separated
		"state":         {state},
		"access_type":   {"offline"}, // Required for refresh token
		"prompt":        {"consent"},  // Force consent to ensure refresh token
	}.Encode()

	// Print instructions
	fmt.Fprintf(os.Stderr, "\n=== Manual OAuth2 Flow ===\n\n")
	fmt.Fprintf(os.Stderr, "1. Visit this URL in your browser:\n\n")
	fmt.Fprintf(os.Stderr, "%s\n\n", authURL)
	fmt.Fprintf(os.Stderr, "2. After authorizing, you'll be redirected to a page that won't load.\n")
	fmt.Fprintf(os.Stderr, "3. Copy the FULL URL from your browser's address bar and paste it here.\n\n")
	fmt.Fprintf(os.Stderr, "Paste the redirect URL: ")

	// Read pasted URL from stdin
	reader := bufio.NewReader(os.Stdin)
	redirectedURL, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}
	redirectedURL = strings.TrimSpace(redirectedURL)

	// Parse the URL to extract code and state
	parsedURL, err := url.Parse(redirectedURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	code := parsedURL.Query().Get("code")
	returnedState := parsedURL.Query().Get("state")

	if code == "" {
		errorMsg := parsedURL.Query().Get("error")
		if errorMsg != "" {
			return nil, fmt.Errorf("authorization failed: %s", errorMsg)
		}
		return nil, fmt.Errorf("no authorization code found in URL")
	}

	// Validate state parameter (CSRF protection)
	if returnedState != state {
		return nil, fmt.Errorf("state mismatch (possible CSRF attack)")
	}

	// Exchange authorization code for tokens
	token, err := oauthCfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nAuthentication successful!\n")
	return token, nil
}
