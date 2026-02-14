# Phase 4: Mail -- Read Operations - Research

**Researched:** 2026-02-14
**Domain:** Zoho Mail API (message retrieval, search, folders, labels, attachments)
**Confidence:** HIGH

## Summary

Phase 4 implements comprehensive mail reading operations through the **Zoho Mail API**. The API provides REST endpoints for message retrieval (list, get, search), folder/label management, thread viewing, and attachment downloads. Unlike the Admin API from Phases 2-3, the Mail API operates on a **per-account basis**, requiring an `accountId` parameter for all operations instead of organization-level `zoid`.

**Critical findings:**
- **Account-scoped API**: All Mail API endpoints require `accountId` (not `zoid`), obtained via `GET /api/accounts`. This is a fundamental architectural difference from Admin APIs.
- **Three-tier message retrieval**: List (summary), Metadata (full headers), Content (body HTML). Each requires separate API calls.
- **Offset pagination**: Messages use standard `start`/`limit` pagination (1-200 per request), not cursor-based like audit logs.
- **Search syntax**: Search uses Zoho's proprietary search key syntax (e.g., "newMails", "from:user@example.com") rather than structured query parameters.
- **Binary attachment downloads**: Attachments return raw `application/octet-stream` (not JSON), requiring special handling.
- **Thread ID in message list**: Thread viewing requires extracting `threadId` from message list responses; no dedicated thread retrieval API documented.

**Primary recommendation:** Create a new `MailClient` (separate from `AdminClient`) that caches `accountId`, implements offset-based pagination for message lists, handles three-tier message retrieval (list → metadata → content), supports Zoho search syntax, and streams binary attachment downloads to disk.

## Standard Stack

### Core (Already in go.mod)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| encoding/json | stdlib | JSON marshaling | API request/response format |
| net/http | stdlib | HTTP client | GET/POST requests for Mail API |
| io | stdlib | Stream handling | Attachment binary downloads |
| os | stdlib | File operations | Writing attachments to disk |
| golang.org/x/oauth2 | v0.35.0 | OAuth2 auth flow | Required for Zoho API authentication |
| golang.org/x/time/rate | v0.14.0 | Rate limiting | Already used for 30 req/min budget |

### Supporting (Already in go.mod)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/alecthomas/kong | v1.14.0 | CLI command parsing | Struct tags for all mail commands |
| github.com/charmbracelet/lipgloss/v2 | v2.0.0-beta1 | Rich terminal output | Styled output formatting |
| github.com/rodaine/table | v1.3.0 | Table rendering | List output for messages and folders |
| github.com/cenkalti/backoff/v4 | v4.3.0 | Exponential backoff retry | Already handles 429 responses |

### New Dependencies Required
**None.** All required libraries are already in the project from Phases 1-3. The standard library's `io` package handles binary attachment streaming natively.

**Installation:**
No new dependencies needed. Existing stack sufficient.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── zoho/
│   ├── client.go           # Existing: region-aware HTTP client
│   ├── admin_client.go     # Existing: Admin API (zoid-based)
│   ├── mail_client.go      # NEW: Mail API (accountId-based)
│   ├── pagination.go       # Existing: PageIterator for offset pagination
│   ├── types.go            # EXTEND: add Mail API types
│   └── search.go           # NEW: Zoho search syntax builder
├── cli/
│   ├── mail_folders.go     # NEW: folder/label list commands
│   ├── mail_messages.go    # NEW: message list/get/search commands
│   ├── mail_threads.go     # NEW: thread view command
│   └── mail_attachments.go # NEW: attachment download command
└── output/
    ├── formatter.go        # Existing: Formatter interface
    └── table.go            # Existing: table rendering
```

### Pattern 1: MailClient with Cached accountId

**What:** New client struct for Mail API operations that caches `accountId` similar to how AdminClient caches `zoid`.

**When to use:** For all mail read operations to maintain consistency with existing client patterns.

**Example:**
```go
// internal/zoho/mail_client.go (new file)
package zoho

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"

    "golang.org/x/oauth2"
    "github.com/semmy-space/zoh/internal/config"
)

// MailClient wraps the Zoho Client with mail-specific functionality
type MailClient struct {
    client    *Client
    accountID string // Cached account ID
}

// NewMailClient creates a new MailClient with the given config and token source
// It automatically resolves and caches the primary account ID
func NewMailClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*MailClient, error) {
    client, err := NewClient(cfg, tokenSource)
    if err != nil {
        return nil, fmt.Errorf("create client: %w", err)
    }

    mc := &MailClient{
        client: client,
    }

    // Resolve primary account ID
    ctx := context.Background()
    accountID, err := mc.getPrimaryAccountID(ctx)
    if err != nil {
        return nil, fmt.Errorf("get account ID: %w", err)
    }
    mc.accountID = accountID

    return mc, nil
}

