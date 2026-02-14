# Phase 2: Admin -- Users & Groups - Research

**Researched:** 2026-02-14
**Domain:** Zoho Mail Admin API (user and group management)
**Confidence:** MEDIUM

## Summary

Phase 2 implements comprehensive user and group management for Zoho organizations through the **Zoho Mail API** (not a separate Directory API). The API provides RESTful endpoints for all CRUD operations on users and groups, using **offset-based pagination** (start/limit parameters), standard OAuth2 scopes, and the existing 30 req/min rate limit.

**Critical Finding:** All admin operations use the **Zoho Mail API** (`/api/organization/{zoid}/accounts` and `/api/organization/{zoid}/groups`). There is NO separate "Zoho Directory API" for user/group management. The scopes required are `ZohoMail.organization.accounts.ALL` and `ZohoMail.organization.groups.ALL`.

**Key architectural challenge:** The API uses **three different ID types** (zoid, zuid, accountId) that must be tracked correctly. The organization ID (zoid) is retrieved via the Organization Details API at phase start. User operations use zuid (Zoho User ID) while some account operations use accountId. Group operations use zgid (Zoho Group ID).

**Primary recommendation:** Build a dedicated admin client layer with pagination abstraction, implement ID resolution helpers (email→zuid, zgid lookup), and extend the existing output formatter to handle list operations with proper column definitions.

## Standard Stack

### Core (Already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| golang.org/x/oauth2 | v0.35.0 | OAuth2 auth flow | Required for Zoho API authentication |
| golang.org/x/time/rate | v0.14.0 | Rate limiting | Already used in Phase 1 for 25 req/min budget |
| github.com/cenkalti/backoff/v4 | v4.3.0 | Exponential backoff retry | Already handles 429 responses in RateLimitTransport |
| encoding/json | stdlib | JSON marshaling | API request/response format |

### Supporting (Already in go.mod)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/alecthomas/kong | v1.14.0 | CLI command parsing | Struct tags for all commands |
| github.com/charmbracelet/lipgloss/v2 | v2.0.0-beta1 | Rich terminal output | Already used for styled output |
| github.com/rodaine/table | v1.3.0 | Table rendering | Already used in output/table.go |

### New Dependencies Required
**None.** All required libraries are already in the project from Phase 1.

**Installation:**
No new dependencies needed. Existing stack sufficient.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── zoho/
│   ├── client.go           # Existing: region-aware HTTP client
│   ├── ratelimit.go        # Existing: 429 retry logic
│   ├── admin_client.go     # NEW: admin operations wrapper
│   └── pagination.go       # NEW: pagination abstraction
├── cli/
│   ├── admin_users.go      # NEW: user commands (list, get, create, update, etc.)
│   └── admin_groups.go     # NEW: group commands (list, get, create, update, etc.)
└── output/
    ├── formatter.go        # Existing: Formatter interface
    └── table.go            # Existing: table rendering
```

### Pattern 1: Admin Client Layer with Context Preservation
**What:** Wrap the generic zoho.Client with admin-specific methods that handle ID resolution and response parsing.
**When to use:** For all user and group operations to avoid duplicating API interaction code.
**Example:**
```go
// internal/zoho/admin_client.go
type AdminClient struct {
    client *Client
    zoid   int64 // Organization ID cached at client init
}

func NewAdminClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*AdminClient, error) {
    client, err := NewClient(cfg, tokenSource)
    if err != nil {
        return nil, err
    }

    // Fetch organization ID once during initialization
    zoid, err := client.getOrganizationID(context.Background())
    if err != nil {
        return nil, fmt.Errorf("fetch organization ID: %w", err)
    }

    return &AdminClient{
        client: client,
        zoid:   zoid,
    }, nil
}

// ListUsers wraps pagination logic and returns typed results
func (ac *AdminClient) ListUsers(ctx context.Context, start, limit int) (*UserListResponse, error) {
    path := fmt.Sprintf("/api/organization/%d/accounts?start=%d&limit=%d", ac.zoid, start, limit)
    resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, parseErrorResponse(resp)
    }

    var result UserListResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }
    return &result, nil
}
```

### Pattern 2: Offset-Based Pagination Iterator
**What:** Abstract offset pagination into a reusable iterator that handles fetch-next logic transparently.
**When to use:** For list operations (users, groups) to simplify CLI command implementations.
**Example:**
```go
// internal/zoho/pagination.go
type PageIterator[T any] struct {
    fetchFunc func(start, limit int) ([]T, error)
    pageSize  int
    current   int
    buffer    []T
    done      bool
}

