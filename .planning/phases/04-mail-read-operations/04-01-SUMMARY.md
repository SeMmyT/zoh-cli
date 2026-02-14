---
phase: 04-mail-read-operations
plan: 01
subsystem: mail-client
tags: [mail-api, client-infrastructure, folders, labels, messages, mail-read]
dependency_graph:
  requires: [phase-01-auth-and-ux]
  provides: [mail-client, mail-types, mail-folder-commands, mail-label-commands, mail-message-commands]
  affects: [phase-04-plan-02]
tech_stack:
  added: [zoho-mail-api]
  patterns: [three-tier-retrieval, cached-account-id, folder-name-resolution, display-structs, html-stripping]
key_files:
  created:
    - internal/zoho/mail_types.go
    - internal/zoho/mail_client.go
    - internal/cli/mail_folders.go
    - internal/cli/mail_messages.go
  modified:
    - internal/cli/cli.go
decisions:
  - "MailClient caches primary accountID on initialization (mirrors AdminClient's cached zoid pattern)"
  - "Separate mail_types.go file keeps admin vs mail types isolated"
  - "All mail requests use DoMail (MailBase URL) instead of Do (APIBase URL) - fundamental architectural difference from AdminClient"
  - "GetFolderByName uses case-insensitive comparison via strings.EqualFold"
  - "ToAddress/CcAddress are strings (comma-separated) not arrays - matches actual API response format"
  - "MessageListRow display struct for timestamp formatting (unix ms to human-readable)"
  - "MessageDetail combines metadata + content from two API calls (three-tier pattern)"
  - "HTML stripping for plain/rich modes using simple regex - adequate for terminal display, JSON returns raw HTML"
  - "formatBytes, formatPriority, formatBool helpers for human-readable output"
  - "Folder resolution: try GetFolderByName first, fall back to treating input as folder ID"
metrics:
  duration: 3 min
  tasks_completed: 2
  files_created: 4
  files_modified: 1
  commits: 2
  lines_added: 741
  completed_date: 2026-02-14
---

# Phase 04 Plan 01: Mail Client Infrastructure and Read Commands Summary

**One-liner:** MailClient with cached account ID resolution, mail type definitions, and folder/label/message CLI commands using DoMail for Zoho Mail API access.

## What Was Built

### Task 1: Mail API Types and MailClient (Commit: f0da8b3)

Created the foundational mail client infrastructure:

**internal/zoho/mail_types.go** - All mail API type definitions:
- `MailAccount`, `MailAccountListResponse` - Account listing
- `Folder`, `FolderListResponse` - Folder structure with unread/message counts
- `Label`, `LabelListResponse` - Label/tag system
- `MessageSummary`, `MessageListResponse` - Message list view with timestamps, attachments, priority
- `MessageMetadata`, `MessageMetadataResponse` - Full message headers (from/to/cc, dates, size)
- `MessageContent`, `MessageContentResponse` - HTML body content
- `Attachment`, `AttachmentListResponse` - Attachment metadata (reserved for Phase 5)

**internal/zoho/mail_client.go** - MailClient wrapping Client:
- `NewMailClient` - Creates client, calls getPrimaryAccountID to cache accountID
- `getPrimaryAccountID` - Fetches primary account from `/api/accounts`, caches ID
- `parseErrorResponse` - Same pattern as AdminClient for error handling
- `ListFolders` - GET `/api/accounts/{accountId}/folders`
- `GetFolderByName` - Iterates folders, case-insensitive match via `strings.EqualFold`
- `ListLabels` - GET `/api/accounts/{accountId}/labels`
- `ListMessages` - GET `/api/accounts/{accountId}/messages/view?folderId={id}&start={start}&limit={limit}`
- `GetMessageMetadata` - GET `/api/accounts/{accountId}/folders/{folderId}/messages/{messageId}/details`
- `GetMessageContent` - GET `/api/accounts/{accountId}/folders/{folderId}/messages/{messageId}/content`

All methods use `client.DoMail` (MailBase URL: `https://mail.zoho.eu`) instead of `client.Do` (APIBase URL: `https://www.zohoapis.eu`). This is the critical architectural difference from AdminClient.

### Task 2: CLI Commands (Commit: 3c75e7c)

Created comprehensive mail commands:

**internal/cli/mail_folders.go**:
- `newMailClient` helper - Mirrors `newAdminClient` pattern (secrets → token cache → MailClient)
- `MailFoldersListCmd` - Lists all folders with Name, Type, Path, Messages, Unread columns
- `MailLabelsListCmd` - Lists all labels with Name, Color columns

