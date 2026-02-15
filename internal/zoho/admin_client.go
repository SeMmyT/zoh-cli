package zoho

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"

	"github.com/SeMmyT/zoh/internal/config"
)

// AdminClient wraps the Zoho Client with admin-specific functionality
type AdminClient struct {
	client    *Client
	zoid      int64  // Cached organization ID
	accountID string // Cached account ID
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

	// Resolve organization ID and account ID via /api/accounts
	ctx := context.Background()
	zoid, accountID, err := ac.resolveIdentity(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve identity: %w", err)
	}
	ac.zoid = zoid
	ac.accountID = accountID

	return ac, nil
}

// resolveIdentity fetches the organization ID and account ID from /api/accounts.
// Uses ZohoMail.accounts scope (not the partner-only /api/organization endpoint).
func (ac *AdminClient) resolveIdentity(ctx context.Context) (zoid int64, accountID string, err error) {
	resp, err := ac.client.DoMail(ctx, http.MethodGet, "/api/accounts", nil)
	if err != nil {
		return 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", ac.parseErrorResponse(resp)
	}

	var acctResp AccountsResponse
	if err := json.NewDecoder(resp.Body).Decode(&acctResp); err != nil {
		return 0, "", fmt.Errorf("decode response: %w", err)
	}

	if acctResp.Status.Code != 200 {
		return 0, "", fmt.Errorf("API error: %s (code %d)", acctResp.Status.Description, acctResp.Status.Code)
	}

	if len(acctResp.Data) == 0 {
		return 0, "", fmt.Errorf("no accounts found")
	}

	acct := acctResp.Data[0]
	if acct.PolicyID.Zoid == 0 {
		return 0, "", fmt.Errorf("organization ID not found in account data")
	}

	return acct.PolicyID.Zoid, acct.AccountID, nil
}