// getPrimaryAccountID fetches the first account ID from the accounts list
func (mc *MailClient) getPrimaryAccountID(ctx context.Context) (string, error) {
    resp, err := mc.client.Do(ctx, http.MethodGet, "/api/accounts", nil)
    if err != nil {
        return "", fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", mc.parseErrorResponse(resp)
    }

    var accountsResp AccountListResponse
    if err := json.NewDecoder(resp.Body).Decode(&accountsResp); err != nil {
        return "", fmt.Errorf("decode response: %w", err)
    }

    if accountsResp.Status.Code != 200 {
        return "", fmt.Errorf("API error: %s (code %d)",
            accountsResp.Status.Description, accountsResp.Status.Code)
    }

    if len(accountsResp.Data) == 0 {
        return "", fmt.Errorf("no mail accounts found")
    }

    // Return the first account (primary account)
    return accountsResp.Data[0].AccountID, nil
}

// parseErrorResponse reuses the same error parsing logic as AdminClient
func (mc *MailClient) parseErrorResponse(resp *http.Response) error {
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
    }

    var apiErr APIError
    if err := json.Unmarshal(body, &apiErr); err != nil {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
    }

    return fmt.Errorf("HTTP %d: %w", resp.StatusCode, &apiErr)
}
```

### Pattern 2: Three-Tier Message Retrieval

**What:** Separate methods for list (summary), metadata (full headers), and content (body) retrieval to optimize API calls.

**When to use:**
- **List**: For displaying message summaries in tables
- **Metadata**: When full headers needed (for display or reply-to logic)
- **Content**: Only when user requests to read the full message body

**Example:**
```go
// ListMessages retrieves message summaries from a folder with pagination
func (mc *MailClient) ListMessages(ctx context.Context, folderID string, start, limit int) ([]MessageSummary, error) {
    path := fmt.Sprintf("/api/accounts/%s/messages/view?folderId=%s&start=%d&limit=%d",
        mc.accountID, folderID, start, limit)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var msgResp MessageListResponse
    if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if msgResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            msgResp.Status.Description, msgResp.Status.Code)
    }

    return msgResp.Data, nil
}

// GetMessageMetadata retrieves full metadata for a specific message
func (mc *MailClient) GetMessageMetadata(ctx context.Context, folderID, messageID string) (*MessageMetadata, error) {
    path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/details",
        mc.accountID, folderID, messageID)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var metaResp MessageMetadataResponse
    if err := json.NewDecoder(resp.Body).Decode(&metaResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if metaResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            metaResp.Status.Description, metaResp.Status.Code)
    }

    return &metaResp.Data, nil
}

// GetMessageContent retrieves HTML body content for a specific message
func (mc *MailClient) GetMessageContent(ctx context.Context, folderID, messageID string) (*MessageContent, error) {
    path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/content",
        mc.accountID, folderID, messageID)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var contentResp MessageContentResponse
    if err := json.NewDecoder(resp.Body).Decode(&contentResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if contentResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            contentResp.Status.Description, contentResp.Status.Code)
    }

    return &contentResp.Data, nil
}
```

### Pattern 3: Search with Query Builder

**What:** Helper functions to build Zoho search syntax strings from common search criteria.

**When to use:** For implementing user-friendly search flags in CLI commands.

**Example:**
```go
// internal/zoho/search.go (new file)
package zoho

import (
    "fmt"
    "net/url"
    "strings"
    "time"
)

// SearchQuery builds Zoho Mail search syntax strings
type SearchQuery struct {
    parts []string
}

// NewSearchQuery creates a new search query builder
func NewSearchQuery() *SearchQuery {
    return &SearchQuery{parts: []string{}}
}

// From adds sender filter
func (sq *SearchQuery) From(email string) *SearchQuery {
    sq.parts = append(sq.parts, fmt.Sprintf("from:%s", email))
    return sq
}

// Subject adds subject filter
func (sq *SearchQuery) Subject(text string) *SearchQuery {
    sq.parts = append(sq.parts, fmt.Sprintf("subject:%s", text))
    return sq
}

// DateAfter adds date range filter (after date)
func (sq *SearchQuery) DateAfter(date time.Time) *SearchQuery {
    sq.parts = append(sq.parts, fmt.Sprintf("after:%s", date.Format("2006/01/02")))
    return sq
}

// DateBefore adds date range filter (before date)
func (sq *SearchQuery) DateBefore(date time.Time) *SearchQuery {
    sq.parts = append(sq.parts, fmt.Sprintf("before:%s", date.Format("2006/01/02")))
    return sq
}