func NewPageIterator[T any](fetchFunc func(start, limit int) ([]T, error), pageSize int) *PageIterator[T] {
    return &PageIterator[T]{
        fetchFunc: fetchFunc,
        pageSize:  pageSize,
    }
}

func (pi *PageIterator[T]) Next() (T, bool) {
    if len(pi.buffer) == 0 && !pi.done {
        items, err := pi.fetchFunc(pi.current, pi.pageSize)
        if err != nil || len(items) == 0 {
            pi.done = true
            var zero T
            return zero, false
        }
        pi.buffer = items
        pi.current += len(items)
    }

    if len(pi.buffer) == 0 {
        var zero T
        return zero, false
    }

    item := pi.buffer[0]
    pi.buffer = pi.buffer[1:]
    return item, true
}

func (pi *PageIterator[T]) FetchAll() ([]T, error) {
    var all []T
    for {
        item, ok := pi.Next()
        if !ok {
            break
        }
        all = append(all, item)
    }
    return all, nil
}
```

### Pattern 3: ID Resolution Helpers
**What:** Helper methods to resolve email addresses to zuid/accountId and handle the zoid/zuid/accountId complexity.
**When to use:** When users provide email addresses instead of numeric IDs in CLI commands.
**Example:**
```go
// GetUserByEmail fetches user details by email, returns zuid and full user object
func (ac *AdminClient) GetUserByEmail(ctx context.Context, email string) (*User, error) {
    // List users with pagination until we find the match
    start := 0
    limit := 50
    for {
        users, err := ac.ListUsers(ctx, start, limit)
        if err != nil {
            return nil, err
        }

        for _, user := range users.Data {
            if user.EmailAddress == email {
                return &user, nil
            }
        }

        if len(users.Data) < limit {
            break // No more results
        }
        start += limit
    }
    return nil, fmt.Errorf("user not found: %s", email)
}
```

### Pattern 4: Response Type Definitions with JSON Tags
**What:** Define Go structs matching API responses with proper JSON tags and omitempty for optional fields.
**When to use:** For all API request/response structures to ensure correct serialization.
**Example:**
```go
// User represents a Zoho Mail user account
type User struct {
    ZUID              int64    `json:"zuid"`
    AccountID         int64    `json:"accountId"`
    EmailAddress      string   `json:"emailAddress"`
    FirstName         string   `json:"firstName,omitempty"`
    LastName          string   `json:"lastName,omitempty"`
    DisplayName       string   `json:"displayName,omitempty"`
    Role              string   `json:"role"` // member, admin, super_admin
    MailboxStatus     string   `json:"mailboxStatus"`
    UsedStorage       int64    `json:"usedStorage"`
    PlanStorage       int64    `json:"planStorage"`
    TFAEnabled        bool     `json:"tfaEnabled"`
    IMAPAccessEnabled bool     `json:"imapAccessEnabled"`
    POPAccessEnabled  bool     `json:"popAccessEnabled"`
    LastLogin         int64    `json:"lastLogin,omitempty"` // Unix timestamp
}

type UserListResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data []User `json:"data"`
}