**internal/cli/mail_messages.go**:
- `MessageListRow` display struct - Formats ReceivedTime (unix ms → "2006-01-02 15:04"), HasAttachment (bool → "Y"/"")
- `MessageDetail` display struct - Combines metadata + content, formats dates, sizes, priority, strips HTML
- `MailMessagesListCmd` - Lists messages with pagination (--folder, --limit, --all via PageIterator)
- `MailMessagesGetCmd` - Fetches metadata + content (two API calls), displays combined view
- `formatBytes` - Converts bytes to human-readable (B, KB, MB, etc.)
- `formatPriority` - Maps int to string (0=Normal, 1=High)
- `formatBool` - Converts bool to Yes/No
- `formatBody` - Strips HTML tags for plain/rich modes (JSON returns raw HTML)

**internal/cli/cli.go** - Registered mail command tree:
```
zoh mail folders list
zoh mail labels list
zoh mail messages list [--folder Inbox] [--limit 50] [--all]
zoh mail messages get <id> --folder <name-or-id>
```

## Deviations from Plan

None - plan executed exactly as written.

## Key Patterns Established

1. **MailClient mirrors AdminClient** - Same initialization pattern (cached ID), error handling, method structure
2. **DoMail vs Do separation** - MailClient uses MailBase URL, AdminClient uses APIBase URL
3. **Display structs for formatting** - MessageListRow and MessageDetail handle timestamp/size/bool conversions before printing (Column doesn't support Transform field)
4. **Three-tier message retrieval** - GetMessageMetadata + GetMessageContent = complete message view
5. **Folder resolution flexibility** - Accept folder name OR folder ID, try name lookup first
6. **HTML stripping for terminals** - Simple regex removes tags for plain/rich, JSON preserves raw HTML
7. **PageIterator reuse** - Same pagination pattern as admin commands (messages support --all flag)

## Technical Decisions

- **Cached accountID**: MailClient resolves and caches primary account ID on initialization (same rationale as AdminClient's cached zoid - Zoho mail APIs require accountId in URLs)
- **Separate mail_types.go**: Keeps admin types (in types.go) separate from mail types - cleaner organization as codebase grows
- **ToAddress/CcAddress as strings**: API returns comma-separated strings, not arrays - using `string` type matches reality
- **Case-insensitive folder lookup**: `strings.EqualFold` for GetFolderByName - better UX (users can type "inbox" or "Inbox")
- **Simple HTML stripping**: `regexp.MustCompile("<[^>]*>")` adequate for terminal display - more sophisticated parsing not needed for read-only view

## Verification Results

All verification steps passed:
- ✓ `go build ./...` - Entire project compiles
- ✓ `go vet ./...` - No warnings
- ✓ `./zoh mail --help` - Shows folders, labels, messages subcommands
- ✓ `./zoh mail folders list --help` - Shows command help (no required flags)
- ✓ `./zoh mail labels list --help` - Shows command help (no required flags)
- ✓ `./zoh mail messages list --help` - Shows --folder, --limit, --all flags
- ✓ `./zoh mail messages get --help` - Shows message-id arg and --folder flag (required)

## Commits

| Hash    | Type | Description                                                      |
|---------|------|------------------------------------------------------------------|
| f0da8b3 | feat | Mail types and MailClient with account ID resolution            |
| 3c75e7c | feat | Mail CLI commands for folders, labels, and messages             |

## Files Changed

**Created (4 files):**
- `internal/zoho/mail_types.go` (151 lines) - All mail API types
- `internal/zoho/mail_client.go` (220 lines) - MailClient with DoMail methods
- `internal/cli/mail_folders.go` (113 lines) - Folder/label list commands
- `internal/cli/mail_messages.go` (257 lines) - Message list/get commands

**Modified (1 file):**
- `internal/cli/cli.go` (+29 lines) - Mail command tree registration

## Dependencies

**Requires:**
- Phase 01 (auth-and-ux): Client.DoMail method, Formatter, output modes, PageIterator
- No dependency on Phase 02 or 03 (admin commands)

**Provides:**
- MailClient infrastructure for Phase 4 Plan 2 (message operations: move, delete, update, search)
- Mail type definitions for all future mail commands
- Folder/label listing (MAIL-READ-04, MAIL-READ-05 requirements)
- Message list/get (MAIL-READ-01, MAIL-READ-02 requirements)

**Affects:**
- Phase 04 Plan 02 will reuse MailClient, mail types, and folder resolution helpers

## Self-Check: PASSED

**Files exist:**
- FOUND: internal/zoho/mail_types.go
- FOUND: internal/zoho/mail_client.go
- FOUND: internal/cli/mail_folders.go
- FOUND: internal/cli/mail_messages.go

**Commits exist:**
- FOUND: f0da8b3
- FOUND: 3c75e7c

**Binary works:**
- FOUND: ./zoh mail --help shows all subcommands
- FOUND: All help outputs show correct flags and arguments
