package zoho

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"

	"github.com/semmy-space/zoh/internal/config"
)

// AdminClient wraps the Zoho Client with admin-specific functionality
type AdminClient struct {
	client *Client
	zoid   int64 // Cached organization ID
}

// NewAdminClient creates a new AdminClient with the given config and token source
// It automatically resolves and caches the organization ID
func NewAdminClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*AdminClient, error) {
	client, err := NewClient(cfg, tokenSource)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	ac := &AdminClient{
		client: client,
	}

	// Resolve organization ID
	ctx := context.Background()
	zoid, err := ac.getOrganizationID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get organization ID: %w", err)
	}
	ac.zoid = zoid

	return ac, nil
}

// getOrganizationID fetches the organization ID from the Zoho API
func (ac *AdminClient) getOrganizationID(ctx context.Context) (int64, error) {
	resp, err := ac.client.Do(ctx, http.MethodGet, "/api/organization/", nil)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, ac.parseErrorResponse(resp)
	}

	var orgResp OrgResponse
	if err := json.NewDecoder(resp.Body).Decode(&orgResp); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}

	if orgResp.Status.Code != 200 {
		return 0, fmt.Errorf("API error: %s (code %d)", orgResp.Status.Description, orgResp.Status.Code)
	}

	return orgResp.Data.OrganizationID, nil
}

// ListUsers fetches a list of users with pagination
func (ac *AdminClient) ListUsers(ctx context.Context, start, limit int) ([]User, error) {
	path := fmt.Sprintf("/api/organization/%d/accounts?start=%d&limit=%d", ac.zoid, start, limit)
	resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var userResp UserListResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if userResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", userResp.Status.Description, userResp.Status.Code)
	}

	return userResp.Data, nil
}

// GetUser fetches a single user by account ID
func (ac *AdminClient) GetUser(ctx context.Context, accountID int64) (*User, error) {
	path := fmt.Sprintf("/api/organization/%d/accounts/%d", ac.zoid, accountID)
	resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var userResp UserDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if userResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", userResp.Status.Description, userResp.Status.Code)
	}

	return &userResp.Data, nil
}

// GetUserByEmail fetches a user by email address
// This iterates through all users until a match is found
func (ac *AdminClient) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	// Create a page iterator to search through all users
	iterator := NewPageIterator(func(start, limit int) ([]User, error) {
		return ac.ListUsers(ctx, start, limit)
	}, 50)

	// Fetch all users (this will paginate automatically)
	users, err := iterator.FetchAll()
	if err != nil {
		return nil, fmt.Errorf("fetch users: %w", err)
	}

	// Search for matching email
	for _, user := range users {
		if user.EmailAddress == email {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %s", email)
}

// parseErrorResponse attempts to parse an error response from the Zoho API
func (ac *AdminClient) parseErrorResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	var apiErr APIError
	if err := json.Unmarshal(body, &apiErr); err != nil {
		// If we can't parse the error, return the raw body
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// If we successfully parsed an APIError, use its Error() method
	return fmt.Errorf("HTTP %d: %w", resp.StatusCode, &apiErr)
}

// GetUserByIdentifier is a helper that accepts either a ZUID or email address
func (ac *AdminClient) GetUserByIdentifier(ctx context.Context, identifier string) (*User, error) {
	// Try to parse as int64 (ZUID)
	if zuid, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		return ac.GetUser(ctx, zuid)
	}

	// Otherwise, treat as email
	return ac.GetUserByEmail(ctx, identifier)
}
