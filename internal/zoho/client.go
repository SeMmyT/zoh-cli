package zoho

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/time/rate"

	"github.com/SeMmyT/zoh/internal/config"
)

// Client is a region-aware HTTP client for Zoho APIs.
type Client struct {
	httpClient  *http.Client
	region      config.RegionConfig
	rateLimiter *rate.Limiter
}

// NewClient creates a new Zoho API client with OAuth2 authentication and rate limiting.
func NewClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*Client, error) {
	// Get region configuration
	regionConfig, err := config.GetRegion(cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("invalid region %q: %w", cfg.Region, err)
	}

	// Create rate limiter
	rateLimiter := NewRateLimiter()

	// Build transport chain:
	// 1. Base transport (http.DefaultTransport)
	// 2. OAuth2 transport (adds Authorization: Bearer header)
	// 3. Rate limit transport (enforces rate limits and handles 429)
	baseTransport := http.DefaultTransport
	oauth2Transport := &oauth2.Transport{
		Base:   baseTransport,
		Source: tokenSource,
	}
	rateLimitTransport := &RateLimitTransport{
		Base:    oauth2Transport,
		Limiter: rateLimiter,
	}

	// Create HTTP client with timeouts
	httpClient := &http.Client{
		Transport: rateLimitTransport,
		Timeout:   30 * time.Second,
	}

	return &Client{
		httpClient:  httpClient,
		region:      regionConfig,
		rateLimiter: rateLimiter,
	}, nil
}

// Do executes an HTTP request to the Zoho API.
// The path should be relative (e.g., "/admin/v1/users").
// The caller is responsible for reading the response body and checking the status code.
func (c *Client) Do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.region.APIBase + path
	return c.doRequest(ctx, method, url, body)
}

// DoMail executes an HTTP request to the Zoho Mail API.
// The path should be relative (e.g., "/api/accounts/123/messages").
// Used for mail-specific operations in Phase 4+.
func (c *Client) DoMail(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.region.MailBase + path
	return c.doRequest(ctx, method, url, body)
}

// DoAuth executes an HTTP request to the Zoho Accounts API.
// The path should be relative (e.g., "/oauth/v2/token/info").
// Does NOT add OAuth2 bearer token (auth endpoints use different auth methods).
func (c *Client) DoAuth(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	url := c.region.AccountsServer + path

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set content type for POST/PUT requests
	if body != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute without OAuth2 token (use base client without oauth2.Transport)
	// Auth endpoints use client_id/client_secret or token introspection
	baseClient := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   30 * time.Second,
	}
	return baseClient.Do(req)
}

// doRequest is the common request execution logic for Do and DoMail.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set content type for POST/PUT/PATCH requests
	if body != nil && (method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch) {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute via HTTP client (goes through OAuth2 + rate limit transports)
	return c.httpClient.Do(req)
}