// ListUsers fetches a list of users with pagination
func (ac *AdminClient) ListUsers(ctx context.Context, start, limit int) ([]User, error) {
	path := fmt.Sprintf("/api/organization/%d/accounts?start=%d&limit=%d", ac.zoid, start, limit)
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
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
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
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
		if user.PrimaryEmail() == email {
			return &user, nil
		}
		// Also check all email addresses (aliases)
		for _, ea := range user.EmailAddress {
			if ea.MailID == email {
				return &user, nil
			}
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

// CreateUser creates a new user in the organization
func (ac *AdminClient) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
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

// UpdateUserRole changes a user's role in the organization
func (ac *AdminClient) UpdateUserRole(ctx context.Context, zuid int64, newRole string) error {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

	req := UpdateUserRequest{
		Mode:    "changeRole",
		ZUID:    zuid,
		NewRole: newRole,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// EnableUser activates a user account
func (ac *AdminClient) EnableUser(ctx context.Context, zuid int64) error {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

	req := UpdateUserRequest{
		Mode: "enableUser",
		ZUID: zuid,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// DisableUser deactivates a user account with specified options
func (ac *AdminClient) DisableUser(ctx context.Context, zuid int64, opts DisableUserOpts) error {
	path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)

	// Build request body with mode, zuid, and option fields
	reqBody := map[string]interface{}{
		"mode": "disableUser",
		"zuid": zuid,
	}

	if opts.BlockIncoming {
		reqBody["blockIncoming"] = true
	}
	if opts.RemoveMailForward {
		reqBody["removeMailForward"] = true
	}
	if opts.RemoveGroupMembership {
		reqBody["removeGroupMembership"] = true
	}
	if opts.RemoveAlias {
		reqBody["removeAlias"] = true
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// DeleteUser permanently deletes a user from the organization
func (ac *AdminClient) DeleteUser(ctx context.Context, zuid int64) error {
	path := fmt.Sprintf("/api/organization/%d/accounts/%d", ac.zoid, zuid)

	resp, err := ac.client.DoMail(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// ListGroups fetches a list of groups with pagination
func (ac *AdminClient) ListGroups(ctx context.Context, start, limit int) ([]Group, error) {
	path := fmt.Sprintf("/api/organization/%d/groups?start=%d&limit=%d", ac.zoid, start, limit)
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var groupResp GroupListResponse
	if err := json.NewDecoder(resp.Body).Decode(&groupResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if groupResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", groupResp.Status.Description, groupResp.Status.Code)
	}

	return groupResp.Data.Groups, nil
}

// GetGroup fetches a single group by ZGID
func (ac *AdminClient) GetGroup(ctx context.Context, zgid int64) (*Group, error) {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var groupResp GroupDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&groupResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if groupResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", groupResp.Status.Description, groupResp.Status.Code)
	}

	return &groupResp.Data, nil
}

// GetGroupMembers fetches the member list for a group
func (ac *AdminClient) GetGroupMembers(ctx context.Context, zgid int64) ([]GroupMember, error) {
	path := fmt.Sprintf("/api/organization/%d/groups/%d/members", ac.zoid, zgid)
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var membersResp GroupMembersResponse
	if err := json.NewDecoder(resp.Body).Decode(&membersResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if membersResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", membersResp.Status.Description, membersResp.Status.Code)
	}

	return membersResp.Data, nil
}

// GetGroupByEmail fetches a group by email address
// This iterates through all groups until a match is found
func (ac *AdminClient) GetGroupByEmail(ctx context.Context, email string) (*Group, error) {
	// Create a page iterator to search through all groups
	iterator := NewPageIterator(func(start, limit int) ([]Group, error) {
		return ac.ListGroups(ctx, start, limit)
	}, 50)

	// Fetch all groups (this will paginate automatically)
	groups, err := iterator.FetchAll()
	if err != nil {
		return nil, fmt.Errorf("fetch groups: %w", err)
	}

	// Search for matching email
	for _, group := range groups {
		if group.GroupEmailAddress == email {
			return &group, nil
		}
	}

	return nil, fmt.Errorf("group not found: %s", email)
}

// CreateGroup creates a new group
func (ac *AdminClient) CreateGroup(ctx context.Context, req CreateGroupRequest) (*Group, error) {
	path := fmt.Sprintf("/api/organization/%d/groups", ac.zoid)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, ac.parseErrorResponse(resp)
	}

	var groupResp GroupDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&groupResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if groupResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", groupResp.Status.Description, groupResp.Status.Code)
	}

	return &groupResp.Data, nil
}

// UpdateGroup updates a group's name and/or description
func (ac *AdminClient) UpdateGroup(ctx context.Context, zgid int64, name, description string) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)

	// Build request body with only non-empty fields
	reqBody := map[string]interface{}{
		"mode": "updateMailGroup",
	}
	if name != "" {
		reqBody["groupName"] = name
	}
	if description != "" {
		reqBody["description"] = description
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// DeleteGroup permanently deletes a group
func (ac *AdminClient) DeleteGroup(ctx context.Context, zgid int64) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)
	resp, err := ac.client.DoMail(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// AddGroupMembers adds members to a group with batching for large member lists
func (ac *AdminClient) AddGroupMembers(ctx context.Context, zgid int64, members []GroupMemberToAdd) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)

	// Batch members in groups of 50 to stay within API limits
	batchSize := 50
	for i := 0; i < len(members); i += batchSize {
		end := i + batchSize
		if end > len(members) {
			end = len(members)
		}

		batch := members[i:end]
		req := AddGroupMembersRequest{
			Mode:                "addMailGroupMember",
			MailGroupMemberList: batch,
		}

		body, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}

		resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return ac.parseErrorResponse(resp)
		}
	}

	return nil
}

// RemoveGroupMembers removes members from a group
func (ac *AdminClient) RemoveGroupMembers(ctx context.Context, zgid int64, members []GroupMemberToRemove) error {
	path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)

	req := RemoveGroupMembersRequest{
		Mode:                "removeMailGroupMember",
		MailGroupMemberList: members,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// ListDomains fetches all domains in the organization
func (ac *AdminClient) ListDomains(ctx context.Context) ([]Domain, error) {
	path := fmt.Sprintf("/api/organization/%d/domains", ac.zoid)
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var domainResp DomainListResponse
	if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if domainResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", domainResp.Status.Description, domainResp.Status.Code)
	}

	return domainResp.Data.DomainVO, nil
}

// GetDomain fetches details for a specific domain
func (ac *AdminClient) GetDomain(ctx context.Context, domainName string) (*Domain, error) {
	path := fmt.Sprintf("/api/organization/%d/domains/%s", ac.zoid, domainName)
	resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ac.parseErrorResponse(resp)
	}

	var domainResp DomainDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if domainResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", domainResp.Status.Description, domainResp.Status.Code)
	}

	return &domainResp.Data, nil
}

