package zoho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"

	"github.com/semmy-space/zoh/internal/config"
)

// MailAdminClient wraps the Zoho Client with mail admin-specific functionality
type MailAdminClient struct {
	client *Client
	zoid   string // Cached organization ID (string for URL construction)
}

// NewMailAdminClient creates a new MailAdminClient with the given config and token source
// It automatically resolves and caches the organization ID
func NewMailAdminClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*MailAdminClient, error) {
	client, err := NewClient(cfg, tokenSource)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	mac := &MailAdminClient{
		client: client,
	}

	// Resolve organization ID
	ctx := context.Background()
	zoid, err := mac.getOrganizationID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get organization ID: %w", err)
	}
	mac.zoid = zoid

	return mac, nil
}

// getOrganizationID fetches the organization ID from the Zoho API
// Note: Organization endpoint uses APIBase (client.Do), not MailBase
func (mac *MailAdminClient) getOrganizationID(ctx context.Context) (string, error) {
	resp, err := mac.client.Do(ctx, http.MethodGet, "/api/organization/", nil)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", mac.parseErrorResponse(resp)
	}

	var orgResp OrgResponse
	if err := json.NewDecoder(resp.Body).Decode(&orgResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if orgResp.Status.Code != 200 {
		return "", fmt.Errorf("API error: %s (code %d)", orgResp.Status.Description, orgResp.Status.Code)
	}

	return fmt.Sprintf("%d", orgResp.Data.OrganizationID), nil
}

// parseErrorResponse attempts to parse an error response from the Zoho API
func (mac *MailAdminClient) parseErrorResponse(resp *http.Response) error {
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

// GetSpamSettings fetches spam settings for a given category
// NOTE: Research confidence is MEDIUM for this endpoint - may not be supported
func (mac *MailAdminClient) GetSpamSettings(ctx context.Context, category SpamCategory) ([]string, error) {
	path := fmt.Sprintf("/api/organization/%s/antispam/data?spamCategory=%s",
		mac.zoid, url.QueryEscape(string(category)))

	resp, err := mac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mac.parseErrorResponse(resp)
	}

	var spamResp SpamSettingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&spamResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if spamResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", spamResp.Status.Description, spamResp.Status.Code)
	}

	return spamResp.Data, nil
}

// UpdateSpamList updates spam settings for a given category
func (mac *MailAdminClient) UpdateSpamList(ctx context.Context, category SpamCategory, values []string) error {
	path := fmt.Sprintf("/api/organization/%s/antispam/data", mac.zoid)

	req := SpamUpdateRequest{
		SpamCategory: string(category),
		Value:        values,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := mac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return mac.parseErrorResponse(resp)
	}

	var apiResp struct {
		Status struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
		} `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if apiResp.Status.Code != 200 {
		return fmt.Errorf("API error: %s (code %d)", apiResp.Status.Description, apiResp.Status.Code)
	}

	return nil
}

// GetRetentionPolicy fetches retention policy settings
// Returns raw JSON since policy structure is not well-documented
func (mac *MailAdminClient) GetRetentionPolicy(ctx context.Context) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/organization/%s/mailpolicy/retention", mac.zoid)

	resp, err := mac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mac.parseErrorResponse(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Validate it's valid JSON
	var test interface{}
	if err := json.Unmarshal(body, &test); err != nil {
		return nil, fmt.Errorf("invalid JSON response: %w", err)
	}

	return json.RawMessage(body), nil
}

// GetDeliveryLogs fetches delivery logs with pagination
func (mac *MailAdminClient) GetDeliveryLogs(ctx context.Context, start, limit int) ([]DeliveryLog, error) {
	path := fmt.Sprintf("/api/organization/%s/deliverylog?start=%d&limit=%d",
		mac.zoid, start, limit)

	resp, err := mac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mac.parseErrorResponse(resp)
	}

	var logResp DeliveryLogListResponse
	if err := json.NewDecoder(resp.Body).Decode(&logResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if logResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", logResp.Status.Description, logResp.Status.Code)
	}

	return logResp.Data, nil
}