// HasAttachment filters for messages with attachments
func (sq *SearchQuery) HasAttachment() *SearchQuery {
    sq.parts = append(sq.parts, "has:attachment")
    return sq
}

// IsUnread filters for unread messages
func (sq *SearchQuery) IsUnread() *SearchQuery {
    sq.parts = append(sq.parts, "is:unread")
    return sq
}

// Text adds general text search
func (sq *SearchQuery) Text(query string) *SearchQuery {
    sq.parts = append(sq.parts, query)
    return sq
}

// Build returns the search syntax string
func (sq *SearchQuery) Build() string {
    return strings.Join(sq.parts, " ")
}

// SearchMessages searches messages using Zoho search syntax
func (mc *MailClient) SearchMessages(ctx context.Context, searchKey string, start, limit int) ([]MessageSummary, error) {
    path := fmt.Sprintf("/api/accounts/%s/messages/search?searchKey=%s&start=%d&limit=%d",
        mc.accountID, url.QueryEscape(searchKey), start, limit)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var msgResp MessageListResponse
    if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if msgResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            msgResp.Status.Description, msgResp.Status.Code)
    }

    return msgResp.Data, nil
}
```

### Pattern 4: Binary Attachment Streaming

**What:** Stream attachment downloads directly to disk without loading entire file into memory.

**When to use:** For all attachment downloads to handle large files efficiently.

**Example:**
```go
// DownloadAttachment downloads an attachment to the specified file path
func (mc *MailClient) DownloadAttachment(ctx context.Context, folderID, messageID, attachmentID, destPath string) error {
    path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/attachments/%s",
        mc.accountID, folderID, messageID, attachmentID)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return mc.parseErrorResponse(resp)
    }

    // Create destination file
    outFile, err := os.Create(destPath)
    if err != nil {
        return fmt.Errorf("create file: %w", err)
    }
    defer outFile.Close()

    // Stream response body to file
    if _, err := io.Copy(outFile, resp.Body); err != nil {
        return fmt.Errorf("write file: %w", err)
    }

    return nil
}

// ListAttachments retrieves attachment metadata for a message
func (mc *MailClient) ListAttachments(ctx context.Context, folderID, messageID string) ([]Attachment, error) {
    path := fmt.Sprintf("/api/accounts/%s/folders/%s/messages/%s/attachments",
        mc.accountID, folderID, messageID)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var attachResp AttachmentListResponse
    if err := json.NewDecoder(resp.Body).Decode(&attachResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if attachResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            attachResp.Status.Description, attachResp.Status.Code)
    }

    return attachResp.Data, nil
}
```

### Pattern 5: Folder and Label Listing

**What:** Methods to retrieve folder and label lists for navigation and filtering.

**When to use:** For displaying available folders/labels and resolving folder/label IDs from names.

**Example:**
```go
// ListFolders retrieves all folders for the account
func (mc *MailClient) ListFolders(ctx context.Context) ([]Folder, error) {
    path := fmt.Sprintf("/api/accounts/%s/folders", mc.accountID)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var folderResp FolderListResponse
    if err := json.NewDecoder(resp.Body).Decode(&folderResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if folderResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            folderResp.Status.Description, folderResp.Status.Code)
    }

    return folderResp.Data, nil
}

// GetFolderByName retrieves folder ID by folder name (e.g., "Inbox")
func (mc *MailClient) GetFolderByName(ctx context.Context, name string) (*Folder, error) {
    folders, err := mc.ListFolders(ctx)
    if err != nil {
        return nil, err
    }

    for _, folder := range folders {
        if folder.FolderName == name {
            return &folder, nil
        }
    }

    return nil, fmt.Errorf("folder not found: %s", name)
}