// AddDomain adds a new domain to the organization
func (ac *AdminClient) AddDomain(ctx context.Context, domainName string) (*Domain, error) {
	path := fmt.Sprintf("/api/organization/%d/domains", ac.zoid)

	req := AddDomainRequest{
		DomainName: domainName,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, ac.parseErrorResponse(resp)
	}

	var domainResp DomainDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if domainResp.Status.Code != 200 {
		return nil, fmt.Errorf("API error: %s (code %d)", domainResp.Status.Description, domainResp.Status.Code)
	}

	return &domainResp.Data, nil
}

// VerifyDomain verifies domain ownership using the specified method
func (ac *AdminClient) VerifyDomain(ctx context.Context, domainName, method string) error {
	// Validate method
	validMethods := map[string]bool{
		"verifyDomainByTXT":   true,
		"verifyDomainByCName": true,
		"verifyDomainByHTML":  true,
	}
	if !validMethods[method] {
		return fmt.Errorf("invalid verification method: %s (must be verifyDomainByTXT, verifyDomainByCName, or verifyDomainByHTML)", method)
	}

	path := fmt.Sprintf("/api/organization/%d/domains/%s", ac.zoid, domainName)

	req := DomainModeRequest{
		Mode: method,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// UpdateDomainSettings updates domain settings using the specified mode
func (ac *AdminClient) UpdateDomainSettings(ctx context.Context, domainName, mode string) error {
	// Validate mode
	validModes := map[string]bool{
		"enableHosting":  true,
		"disableHosting": true,
		"setPrimary":     true,
		"enableDkim":     true,
		"disableDkim":    true,
	}
	if !validModes[mode] {
		return fmt.Errorf("invalid mode: %s (must be enableHosting, disableHosting, setPrimary, enableDkim, or disableDkim)", mode)
	}

	path := fmt.Sprintf("/api/organization/%d/domains/%s", ac.zoid, domainName)

	req := DomainModeRequest{
		Mode: mode,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := ac.client.DoMail(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ac.parseErrorResponse(resp)
	}

	return nil
}

// GetAuditLogs fetches admin action audit logs with cursor pagination
func (ac *AdminClient) GetAuditLogs(ctx context.Context, startTime, endTime time.Time, searchKey string, limit int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	var allLogs []AuditLog
	lastEntityID := ""
	lastIndexTime := ""

	for {
		// Build query parameters
		path := fmt.Sprintf("/api/organization/%d/activity?startTime=%d&endTime=%d&limit=%d",
			ac.zoid, startTime.UnixMilli(), endTime.UnixMilli(), limit)

		if searchKey != "" {
			path += "&searchKey=" + url.QueryEscape(searchKey)
		}

		// Add cursor parameters for pagination
		if lastEntityID != "" {
			path += "&lastEntityId=" + url.QueryEscape(lastEntityID)
			path += "&lastIndexTime=" + url.QueryEscape(lastIndexTime)
		}

		resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, ac.parseErrorResponse(resp)
		}

		var auditResp AuditLogResponse
		if err := json.NewDecoder(resp.Body).Decode(&auditResp); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		if auditResp.Status.Code != 200 {
			return nil, fmt.Errorf("API error: %s (code %d)", auditResp.Status.Description, auditResp.Status.Code)
		}

		// Append logs from this page
		allLogs = append(allLogs, auditResp.Data.Audit...)

		// Check if there are more pages
		if auditResp.Data.LastEntityID == "" || len(auditResp.Data.Audit) == 0 {
			break
		}

		// Update cursor for next request
		lastEntityID = auditResp.Data.LastEntityID
		lastIndexTime = auditResp.Data.LastIndexTime
	}

	return allLogs, nil
}

// GetLoginHistory fetches login history logs with scroll-based pagination
func (ac *AdminClient) GetLoginHistory(ctx context.Context, mode string, startTime, endTime time.Time, batchSize int) ([]LoginHistoryEntry, error) {
	// Validate 90-day retention limit
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90)
	if startTime.Before(ninetyDaysAgo) {
		return nil, fmt.Errorf("login history only available for last 90 days")
	}

	if batchSize <= 0 {
		batchSize = 100
	}

	var allEntries []LoginHistoryEntry
	scrollID := ""

	for {
		// Build query parameters
		path := fmt.Sprintf("/api/organization/%d/accounts/reports/loginHistory?mode=%s&fromTime=%d&toTime=%d&batchSize=%d",
			ac.zoid, mode, startTime.UnixMilli(), endTime.UnixMilli(), batchSize)

		// Add scroll ID for pagination
		if scrollID != "" {
			path += "&scrollId=" + url.QueryEscape(scrollID)
		}

		resp, err := ac.client.DoMail(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, ac.parseErrorResponse(resp)
		}

		var loginResp LoginHistoryResponse
		if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		if loginResp.Status.Code != 200 {
			return nil, fmt.Errorf("API error: %s (code %d)", loginResp.Status.Description, loginResp.Status.Code)
		}

		// Append entries from this page
		allEntries = append(allEntries, loginResp.Data.LoginHistory...)

		// Check if there are more pages
		if loginResp.Data.ScrollID == "" || len(loginResp.Data.LoginHistory) == 0 {
			break
		}

		// Update scroll ID for next request
		scrollID = loginResp.Data.ScrollID
	}

	return allEntries, nil
}

// GetSMTPLogs fetches SMTP transaction logs with forward pagination
func (ac *AdminClient) GetSMTPLogs(ctx context.Context, startTime, endTime time.Time, searchCriteria, searchKey string, limit int) ([]SMTPLogEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	path := fmt.Sprintf("/api/organization/%d/smtplogs", ac.zoid)

	var allEntries []SMTPLogEntry
	isNext := false
	pageKey := ""

	for {
		// Build request body
		reqBody := map[string]interface{}{
			"fromDateTime":   startTime.UnixMilli(),
			"toDateTime":     endTime.UnixMilli(),
			"searchCriteria": searchCriteria,
			"searchKey":      searchKey,
			"limit":          limit,
			"isNext":         isNext,
			"isPrevious":     false,
			"pageKey":        pageKey,
			"prevKey":        "",
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}

		resp, err := ac.client.DoMail(ctx, http.MethodPost, path, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, ac.parseErrorResponse(resp)
		}

		var smtpResp SMTPLogResponse
		if err := json.NewDecoder(resp.Body).Decode(&smtpResp); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		if smtpResp.Status.Code != 200 {
			return nil, fmt.Errorf("API error: %s (code %d)", smtpResp.Status.Description, smtpResp.Status.Code)
		}

		// Append entries from this page
		allEntries = append(allEntries, smtpResp.Data.Response...)

		// Check if there are more pages
		if !smtpResp.Data.HasNext || len(smtpResp.Data.Response) == 0 {
			break
		}

		// Update pagination parameters for next request
		isNext = true
		pageKey = smtpResp.Data.PageKey
	}

	return allEntries, nil
}