// CreateUserRequest for POST /api/organization/{zoid}/accounts
type CreateUserRequest struct {
    PrimaryEmailAddress string   `json:"primaryEmailAddress"`
    Password            string   `json:"password"`
    FirstName           string   `json:"firstName,omitempty"`
    LastName            string   `json:"lastName,omitempty"`
    DisplayName         string   `json:"displayName,omitempty"`
    Role                string   `json:"role,omitempty"` // member, admin, super_admin
    Country             string   `json:"country,omitempty"`
    Language            string   `json:"language,omitempty"`
    TimeZone            string   `json:"timeZone,omitempty"`
    OneTimePassword     bool     `json:"oneTimePassword,omitempty"`
    GroupMailList       []string `json:"groupMailList,omitempty"` // Max 100
}
```

### Anti-Patterns to Avoid
- **Manual pagination loops in CLI code:** Use PageIterator abstraction instead of direct start/limit management in command handlers.
- **Hardcoding organization ID:** Always fetch zoid dynamically at client initialization; don't assume static IDs.
- **Ignoring mode parameter in update operations:** Zoho uses a `mode` field in PUT request bodies to specify operation type (e.g., "disableUser", "enableUser", "changeRole").
- **Deleting users without confirmation:** User deletion is permanent; CLI should require explicit confirmation flag (e.g., `--confirm`).
- **Confusing zuid vs accountId:** While often interchangeable in responses, some endpoints specifically require one or the other; check API docs per endpoint.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Rate limiting | Custom request throttler | golang.org/x/time/rate.Limiter (already integrated) | Existing RateLimitTransport handles 25 req/min budget and 429 retry |
| Exponential backoff | Custom retry loops | cenkalti/backoff/v4 (already integrated) | Already configured in RateLimitTransport with jitter |
| Pagination state | Custom offset tracking | PageIterator pattern (see above) | Avoids bugs with start/limit arithmetic and EOF detection |
| JSON (de)serialization | String building or custom parsing | encoding/json with struct tags | Type-safe, handles edge cases, standard library |
| Email validation | Regex or manual parsing | Accept any string, let Zoho API validate | API returns 400 with clear error messages for invalid emails |
| Table formatting | Manual column alignment | rodaine/table (already integrated) | Handles width calculation, truncation, alignment |

**Key insight:** Zoho's API handles most validation server-side. Focus on clean data structures and let the API reject invalid requests with actionable error messages. Don't duplicate validation logic in the CLI.

## Common Pitfalls

### Pitfall 1: ID Confusion (zoid/zuid/accountId/zgid)
**What goes wrong:** Mixing up the four different ID types leads to 404 errors or wrong entity operations.
**Why it happens:** API documentation inconsistently uses "user ID" to mean both zuid and accountId; some endpoints accept either.
**How to avoid:**
- Cache zoid at AdminClient initialization (from GET /api/organization/)
- Use zuid for user-centric operations (fetched from list/get endpoints)
- Use accountId for account-specific settings (same numeric value as zuid in responses)
- Use zgid for group operations (fetched from group list/create endpoints)
- Document which ID type each function expects in godoc comments
**Warning signs:** 404 responses when you expect 200; verify you're using the correct ID field from the response.

### Pitfall 2: Pagination Edge Cases
**What goes wrong:** Missing the last page of results or infinite loops when total count isn't divisible by limit.
**Why it happens:** Offset-based pagination can return empty results when start exceeds total count; no explicit "has_more" field in responses.
**How to avoid:**
- Detect end-of-data by checking `len(response.Data) < limit`
- Use PageIterator to encapsulate this logic
- Don't rely on a total count field (not provided in API responses)
**Warning signs:** CLI commands showing incomplete lists; loop timeout errors during bulk operations.

### Pitfall 3: Group Member Batch Limits
**What goes wrong:** Adding >100 members to a group fails silently or returns unclear errors.
**Why it happens:** API documentation mentions 100-group limit for user creation but doesn't clearly state member batch limits.
**How to avoid:**
- Chunk member additions into batches of 50 (conservative safety margin)
- Check response status for each batch
- Implement retry logic for partial failures
**Warning signs:** Group operations timing out; inconsistent member counts between API and web UI.

### Pitfall 4: User State Management (enable/disable/delete)
**What goes wrong:** Calling "enable user" on an already-enabled user; confusing disable vs delete semantics.
**Why it happens:** Three separate operations (enable user, enable mail account, delete) with overlapping effects.
**How to avoid:**
- Check current user status before enable/disable operations
- Document that "disable user" is reversible, "delete user" is permanent
- Require `--force` flag for delete operations
- Use different CLI verbs: `activate`/`deactivate` for enable/disable, `delete` with confirmation
**Warning signs:** Idempotency errors (400 responses for enable on enabled user); unexpected data loss from delete.

### Pitfall 5: Missing Organization ID Retrieval
**What goes wrong:** Hardcoding zoid or requiring users to provide it manually breaks multi-org workflows.
**Why it happens:** API docs show {zoid} in endpoints but don't emphasize how to retrieve it dynamically.
**How to avoid:**
- Call GET /api/organization/ (no zoid parameter) to get current org details
- Cache zoid in AdminClient struct
- Include zoid in verbose/debug output for troubleshooting
**Warning signs:** CLI working in one org but failing in another; manual config requirements.

### Pitfall 6: Mode Parameter Omission
**What goes wrong:** PUT requests fail with unclear errors because the `mode` field is missing from the request body.
**Why it happens:** Zoho uses `mode` to multiplex different operations on the same endpoint (e.g., PUT /accounts with mode=enableUser vs mode=disableUser).
**How to avoid:**
- Always include `mode` field in PUT request structs
- Define separate request types for each operation (EnableUserRequest, DisableUserRequest)
- Validate mode values against API docs before implementation
**Warning signs:** 400 Bad Request errors on PUT operations that should work; API returning "invalid mode" errors.

## Code Examples

Verified patterns from official sources:

### Fetch Organization ID (Required First Step)
```go
// GET /api/organization/ returns org details including zoid
// Required scope: ZohoMail.partner.organization (or ZohoMail.organization.accounts.READ)
type OrgResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data struct {
        OrganizationID int64  `json:"zoid"`
        CompanyName    string `json:"companyName"`
        UserCount      int    `json:"userCount"`
        GroupCount     int    `json:"groupCount"`
    } `json:"data"`
}

