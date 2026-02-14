# Requirements: zoh CLI

## v1 Requirements

### Authentication & Configuration

- [ ] **AUTH-01**: User can authenticate with Zoho via interactive OAuth2 flow (browser opens, localhost callback)
- [ ] **AUTH-02**: User can authenticate via manual flow (paste redirect URL) for environments without browser
- [ ] **AUTH-03**: User can authenticate via headless/remote two-step flow for servers
- [ ] **AUTH-04**: Refresh tokens are stored in OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager)
- [ ] **AUTH-05**: Encrypted file-based fallback exists for WSL, containers, and headless Linux without D-Bus
- [ ] **AUTH-06**: Access tokens auto-refresh transparently using stored refresh tokens
- [ ] **AUTH-07**: Token cache uses file locking to support concurrent CLI invocations
- [ ] **AUTH-08**: User can configure Zoho region (.com, .eu, .in, .com.au, .jp) and all API endpoints resolve correctly
- [ ] **AUTH-09**: User can manage config via `zoh config get/set/unset/list/path` commands
- [ ] **AUTH-10**: Config stored in XDG-compliant location with JSON5 format (supports comments)
- [ ] **AUTH-11**: User can log out and remove stored credentials via `zoh auth logout`
- [ ] **AUTH-12**: User can list stored accounts and validate tokens via `zoh auth list [--check]`

### Admin — User Management

- [ ] **ADMIN-USR-01**: User can list all org users with pagination
- [ ] **ADMIN-USR-02**: User can get detailed info for a specific user by ID or email
- [ ] **ADMIN-USR-03**: User can create a new user in the organization
- [ ] **ADMIN-USR-04**: User can update user details (name, role, status)
- [ ] **ADMIN-USR-05**: User can activate/deactivate a user account
- [ ] **ADMIN-USR-06**: User can delete a user from the organization

### Admin — Group Management

- [ ] **ADMIN-GRP-01**: User can list all groups in the organization
- [ ] **ADMIN-GRP-02**: User can get group details including members
- [ ] **ADMIN-GRP-03**: User can create a new group
- [ ] **ADMIN-GRP-04**: User can update group settings (name, description, permissions)
- [ ] **ADMIN-GRP-05**: User can add/remove members from a group
- [ ] **ADMIN-GRP-06**: User can delete a group

### Admin — Domain Management

- [ ] **ADMIN-DOM-01**: User can list all domains in the organization
- [ ] **ADMIN-DOM-02**: User can get domain details including DNS verification status
- [ ] **ADMIN-DOM-03**: User can add a new domain
- [ ] **ADMIN-DOM-04**: User can verify domain ownership (display required DNS records)
- [ ] **ADMIN-DOM-05**: User can view/update domain-level settings

### Admin — Audit & Security

- [ ] **ADMIN-AUD-01**: User can view login audit logs with date range filtering
- [ ] **ADMIN-AUD-02**: User can view admin action logs with date range filtering
- [ ] **ADMIN-AUD-03**: User can list active sessions/devices for users
- [ ] **ADMIN-AUD-04**: User can view security policy settings (2FA status, password policies)

### Mail — Read Operations

- [ ] **MAIL-READ-01**: User can list messages in a folder with pagination
- [ ] **MAIL-READ-02**: User can get a specific message by ID (headers, body, metadata)
- [ ] **MAIL-READ-03**: User can search messages by query (subject, from, date range, etc.)
- [ ] **MAIL-READ-04**: User can list mail folders
- [ ] **MAIL-READ-05**: User can list labels/tags
- [ ] **MAIL-READ-06**: User can view threads (grouped messages)
- [ ] **MAIL-READ-07**: User can download attachments from a message

### Mail — Send Operations

- [ ] **MAIL-SEND-01**: User can compose and send a new email with to/cc/bcc, subject, body
- [ ] **MAIL-SEND-02**: User can reply to a message (reply/reply-all)
- [ ] **MAIL-SEND-03**: User can forward a message
- [ ] **MAIL-SEND-04**: User can attach files when sending
- [ ] **MAIL-SEND-05**: User can send HTML or plain text body

### Mail — Settings

- [ ] **MAIL-SET-01**: User can view/update email signatures
- [ ] **MAIL-SET-02**: User can view/update vacation auto-reply settings
- [ ] **MAIL-SET-03**: User can view/update display name and sender aliases
- [ ] **MAIL-SET-04**: User can view/update mail forwarding settings

### Mail — Admin

- [ ] **MAIL-ADM-01**: User can view/update retention policies
- [ ] **MAIL-ADM-02**: User can view/update spam filter settings
- [ ] **MAIL-ADM-03**: User can manage email allowlists and blocklists
- [ ] **MAIL-ADM-04**: User can view mail delivery logs/status

### CLI UX & Infrastructure

- [ ] **UX-01**: CLI supports three output modes: JSON (--json), plain (--plain), rich (default TTY)
- [ ] **UX-02**: Data goes to stdout, hints/progress/errors go to stderr
- [ ] **UX-03**: Desire paths exist: action-first shortcuts (zoh send, zoh ls) alongside service hierarchy
- [ ] **UX-04**: Stable, documented exit codes for all error categories (auth, not found, rate limit, etc.)
- [ ] **UX-05**: `--results-only` flag strips JSON envelope, returns data array
- [ ] **UX-06**: `--no-input` flag disables all prompts (fails instead of asking)
- [ ] **UX-07**: `--dry-run` flag shows what would happen without executing
- [ ] **UX-08**: `zoh schema [command]` emits machine-readable command tree as JSON
- [ ] **UX-09**: Built-in rate limiter respects Zoho's 30 req/min limit with backoff
- [ ] **UX-10**: Shell completion support (bash, zsh, fish)
- [ ] **UX-11**: `--force` flag skips destructive operation confirmations

## v2 Requirements (Deferred)

- [ ] Bulk operations via CSV (create/update/delete users in batch) — high value but needs stable single-op first
- [ ] Multi-account support with aliases — v1 targets single org
- [ ] Email filter/rule management — NO API exists, would need Zoho to add
- [ ] eDiscovery / compliance search — may need separate Zoho eProtect API
- [ ] Export reports (user activity, mail stats) — compound feature on top of audit logs
- [ ] `--enable-commands` sandboxing for agent use — nice to have after core works
- [ ] Homebrew / package manager distribution — figure out after v1 is stable

## Out of Scope

- Zoho CRM — separate service, community contribution later
- Zoho Desk — separate service, community contribution later
- Zoho Projects — separate service, community contribution later
- All other Zoho apps — extensible architecture allows future addition
- Web UI / TUI dashboard — this is a CLI, not an interactive app
- Screen-scraping for API gaps — if Zoho doesn't expose it via API, print helpful redirect to web UI
- Email rendering in terminal — show raw text/HTML, don't try to render rich email

## Traceability

<!-- Updated by roadmap creation — maps REQ-IDs to phases -->

| Requirement | Phase | Status |
|-------------|-------|--------|
| (populated by roadmapper) | | |

---
*Requirements defined: 2026-02-14*
*Total v1 requirements: 55*
