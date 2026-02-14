---
phase: 02-admin-users-groups
plan: 01
subsystem: admin-api-client
tags: [admin, api-client, pagination, users, cli-commands]
dependency_graph:
  requires:
    - 01-01 (CLI scaffold, config, output formatters)
    - 01-02 (OAuth2, token cache, secrets store)
    - 01-03 (HTTP client with rate limiting)
  provides:
    - AdminClient with org ID resolution
    - Generic PageIterator for offset pagination
    - User list and get CLI commands
    - API type definitions for admin operations
  affects:
    - All future admin commands (users, groups, domains)
    - Phase 2 plans (02-02 through 02-05)
tech_stack:
  added:
    - AdminClient abstraction layer
    - Generic PageIterator[T] pattern
    - Zoho Admin API types (User, Group, Org)
  patterns:
    - Cached organization ID (zoid) resolution
    - newAdminClient helper function pattern
    - ZUID vs email auto-detection in CLI
    - PageIterator.FetchAll vs single page fetch
key_files:
  created:
    - internal/zoho/types.go (API request/response structs)
    - internal/zoho/admin_client.go (AdminClient with zoid cache)
    - internal/zoho/pagination.go (Generic PageIterator)
    - internal/cli/admin_users.go (CLI commands for users)
  modified:
    - internal/cli/cli.go (Added Admin command tree)
decisions:
  - title: AdminClient caches organization ID on initialization
    rationale: Zoho admin APIs require zoid in URLs (/api/organization/{zoid}/...). Fetching once and caching avoids redundant API calls.
  - title: Generic PageIterator with type parameter
    rationale: Go 1.24 supports generics. Reusable pagination logic for users, groups, and future admin resources.
  - title: GetUserByEmail iterates all users
    rationale: Zoho API lacks email-based user lookup. Pagination iterator makes this efficient.
  - title: newAdminClient helper in admin_users.go
    rationale: Mirrors auth.go pattern (secrets.NewStore -> auth.NewTokenCache -> client). Avoids duplication across admin commands.
  - title: ZUID vs email auto-detection in GetUserByIdentifier
    rationale: Better UX - users can run "zoh admin users get 12345" or "zoh admin users get user@example.com" without flags.
metrics:
  duration: 4 min
  completed_date: 2026-02-14
  tasks_completed: 2
  files_created: 4
  files_modified: 1
  commits: 2
---

# Phase 02 Plan 01: Admin API Client and User Commands

**One-liner:** AdminClient with org ID resolution, generic offset pagination, and working `zoh admin users list/get` commands

## Objective

Build the admin API client layer with organization ID resolution, offset-based pagination abstraction, API type definitions, and implement user list/get CLI commands.

## What Was Built

### Task 1: API types, AdminClient with org ID resolution, and PageIterator (Commit: d6a920f)

**Files Created:**
- `internal/zoho/types.go` - Complete API type definitions:
  - `OrgResponse` with nested Status and Data (zoid, CompanyName, UserCount, GroupCount)
  - `User` struct with 13 fields (ZUID, EmailAddress, DisplayName, Role, MailboxStatus, storage, access flags, LastLogin timestamp)
  - `UserListResponse` and `UserDetailResponse` for list/get operations
  - `CreateUserRequest` and `UpdateUserRequest` for future user management
  - `Group`, `GroupListResponse`, `GroupDetailResponse` for future group operations
  - `GroupMember`, `CreateGroupRequest`, add/remove member requests
  - `DeleteConfirmation` for delete operations
  - `APIError` with Error() implementation for proper error handling

- `internal/zoho/admin_client.go` - AdminClient wrapper:
  - `AdminClient` struct with `*Client` and cached `zoid` fields
  - `NewAdminClient()` creates Client and calls `getOrganizationID()` to cache zoid
  - `getOrganizationID()` - GET /api/organization/, decodes OrgResponse, returns zoid
  - `ListUsers(ctx, start, limit)` - GET /api/organization/{zoid}/accounts with pagination params
  - `GetUser(ctx, accountID)` - GET /api/organization/{zoid}/accounts/{accountID} for single user
  - `GetUserByEmail(ctx, email)` - Iterates all users via PageIterator to find email match
  - `GetUserByIdentifier(ctx, identifier)` - Helper that auto-detects ZUID (int64) vs email (string)
  - `parseErrorResponse(resp)` - Extracts APIError from response body with HTTP status code

- `internal/zoho/pagination.go` - Generic pagination:
  - `PageIterator[T any]` struct with fetchFunc, pageSize, current offset, done flag
  - `NewPageIterator[T](fetchFunc, pageSize)` - Creates iterator with default pageSize=50
  - `FetchAll()` - Fetches all pages until len(results) < pageSize, returns combined slice
  - `FetchPage(start)` - Fetches single page at given offset

**Verification:**
- `go build ./internal/zoho/...` - Compiled successfully
- `go vet ./internal/zoho/...` - No warnings