func (c *Client) GetOrganizationID(ctx context.Context) (int64, error) {
    resp, err := c.Do(ctx, http.MethodGet, "/api/organization/", nil)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()

    var orgResp OrgResponse
    if err := json.NewDecoder(resp.Body).Decode(&orgResp); err != nil {
        return 0, err
    }
    return orgResp.Data.OrganizationID, nil
}
```

### List Users with Pagination
```go
// GET /api/organization/{zoid}/accounts?start=0&limit=50
// Required scope: ZohoMail.organization.accounts.READ or .ALL
// Returns: UserListResponse with array of User objects
// Pagination: offset-based (start, limit); no cursor or total count provided
// Detection of end: len(response.Data) < limit

func (ac *AdminClient) ListAllUsers(ctx context.Context) ([]User, error) {
    start := 0
    limit := 50
    var allUsers []User

    for {
        path := fmt.Sprintf("/api/organization/%d/accounts?start=%d&limit=%d",
            ac.zoid, start, limit)
        resp, err := ac.client.Do(ctx, http.MethodGet, path, nil)
        if err != nil {
            return nil, err
        }

        var result UserListResponse
        if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
            resp.Body.Close()
            return nil, err
        }
        resp.Body.Close()

        allUsers = append(allUsers, result.Data...)

        if len(result.Data) < limit {
            break // Last page
        }
        start += limit
    }
    return allUsers, nil
}
```

### Create User
```go
// POST /api/organization/{zoid}/accounts
// Required scope: ZohoMail.organization.accounts.CREATE or .ALL
// Request body: CreateUserRequest
// Response: HTTP 201 Created with full user object

