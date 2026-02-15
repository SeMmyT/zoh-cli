package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/gofrs/flock"
	"github.com/SeMmyT/zohcli/internal/config"
	"github.com/SeMmyT/zohcli/internal/output"
	"github.com/SeMmyT/zohcli/internal/secrets"
	"golang.org/x/oauth2"
)

// TokenCache implements oauth2.TokenSource with file-based caching and file locking.
// It prevents concurrent refresh stampede and auto-refreshes tokens proactively.
type TokenCache struct {
	cachePath    string        // Path to cached access token file
	lockPath     string        // Path to lock file
	store        secrets.Store // Secret store for refresh token
	region       string        // Region code
	clientID     string
	clientSecret string
	accountsURL  string // Region-specific accounts server
}

// cachedToken represents the token structure stored in the cache file.
type cachedToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Expiry      time.Time `json:"expiry"`
}

// NewTokenCache creates a new token cache for the given configuration.
func NewTokenCache(cfg *config.Config, store secrets.Store) (*TokenCache, error) {
	region, err := config.GetRegion(cfg.Region)
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(xdg.CacheHome, "zoh", fmt.Sprintf("token-%s.json", cfg.Region))
	lockPath := cachePath + ".lock"

	// Ensure cache directory exists
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &TokenCache{
		cachePath:    cachePath,
		lockPath:     lockPath,
		store:        store,
		region:       cfg.Region,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		accountsURL:  region.AccountsServer,
	}, nil
}

// Token implements oauth2.TokenSource.Token().
// Returns a valid access token, refreshing if necessary.
func (tc *TokenCache) Token() (*oauth2.Token, error) {
	// Acquire file lock to prevent concurrent refresh
	lock := flock.New(tc.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locked, err := lock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return nil, fmt.Errorf("failed to acquire lock: timeout")
	}
	defer lock.Unlock()

	// Try to read cached access token
	cached, err := tc.readCachedToken()
	if err == nil {
		// Check if token is still valid with 5-minute proactive refresh window
		if time.Until(cached.Expiry) > 5*time.Minute {
			return &oauth2.Token{
				AccessToken: cached.AccessToken,
				TokenType:   cached.TokenType,
				Expiry:      cached.Expiry,
			}, nil
		}
	}

	// Token expired or within proactive refresh window - refresh it
	token, err := tc.refreshToken()
	if err != nil {
		return nil, err
	}

	// Cache the new access token
	if err := tc.writeCachedToken(token); err != nil {
		// Non-fatal - return token anyway
		fmt.Fprintf(os.Stderr, "Warning: failed to cache token: %v\n", err)
	}

	return token, nil
}

// readCachedToken reads the cached access token from disk.
func (tc *TokenCache) readCachedToken() (*cachedToken, error) {
	data, err := os.ReadFile(tc.cachePath)
	if err != nil {
		return nil, err
	}

	var token cachedToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// writeCachedToken writes the access token to the cache file.
func (tc *TokenCache) writeCachedToken(token *oauth2.Token) error {
	cached := cachedToken{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		Expiry:      token.Expiry,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tc.cachePath, data, 0600)
}

// refreshToken exchanges the refresh token for a new access token.
func (tc *TokenCache) refreshToken() (*oauth2.Token, error) {
	// Get refresh token from secrets store
	refreshTokenKey := fmt.Sprintf("refresh_token_%s", tc.region)
	refreshToken, err := tc.store.Get(refreshTokenKey)
	if err != nil {
		if err == secrets.ErrNotFound {
			return nil, &output.CLIError{
				Message:  "No refresh token found. Run: zoh auth login",
				ExitCode: output.ExitAuth,
			}
		}
		return nil, fmt.Errorf("failed to read refresh token: %w", err)
	}

	// POST to Zoho's token endpoint
	tokenURL := tc.accountsURL + "/oauth/v2/token"
	resp, err := http.PostForm(tokenURL, url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {tc.clientID},
		"client_secret": {tc.clientSecret},
		"refresh_token": {refreshToken},
	})
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response — Zoho returns errors as HTTP 200, so always parse first
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token,omitempty"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Error        string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Check for errors (Zoho may return error on HTTP 200 or non-200)
	if tokenResp.Error != "" || resp.StatusCode != http.StatusOK {
		errMsg := tokenResp.Error
		if errMsg == "" {
			errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}

		switch errMsg {
		case "invalid_grant":
			return nil, &output.CLIError{
				Message:  "Refresh token expired or revoked. Run: zoh auth login",
				ExitCode: output.ExitAuth,
			}
		case "invalid_client_secret":
			return nil, &output.CLIError{
				Message:  "Client secret is invalid. Update it with: zoh config set client_secret YOUR_SECRET",
				ExitCode: output.ExitAuth,
			}
		case "invalid_client":
			return nil, &output.CLIError{
				Message:  "Client ID is invalid. Update it with: zoh config set client_id YOUR_CLIENT_ID",
				ExitCode: output.ExitAuth,
			}
		default:
			return nil, fmt.Errorf("token refresh failed: %s", errMsg)
		}
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("token refresh returned empty access token")
	}

	// If a new refresh token was issued, update the secrets store
	if tokenResp.RefreshToken != "" && tokenResp.RefreshToken != refreshToken {
		if err := tc.store.Set(refreshTokenKey, tokenResp.RefreshToken); err != nil {
			// Non-fatal - log warning and continue
			fmt.Fprintf(os.Stderr, "Warning: failed to update refresh token: %v\n", err)
		}
	}

	expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return &oauth2.Token{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
		Expiry:      expiry,
	}, nil
}

// SaveInitialTokens stores the tokens from a successful login.
// This should be called after InteractiveLogin or ManualLogin completes.
func (tc *TokenCache) SaveInitialTokens(token *oauth2.Token) error {
	// Acquire lock
	lock := flock.New(tc.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locked, err := lock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("failed to acquire lock: timeout")
	}
	defer lock.Unlock()

	// Store refresh token in secrets store
	refreshTokenKey := fmt.Sprintf("refresh_token_%s", tc.region)
	if err := tc.store.Set(refreshTokenKey, token.RefreshToken); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Cache access token
	if err := tc.writeCachedToken(token); err != nil {
		return fmt.Errorf("failed to cache access token: %w", err)
	}

	return nil
}

// ClearTokens removes all stored tokens (used by logout).
func (tc *TokenCache) ClearTokens() error {
	// Acquire lock
	lock := flock.New(tc.lockPath)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	locked, err := lock.TryLockContext(ctx, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("failed to acquire lock: timeout")
	}
	defer lock.Unlock()

	// Delete refresh token from secrets store
	refreshTokenKey := fmt.Sprintf("refresh_token_%s", tc.region)
	if err := tc.store.Delete(refreshTokenKey); err != nil && err != secrets.ErrNotFound {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	// Delete cache file
	if err := os.Remove(tc.cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	// Delete lock file
	if err := os.Remove(tc.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete lock file: %w", err)
	}

	return nil
}