### Task 2: User list and get CLI commands (Commit: a837525)

**Files Created:**
- `internal/cli/admin_users.go` - User CLI commands:
  - `newAdminClient(cfg)` helper - Creates secrets store → token cache → AdminClient (mirrors auth.go pattern)
  - `AdminUsersListCmd` with `Limit int` (default 50) and `All bool` flags
  - `AdminUsersListCmd.Run()` - Uses PageIterator.FetchAll for `--all`, single page otherwise
  - List output via `fp.Formatter.PrintList()` with columns: Email, Name, Role, Status, ZUID
  - `AdminUsersGetCmd` with `Identifier string` arg (ZUID or email)
  - `AdminUsersGetCmd.Run()` - Parses identifier as int64 (ZUID) or treats as email, calls GetUser/GetUserByEmail
  - Get output via `fp.Formatter.Print()` with full user details
  - Error handling with CLIError and exit codes (ExitAuth, ExitAPIError, ExitGeneral)

**Files Modified:**
- `internal/cli/cli.go`:
  - Added `Admin AdminCmd` to CLI struct
  - Added `AdminCmd` struct with `Users AdminUsersCmd`
  - Added `AdminUsersCmd` struct with `List AdminUsersListCmd` and `Get AdminUsersGetCmd`
  - Creates command hierarchy: `zoh admin users list` and `zoh admin users get <id-or-email>`

**Verification:**
- `go build ./...` - Full project compiled successfully
- `go vet ./...` - No warnings
- `./zoh admin --help` - Shows "users" subcommand
- `./zoh admin users --help` - Shows "list" and "get" subcommands
- `./zoh admin users list --help` - Shows --limit and --all flags
- `./zoh admin users get --help` - Shows identifier argument

## Deviations from Plan

None - plan executed exactly as written.

## Success Criteria Met

- [x] AdminClient wraps Client with cached zoid and has ListUsers, GetUser, GetUserByEmail methods
- [x] PageIterator provides generic offset-based pagination (FetchAll, FetchPage)
- [x] All API types (User, Group, requests/responses) defined with proper JSON tags
- [x] `zoh admin users list` shows paginated users with Email, Name, Role, Status, ZUID columns
- [x] `zoh admin users get <id-or-email>` shows full user details
- [x] Error responses parsed into readable CLIError messages
- [x] All output modes (JSON, plain, rich) work for both list and get commands

## Technical Notes

### AdminClient Design

The AdminClient follows the established pattern from Phase 1:
1. **Initialization:** NewAdminClient(cfg, tokenSource) → NewClient() → getOrganizationID() → cache zoid
2. **API calls:** All admin endpoints use client.Do() (not DoMail) with paths like `/api/organization/{zoid}/...`
3. **Error handling:** parseErrorResponse() extracts APIError from response body, wraps with HTTP status code

### Pagination Pattern

PageIterator provides generic offset-based pagination:
- **FetchAll()** - Keeps fetching pages until len(results) < pageSize (last page indicator)
- **FetchPage(start)** - Fetches single page at given offset
- **Default page size:** 50 (matches Zoho API recommendations)

Used in two contexts:
1. **CLI --all flag:** Fetches all pages transparently
2. **GetUserByEmail:** Iterates all users to find email match (no email-based API endpoint)

### CLI Command Hierarchy

```
zoh admin
  └─ users
      ├─ list [--limit N] [--all]
      └─ get <zuid-or-email>
```

The `newAdminClient()` helper ensures consistent initialization across all admin commands (future-proofing for groups, domains, etc.).

## Dependencies for Next Plans

This plan establishes the foundation for all Phase 2 admin operations:
- **02-02 (User create/update/delete)** - Will use AdminClient, add CreateUser/UpdateUser/DeleteUser methods
- **02-03 (Group operations)** - Will use AdminClient, add ListGroups/GetGroup/CreateGroup/etc methods
- **02-04 (Group member management)** - Will use AdminClient, add AddGroupMember/RemoveGroupMember methods
- **02-05 (Phase verification)** - Will test all admin commands end-to-end

## Commits

| Task | Name                                         | Commit  | Files                                                                     |
| ---- | -------------------------------------------- | ------- | ------------------------------------------------------------------------- |
| 1    | AdminClient with org ID and pagination       | d6a920f | types.go, admin_client.go, pagination.go                                  |
| 2    | User list and get CLI commands               | a837525 | admin_users.go, cli.go                                                    |

## Self-Check: PASSED

Verification of created files:
- [x] internal/zoho/types.go exists
- [x] internal/zoho/admin_client.go exists
- [x] internal/zoho/pagination.go exists
- [x] internal/cli/admin_users.go exists
- [x] internal/cli/cli.go modified

Verification of commits:
- [x] Commit d6a920f exists (Task 1)
- [x] Commit a837525 exists (Task 2)

All files created, all commits recorded, plan complete.