// ListLabels retrieves all labels for the account
func (mc *MailClient) ListLabels(ctx context.Context) ([]Label, error) {
    path := fmt.Sprintf("/api/accounts/%s/labels", mc.accountID)

    resp, err := mc.client.Do(ctx, http.MethodGet, path, nil)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, mc.parseErrorResponse(resp)
    }

    var labelResp LabelListResponse
    if err := json.NewDecoder(resp.Body).Decode(&labelResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    if labelResp.Status.Code != 200 {
        return nil, fmt.Errorf("API error: %s (code %d)",
            labelResp.Status.Description, labelResp.Status.Code)
    }

    return labelResp.Data, nil
}
```

### Pattern 6: Thread Viewing via ThreadID

**What:** Extract thread messages by filtering message list for a specific `threadId`.

**When to use:** When user requests to view a conversation/thread.

**Example:**
```go
// GetThread retrieves all messages in a thread by threadId
func (mc *MailClient) GetThread(ctx context.Context, folderID, threadID string) ([]MessageSummary, error) {
    // List all messages in folder
    // Note: Zoho doesn't have a dedicated thread endpoint, so we need to:
    // 1. List messages
    // 2. Filter by threadId client-side
    // This is inefficient but reflects current API limitations

    var allMessages []MessageSummary
    start := 1
    limit := 200 // Max per request

    for {
        messages, err := mc.ListMessages(ctx, folderID, start, limit)
        if err != nil {
            return nil, err
        }

        allMessages = append(allMessages, messages...)

        if len(messages) < limit {
            break
        }
        start += limit
    }

    // Filter for matching threadId
    var threadMessages []MessageSummary
    for _, msg := range allMessages {
        if msg.ThreadID == threadID {
            threadMessages = append(threadMessages, msg)
        }
    }

    if len(threadMessages) == 0 {
        return nil, fmt.Errorf("no messages found in thread: %s", threadID)
    }

    return threadMessages, nil
}
```

### Pattern 7: Response Type Definitions for Mail API

**What:** Define Go structs matching Mail API responses with proper JSON tags.

**When to use:** For all Mail API request/response structures.

**Example:**
```go
// internal/zoho/types.go (extend existing file)

// Account represents a mail account
type Account struct {
    AccountID          string `json:"accountId"`
    EmailAddress       string `json:"emailAddress"`
    AccountDisplayName string `json:"accountDisplayName"`
    Type               string `json:"type"` // ZOHO_ACCOUNT, IMAP_ACCOUNT
    Status             string `json:"status"`
}

// AccountListResponse from GET /api/accounts
type AccountListResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data []Account `json:"data"`
}

// Folder represents a mail folder
type Folder struct {
    FolderID   string `json:"folderId"`
    FolderName string `json:"folderName"`
    FolderType string `json:"folderType"` // Inbox, Drafts, Sent, etc.
    Path       string `json:"path"`
    IMAPAccess bool   `json:"imapAccess"`
    IsArchived int    `json:"isArchived"` // 0 or 1
}

// FolderListResponse from GET /api/accounts/{accountId}/folders
type FolderListResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data []Folder `json:"data"`
}

// Label represents a mail label/tag
type Label struct {
    LabelID    string `json:"labelId"`
    LabelName  string `json:"labelName"`
    LabelColor string `json:"labelColor"`
}

// LabelListResponse from GET /api/accounts/{accountId}/labels
type LabelListResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data []Label `json:"data"`
}

// MessageSummary represents a message in list view (minimal fields)
type MessageSummary struct {
    MessageID     string `json:"messageId"`
    ThreadID      string `json:"threadId"`
    Subject       string `json:"subject"`
    FromAddress   string `json:"fromAddress"`
    Sender        string `json:"sender"`
    ReceivedTime  int64  `json:"receivedTime"` // Unix milliseconds
    Status        string `json:"status"`       // READ, UNREAD
    HasAttachment bool   `json:"hasAttachment"`
    FlagID        int    `json:"flagid"`
    Priority      int    `json:"priority"`
}

// MessageListResponse from GET /api/accounts/{accountId}/messages/view
type MessageListResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data []MessageSummary `json:"data"`
}

// MessageMetadata represents full message metadata (headers, recipients, etc.)
type MessageMetadata struct {
    MessageID      string   `json:"messageId"`
    ThreadID       string   `json:"threadId"`
    FolderID       string   `json:"folderId"`
    Subject        string   `json:"subject"`
    FromAddress    string   `json:"fromAddress"`
    Sender         string   `json:"sender"`
    ToAddress      []string `json:"toAddress"`
    CcAddress      []string `json:"ccAddress"`
    SentDateInGMT  int64    `json:"sentDateInGMT"` // Unix milliseconds
    ReceivedTime   int64    `json:"receivedTime"`
    MessageSize    int64    `json:"messageSize"`
    HasAttachment  bool     `json:"hasAttachment"`
    HasInline      bool     `json:"hasInline"`
    Status         string   `json:"status"`
    Priority       int      `json:"priority"`
    FlagID         int      `json:"flagid"`
}

// MessageMetadataResponse from GET /api/accounts/{accountId}/folders/{folderId}/messages/{messageId}/details
type MessageMetadataResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data MessageMetadata `json:"data"`
}

// MessageContent represents message body content
type MessageContent struct {
    MessageID string `json:"messageId"`
    Content   string `json:"content"` // HTML-formatted body
}