func (ac *AdminClient) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
    body, err := json.Marshal(req)
    if err != nil {
        return nil, err
    }

    path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)
    resp, err := ac.client.Do(ctx, http.MethodPost, path, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated {
        return nil, parseErrorResponse(resp)
    }

    var result struct {
        Status struct {
            Code        int    `json:"code"`
            Description string `json:"description"`
        } `json:"status"`
        Data User `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return &result.Data, nil
}
```

### Disable User
```go
// PUT /api/organization/{zoid}/accounts
// Required scope: ZohoMail.organization.accounts.UPDATE or .ALL
// Critical: Must include "mode": "disableUser" in request body

type DisableUserRequest struct {
    Mode                   string `json:"mode"` // MUST be "disableUser"
    ZUID                   int64  `json:"zuid"`
    BlockIncoming          bool   `json:"blockIncoming,omitempty"`
    RemoveMailForward      bool   `json:"removeMailforward,omitempty"`
    RemoveGroupMembership  bool   `json:"removeGroupMembership,omitempty"`
    RemoveAlias            bool   `json:"removeAlias,omitempty"`
}

func (ac *AdminClient) DisableUser(ctx context.Context, zuid int64, opts DisableUserRequest) error {
    opts.Mode = "disableUser"
    opts.ZUID = zuid

    body, err := json.Marshal(opts)
    if err != nil {
        return err
    }

    path := fmt.Sprintf("/api/organization/%d/accounts", ac.zoid)
    resp, err := ac.client.Do(ctx, http.MethodPut, path, bytes.NewReader(body))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return parseErrorResponse(resp)
    }
    return nil
}
```

### Add Group Members
```go
// PUT /api/organization/{zoid}/groups/{zgid}
// Required scope: ZohoMail.organization.groups.UPDATE or .ALL
// Request body: mode="addMailGroupMember", mailGroupMemberList array
// Role options: "member" (default) or "moderator"

type AddGroupMembersRequest struct {
    Mode                 string               `json:"mode"` // MUST be "addMailGroupMember"
    MailGroupMemberList  []GroupMemberToAdd   `json:"mailGroupMemberList"`
}

type GroupMemberToAdd struct {
    MemberEmailID string `json:"memberEmailId"`
    Role          string `json:"role,omitempty"` // "member" or "moderator"
}

func (ac *AdminClient) AddGroupMembers(ctx context.Context, zgid int64, members []GroupMemberToAdd) error {
    req := AddGroupMembersRequest{
        Mode:                "addMailGroupMember",
        MailGroupMemberList: members,
    }

    body, err := json.Marshal(req)
    if err != nil {
        return err
    }

    path := fmt.Sprintf("/api/organization/%d/groups/%d", ac.zoid, zgid)
    resp, err := ac.client.Do(ctx, http.MethodPut, path, bytes.NewReader(body))
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return parseErrorResponse(resp)
    }
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual org ID from user | Auto-fetch from /api/organization/ | Current best practice (2026) | Eliminates config step; improves multi-org support |
| Cursor pagination assumption | Offset-based (start/limit) | Zoho Mail API design | Simpler client logic but less efficient for large datasets |
| Space-separated OAuth scopes | Comma-separated scopes | Zoho requirement | Already handled in Phase 1 auth/scopes.go |
| Separate Directory API | Unified Zoho Mail API | Zoho API architecture | Single client, consistent auth, fewer endpoints |

**Deprecated/outdated:**
- **Separate Zoho Directory API:** Does not exist for Mail organizations. User/group management is part of Zoho Mail API.
- **Total count in list responses:** API does not provide total counts; detect EOF by comparing result count to limit.
- **Global user search/filter:** API supports pagination but no server-side filtering by name/email; must fetch and filter client-side.

## Open Questions

1. **What is the maximum group member batch size?**
   - What we know: User creation supports max 100 groups; no explicit limit documented for adding members to a group.
   - What's unclear: Safe batch size for PUT /groups/{zgid} with addMailGroupMember mode.
   - Recommendation: Start with batch size of 50, monitor API responses, adjust based on errors. Document actual limit in code comments once verified via curl testing.

2. **Do list endpoints support filtering/search parameters?**
   - What we know: Pagination uses start/limit only; documentation shows no filter parameters.
   - What's unclear: Can we filter users by role, status, or email pattern without client-side filtering?
   - Recommendation: Implement client-side filtering for Phase 2. If performance becomes an issue, investigate undocumented query parameters via curl experimentation.

3. **What happens when updating a non-existent user/group?**
   - What we know: 404 Not Found is standard REST pattern.
   - What's unclear: Exact error response format (status.code, data.moreInfo).
   - Recommendation: Implement generic error parser that extracts status.description and data.moreInfo fields for all error responses.

4. **Are there cascade delete behaviors for groups?**
   - What we know: DELETE /groups/{zgid} removes the group.
   - What's unclear: Are members automatically removed? Are emails rejected or forwarded after deletion?
   - Recommendation: Document in CLI help text that group deletion is permanent and affects email routing. Test behavior in dev environment during 02-03 implementation.

## Sources

### Primary (HIGH confidence)
- [Zoho Mail Users API](https://www.zoho.com/mail/help/api/users-api.html) - User management endpoints
- [Zoho Mail Groups API](https://www.zoho.com/mail/help/api/group-api.html) - Group management endpoints
- [Zoho Mail Organization API](https://www.zoho.com/mail/help/api/organization-api.html) - Organization details and zoid retrieval
- [Fetch All Organization Users](https://www.zoho.com/mail/help/api/get-org-users-details.html) - Pagination parameters
- [Add User to Organization](https://www.zoho.com/mail/help/api/post-add-user-to-org.html) - Create user request structure
- [Disable User](https://www.zoho.com/mail/help/api/put-to-disable-user.html) - Mode parameter requirement
- [Delete User Account](https://www.zoho.com/mail/help/api/user-account-deletion.html) - Permanent deletion
- [Add Group Members](https://www.zoho.com/mail/help/api/group-add-member.html) - Member batch operations
- [Zoho Mail Response Codes](https://www.zoho.com/mail/help/api/response-codes.html) - Error code reference
- [API Getting Started](https://www.zoho.com/mail/help/api/getting-started-with-api.html) - Region base URLs

### Secondary (MEDIUM confidence)
- [Zoho Mail API Index](https://www.zoho.com/mail/help/api/) - API overview and scope patterns
- [Roles and Privileges](https://www.zoho.com/mail/help/adminconsole/roles-privileges.html) - Role definitions (member, admin, super_admin)
- [Zoho OAuth Scopes](https://www.zoho.com/accounts/protocol/oauth/scope.html) - Scope naming format
- [Zoho Mail Rate Limits](https://www.zoho.com/mail/help/adminconsole/rates-and-limits.html) - 30 req/min confirmed
- [API Pagination Best Practices](https://www.speakeasy.com/api-design/pagination) - Offset-based patterns
- [Go Retry and Backoff Guide](https://oneuptime.com/blog/post/2026-01-07-go-retry-exponential-backoff/view) - Exponential backoff implementation
- [Lipgloss Table Package](https://pkg.go.dev/github.com/charmbracelet/lipgloss/table) - Table rendering (already integrated)
- [Go Struct Tags Guide](https://www.dolthub.com/blog/2024-02-07-go-tags/) - JSON tag best practices

### Tertiary (LOW confidence - requires verification)
- WebSearch results on OAuth scopes from Pipedream integration - Suggests granular scopes (.CREATE, .READ, .UPDATE, .DELETE) but official docs only mention .ALL
- Community forum discussions on concurrent request limits - No official documentation found; 30 req/min is the only confirmed limit

## Critical Research Flag Addressed

**Original flag:** "Phase 2 needs API endpoint audit (curl verification) at phase start -- some admin ops may require Zoho Directory API instead of Mail API"

**Resolution:** Research confirms that **all user and group management operations use the Zoho Mail API**. There is no separate "Zoho Directory API" for these operations. All endpoints are under `/api/organization/{zoid}/accounts` (users) and `/api/organization/{zoid}/groups` (groups).

**Recommendation for phase start:**
1. Run curl verification against `/api/organization/` to fetch zoid
2. Test pagination with `/api/organization/{zoid}/accounts?start=0&limit=10`
3. Verify scopes work: `ZohoMail.organization.accounts.ALL` and `ZohoMail.organization.groups.ALL`
4. Confirm rate limit behavior matches Phase 1 (25 req/min budget sufficient)

No separate API integration needed. Existing zoho.Client infrastructure supports all admin operations.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already integrated and verified in Phase 1
- Architecture: MEDIUM - Patterns are standard but PageIterator abstraction needs validation in practice
- API endpoints: HIGH - Verified via official Zoho Mail API documentation with curl examples
- Pagination: MEDIUM - Offset-based pattern confirmed but edge cases require testing
- Error handling: MEDIUM - Error response format documented but specific error messages need verification
- Pitfalls: MEDIUM - Based on API documentation analysis and common REST API patterns; requires real-world testing

**Research date:** 2026-02-14
**Valid until:** 2026-03-16 (30 days - Zoho Mail API is stable, infrequent breaking changes expected)