// MessageContentResponse from GET /api/accounts/{accountId}/folders/{folderId}/messages/{messageId}/content
type MessageContentResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data MessageContent `json:"data"`
}

// Attachment represents an email attachment
type Attachment struct {
    AttachmentID   string `json:"attachmentId"`
    AttachmentName string `json:"attachmentName"`
    AttachmentSize int64  `json:"attachmentSize"`
    AttachmentType string `json:"attachmentType"` // MIME type
}

// AttachmentListResponse from GET /api/accounts/{accountId}/folders/{folderId}/messages/{messageId}/attachments
type AttachmentListResponse struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data []Attachment `json:"data"`
}
```

### Anti-Patterns to Avoid

- **Mixing Admin and Mail APIs**: Don't use `zoid` with Mail API endpoints or `accountId` with Admin API endpoints. They are separate API domains.
- **Loading full message content for lists**: Use three-tier retrieval (list → metadata → content) to minimize API calls and bandwidth.
- **Client-side search when API supports it**: Use the search endpoint with Zoho syntax instead of fetching all messages and filtering locally.
- **Loading attachments into memory**: Stream binary downloads directly to disk using `io.Copy`.
- **Hardcoding folder names in folder IDs**: Always resolve folder names to IDs via `ListFolders` or `GetFolderByName`.
- **Assuming thread endpoint exists**: No dedicated thread retrieval API is documented; must filter messages by `threadId` client-side.
- **Ignoring pagination limits**: Message list endpoint caps at 200 per request; don't assume you can fetch all messages in one call.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Search query syntax | Manual string concatenation | SearchQuery builder pattern | Handles URL encoding, syntax validation, readable API |
| Attachment streaming | Read entire file into []byte | io.Copy to os.File | Handles large files without memory issues |
| Folder ID resolution | Hardcoded folder IDs | GetFolderByName helper | Folder IDs may vary by account, names are stable |
| Offset pagination | Manual start/limit tracking | Existing PageIterator[T] | Already handles page boundaries, reusable |
| Account ID caching | Fetch on every call | MailClient with cached accountID | Reduces API calls, consistent with AdminClient pattern |
| Binary response parsing | Custom stream handling | Standard library io.Reader | Stdlib handles EOF, partial reads, errors |

**Key insight:** Mail API is **account-scoped** (not organization-scoped), requires different client architecture than Admin API. Three-tier message retrieval (list/metadata/content) optimizes bandwidth. Search uses proprietary syntax, not structured query params. Attachments are binary streams, not JSON.

## Common Pitfalls

### Pitfall 1: Confusing accountId with zoid

**What goes wrong:** Using `zoid` (organization ID) with Mail API endpoints returns 404 or 400 errors.

**Why it happens:** Admin API uses `/api/organization/{zoid}/...` but Mail API uses `/api/accounts/{accountId}/...`. These are fundamentally different API domains.

**How to avoid:**
- Create separate `MailClient` and `AdminClient` structs
- Cache `accountId` in MailClient, `zoid` in AdminClient
- Document which client each CLI command uses
- Validate client type in command Run methods

**Warning signs:** 404 errors on valid-looking endpoints, "organization not found" errors when using accountId

### Pitfall 2: Fetching Full Message Content in Lists

**What goes wrong:** Extremely slow message list operations, high bandwidth usage, hitting rate limits.

**Why it happens:** Fetching content for every message in a list requires N additional API calls (one per message).

**How to avoid:**
- List endpoint returns summaries only (subject, sender, date)
- Metadata endpoint for full headers (when displaying single message)
- Content endpoint only when user explicitly reads a message
- Implement lazy loading: list → user selects → fetch content

**Warning signs:** Slow CLI response times, rate limit 429 errors, excessive API call counts

### Pitfall 3: Search Syntax Errors

**What goes wrong:** Search returns no results or unexpected results despite matching messages existing.

**Why it happens:** Zoho search uses proprietary syntax (e.g., "from:user@example.com" not "sender=user@example.com").

**How to avoid:**
- Use SearchQuery builder to construct valid syntax
- Test search syntax in Zoho web UI first
- Document supported operators (from:, subject:, after:, before:, has:, is:)
- URL-encode search keys before sending

**Warning signs:** Empty search results when data exists, "invalid search key" errors

### Pitfall 4: Hardcoding Folder IDs

**What goes wrong:** Folder IDs vary by account; hardcoded IDs work for developer but fail for other users.

**Why it happens:** Assuming "Inbox" always has same folder ID across all accounts.

**How to avoid:**
- Always fetch folder list first: `GET /api/accounts/{accountId}/folders`
- Resolve folder names to IDs at runtime using `GetFolderByName`
- Cache folder list in client for session duration
- Accept folder names (not IDs) in CLI flags

**Warning signs:** "Folder not found" errors for users, commands work for one account but not another

### Pitfall 5: Thread Retrieval Assumptions

**What goes wrong:** Expecting a dedicated thread endpoint like `GET /threads/{threadId}` returns 404.

**Why it happens:** Zoho's documented Thread API only has update operations (PUT), not retrieval (GET).

**How to avoid:**
- Extract `threadId` from message list responses
- Filter messages client-side by matching `threadId`
- Document that thread view requires fetching folder messages first
- Consider caching message list to avoid repeated API calls

**Warning signs:** 404 on thread endpoints, documentation mentions threads but no GET operations

### Pitfall 6: Attachment Memory Exhaustion

**What goes wrong:** CLI crashes or becomes unresponsive when downloading large attachments.

**Why it happens:** Loading entire attachment into memory before writing to disk.

**How to avoid:**
- Use `io.Copy(file, resp.Body)` for streaming
- Don't read response body into []byte
- Set appropriate buffer sizes for large files
- Show progress bar for large downloads

**Warning signs:** High memory usage, OOM errors, slow download speeds

### Pitfall 7: Missing OAuth Scopes for Mail Read

**What goes wrong:** 403 Forbidden errors when listing messages or folders.

**Why it happens:** Mail API requires different scopes than Admin API.

**How to avoid:**
- Request `ZohoMail.messages.READ` or `ZohoMail.messages.ALL` during auth
- Request `ZohoMail.folders.READ` for folder operations
- Request `ZohoMail.tags` for label operations
- Request `ZohoMail.accounts.READ` to list accounts
- Document all required scopes in auth/scopes.go

**Warning signs:** 403 responses for Mail endpoints while Admin endpoints work, "insufficient scope" errors

### Pitfall 8: Pagination Boundary Errors

**What goes wrong:** Missing last few messages in folder or duplicate messages in results.

**Why it happens:** Incorrect start/limit calculation when paginating through message lists.

**How to avoid:**
- Use existing PageIterator[T] pattern (already handles boundaries)
- Detect end-of-list when `len(results) < limit`
- Don't assume total count is available (API doesn't return it)
- Test with folders having exactly 200, 201, 400 messages

**Warning signs:** Inconsistent message counts, last page showing duplicates, missing recent messages

## Code Examples

Verified patterns from official sources:

### Initialize MailClient and Get Primary Account

```go
// Example: Creating MailClient with automatic account ID resolution
func newMailClient(cfg *config.Config, tokenSource oauth2.TokenSource) (*MailClient, error) {
    client, err := NewClient(cfg, tokenSource)
    if err != nil {
        return nil, err
    }

    mc := &MailClient{client: client}

    // Fetch primary account ID
    ctx := context.Background()
    accountID, err := mc.getPrimaryAccountID(ctx)
    if err != nil {
        return nil, err
    }

    mc.accountID = accountID
    return mc, nil
}
```
**Source:** Derived from [GET - Get all accounts of a user](https://www.zoho.com/mail/help/api/get-all-users-accounts.html)

### List Messages with Pagination

```go
// Example: List messages in Inbox with pagination
func listInboxMessages(mc *MailClient) error {
    ctx := context.Background()

    // Get Inbox folder ID
    inbox, err := mc.GetFolderByName(ctx, "Inbox")
    if err != nil {
        return err
    }

    // Create paginator
    iterator := NewPageIterator(func(start, limit int) ([]MessageSummary, error) {
        return mc.ListMessages(ctx, inbox.FolderID, start, limit)
    }, 50)

    // Fetch all messages
    messages, err := iterator.FetchAll()
    if err != nil {
        return err
    }

    fmt.Printf("Found %d messages\n", len(messages))
    return nil
}
```
**Source:** [GET - List Emails](https://www.zoho.com/mail/help/api/get-emails-list.html)

### Search Messages with Builder

```go
// Example: Search for unread emails from specific sender after date
func searchUnreadFrom(mc *MailClient, sender string, after time.Time) ([]MessageSummary, error) {
    ctx := context.Background()

    // Build search query
    query := NewSearchQuery().
        From(sender).
        IsUnread().
        DateAfter(after).
        Build()

    // Execute search
    messages, err := mc.SearchMessages(ctx, query, 1, 100)
    if err != nil {
        return nil, err
    }

    return messages, nil
}
```
**Source:** [GET - List emails based on Search parameters](https://www.zoho.com/mail/help/api/get-search-emails.html)

### Retrieve Full Message (Metadata + Content)

```go
// Example: Display complete message (headers + body)
func readMessage(mc *MailClient, folderID, messageID string) error {
    ctx := context.Background()

    // Get metadata
    metadata, err := mc.GetMessageMetadata(ctx, folderID, messageID)
    if err != nil {
        return err
    }

    // Get content
    content, err := mc.GetMessageContent(ctx, folderID, messageID)
    if err != nil {
        return err
    }

    // Display
    fmt.Printf("Subject: %s\n", metadata.Subject)
    fmt.Printf("From: %s\n", metadata.FromAddress)
    fmt.Printf("Date: %s\n", time.UnixMilli(metadata.SentDateInGMT).Format(time.RFC1123))
    fmt.Printf("\n%s\n", content.Content)

    return nil
}
```
**Source:** [GET - Get meta data of an email](https://www.zoho.com/mail/help/api/get-email-meta-data.html), [GET - Get email content](https://www.zoho.com/mail/help/api/get-email-content.html)

### Download Attachment with Streaming

```go
// Example: Download all attachments from a message
func downloadAttachments(mc *MailClient, folderID, messageID, destDir string) error {
    ctx := context.Background()

    // List attachments
    attachments, err := mc.ListAttachments(ctx, folderID, messageID)
    if err != nil {
        return err
    }

    // Download each attachment
    for _, att := range attachments {
        destPath := filepath.Join(destDir, att.AttachmentName)

        fmt.Fprintf(os.Stderr, "Downloading %s...\n", att.AttachmentName)

        if err := mc.DownloadAttachment(ctx, folderID, messageID, att.AttachmentID, destPath); err != nil {
            return fmt.Errorf("download %s: %w", att.AttachmentName, err)
        }
    }

    return nil
}
```
**Source:** [GET - Getting email attachment content](https://www.zoho.com/mail/help/api/get-attachment-content.html)

### List Folders and Labels

```go
// Example: Display all folders and labels
func showMailStructure(mc *MailClient) error {
    ctx := context.Background()

    // List folders
    folders, err := mc.ListFolders(ctx)
    if err != nil {
        return err
    }

    fmt.Println("Folders:")
    for _, folder := range folders {
        fmt.Printf("  %s (%s) [%s]\n", folder.FolderName, folder.FolderID, folder.FolderType)
    }

    // List labels
    labels, err := mc.ListLabels(ctx)
    if err != nil {
        return err
    }

    fmt.Println("\nLabels:")
    for _, label := range labels {
        fmt.Printf("  %s (%s)\n", label.LabelName, label.LabelColor)
    }

    return nil
}
```
**Source:** [GET - Get all folders](https://www.zoho.com/mail/help/api/get-all-folder-details.html), [Label API Details](https://www.zoho.com/mail/help/api/labels-api.html)

### View Thread Messages

```go
// Example: Display all messages in a thread
func viewThread(mc *MailClient, folderID, threadID string) error {
    ctx := context.Background()

    // Get thread messages (client-side filtering)
    messages, err := mc.GetThread(ctx, folderID, threadID)
    if err != nil {
        return err
    }

    fmt.Printf("Thread: %d messages\n\n", len(messages))

    for i, msg := range messages {
        fmt.Printf("[%d] Subject: %s\n", i+1, msg.Subject)
        fmt.Printf("    From: %s\n", msg.FromAddress)
        fmt.Printf("    Date: %s\n\n", time.UnixMilli(msg.ReceivedTime).Format(time.RFC1123))
    }

    return nil
}
```
**Source:** [Email Messages API](https://www.zoho.com/mail/help/api/email-api.html), [Threads API](https://www.zoho.com/mail/help/api/threads-api.html)

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single message endpoint | Three-tier (list/metadata/content) | Current API design | Optimize bandwidth, fetch only needed data |
| Structured query params | Zoho search syntax | Current API design | More flexible search, requires syntax learning |
| JSON attachment data | Binary octet-stream | Current API design | Efficient for large files, requires streaming |
| Thread-specific endpoint | Filter by threadId | Current API limitation | Requires client-side filtering, less efficient |
| Account autodetection | Explicit accountId | Current API design | Must fetch account list first, adds API call |

**Deprecated/outdated:**
- **Assuming thread GET endpoint**: Zoho Thread API only has PUT operations for updates, not GET for retrieval.
- **Fetching all messages in one call**: Message list has 200-per-request cap; must paginate for large folders.
- **Using organization-level auth for mail**: Mail API requires account-scoped operations, not organization-scoped like Admin API.

## Open Questions

1. **Does Zoho support server-side thread retrieval?**
   - What we know: Thread API documentation shows only PUT operations (mark read, apply labels)
   - What's unclear: Whether there's an undocumented GET endpoint for thread details
   - Recommendation: Test `GET /api/accounts/{accountId}/threads/{threadId}` during implementation; if 404, use client-side filtering as documented

2. **What are the exact rate limits for Mail API vs Admin API?**
   - What we know: Admin API documentation mentions 30 req/min for organization APIs
   - What's unclear: Whether Mail API (account-scoped) has same limit or separate bucket
   - Recommendation: Assume same 30 req/min limit, use existing RateLimitTransport, monitor 429 responses during testing

3. **Can we fetch multiple attachments in a single request?**
   - What we know: Documentation says "the user needs to use the api for each attachment with the respective details"
   - What's unclear: Whether batch download endpoint exists (not documented)
   - Recommendation: Implement single-attachment downloads as documented; consider parallel downloads with rate limiting

4. **How are message IDs and folder IDs scoped?**
   - What we know: API paths show `/accounts/{accountId}/folders/{folderId}/messages/{messageId}`
   - What's unclear: Are messageIds globally unique or only unique within an account/folder?
   - Recommendation: Always include folderId in message operations; don't assume messageId is globally unique

5. **What happens when requesting content for HTML-only vs plain-text messages?**
   - What we know: Content endpoint returns "HTML-formatted message body"
   - What's unclear: How plain-text-only messages are represented (wrapped in HTML? raw text?)
   - Recommendation: Test with both HTML and plain-text messages; may need to detect and handle both formats

6. **Are there search syntax operators for complex queries?**
   - What we know: Basic operators documented (from:, subject:, after:, before:, has:, is:)
   - What's unclear: Full list of operators, negation support (NOT), grouping (AND/OR)
   - Recommendation: Test Zoho web UI search to discover undocumented operators; document findings in search.go

7. **How long are folder IDs and label IDs stable?**
   - What we know: IDs are returned by list endpoints
   - What's unclear: Do IDs change across sessions? After folder renames? Can we cache them?
   - Recommendation: Fetch fresh on each CLI invocation (don't persist to disk); cache in-memory for session duration

## Sources

### Primary (HIGH confidence)
- [Zoho Mail API Index](https://www.zoho.com/mail/help/api/) - Complete API reference
- [GET - List Emails](https://www.zoho.com/mail/help/api/get-emails-list.html) - Message list endpoint
- [GET - Get email content of an Email](https://www.zoho.com/mail/help/api/get-email-content.html) - Content endpoint
- [GET - Get meta data of an email](https://www.zoho.com/mail/help/api/get-email-meta-data.html) - Metadata endpoint
- [GET - List emails based on Search parameters](https://www.zoho.com/mail/help/api/get-search-emails.html) - Search endpoint
- [GET - Get all folders](https://www.zoho.com/mail/help/api/get-all-folder-details.html) - Folders list endpoint
- [Label API Details](https://www.zoho.com/mail/help/api/labels-api.html) - Labels/tags API
- [Threads API](https://www.zoho.com/mail/help/api/threads-api.html) - Thread management
- [GET - Getting email attachment content](https://www.zoho.com/mail/help/api/get-attachment-content.html) - Attachment download
- [GET - Get all accounts of a user](https://www.zoho.com/mail/help/api/get-all-users-accounts.html) - Account list endpoint
- [Email Messages API](https://www.zoho.com/mail/help/api/email-api.html) - Messages API overview
- [Folder API Details](https://www.zoho.com/mail/help/api/folders-api.html) - Folders API overview
- [Accounts API](https://www.zoho.com/mail/help/api/account-api.html) - Account configuration API

### Secondary (MEDIUM confidence)
- [Zoho Mail REST APIs Getting Started](https://www.zoho.com/mail/help/api/getting-started-with-api.html) - General API setup
- [Zoho Mail API Guide - Overview](https://www.zoho.com/mail/help/api/overview.html) - High-level API overview

### Tertiary (LOW confidence - requires verification)
- Community discussions about thread retrieval - No GET endpoint confirmed, needs testing
- Search syntax operators beyond documented basics - Requires web UI testing to discover

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already integrated from Phases 1-3
- Mail API architecture: HIGH - Verified via official Zoho Mail API documentation
- Message retrieval patterns: HIGH - Three-tier approach documented in official API docs
- Search syntax: MEDIUM - Basic operators documented, advanced syntax requires testing
- Thread viewing: MEDIUM - Thread API only documents updates, not retrieval; client-side filtering required
- Attachment streaming: HIGH - Binary response confirmed in official docs, stdlib io.Copy pattern
- Pitfalls: MEDIUM - Based on API documentation analysis and Go best practices

**Research date:** 2026-02-14
**Valid until:** 2026-03-16 (30 days - Zoho Mail API is stable, infrequent breaking changes expected)
