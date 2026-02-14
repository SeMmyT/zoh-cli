# Roadmap: zoh CLI

## Overview

This roadmap delivers a complete Go CLI for Zoho Admin and Mail operations across 6 phases. The build starts with auth and infrastructure (the blocking dependency for everything), validates the full API pipeline through admin user/group operations (the primary pain point), completes the admin story with domains and audit, then delivers mail read and send capabilities, and finishes with CLI polish and power-user shortcuts. Each phase delivers a coherent, independently verifiable capability.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation & Authentication** - Auth infrastructure, config, output formatting, and rate limiting that all commands depend on (Completed: 2026-02-14)
- [ ] **Phase 2: Admin -- Users & Groups** - Full user and group management, proving the API pipeline end-to-end
- [ ] **Phase 3: Admin -- Domains & Audit** - Domain management with DNS verification, plus audit and security log access
- [ ] **Phase 4: Mail -- Read Operations** - Read, search, and organize email from the terminal
- [ ] **Phase 5: Mail -- Send, Settings & Admin** - Compose, reply, forward email with attachments, plus mail settings and admin controls
- [ ] **Phase 6: CLI Polish & Power User UX** - Desire-path shortcuts, dry-run, shell completion, and scripting flags

## Phase Details

### Phase 1: Foundation & Authentication
**Goal**: Users can authenticate with Zoho across any region and the CLI infrastructure (output formatting, rate limiting, config management) is ready for commands to build on
**Depends on**: Nothing (first phase)
**Requirements**: AUTH-01, AUTH-02, AUTH-03, AUTH-04, AUTH-05, AUTH-06, AUTH-07, AUTH-08, AUTH-09, AUTH-10, AUTH-11, AUTH-12, UX-01, UX-02, UX-04, UX-09
**Success Criteria** (what must be TRUE):
  1. User can run `zoh auth login` and complete OAuth2 flow (browser-based or manual paste) to authenticate with any Zoho region
  2. Tokens are stored securely in OS keyring (or encrypted file fallback on WSL/headless) and access tokens refresh transparently without user intervention
  3. User can manage configuration via `zoh config get/set/list/path` and config is stored in XDG-compliant location
  4. All commands produce correct output in JSON, plain, and rich modes (data to stdout, errors/hints to stderr) with documented exit codes
  5. Concurrent CLI invocations do not corrupt token state, and API calls respect the 30 req/min rate limit with automatic backoff
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md -- Project scaffold, Kong CLI setup, config system, and output formatter framework
- [x] 01-02-PLAN.md -- OAuth2 authentication flows, keyring storage, token refresh, and file locking
- [x] 01-03-PLAN.md -- Region-aware HTTP client, rate limiter, auth commands (login/logout/list), and config commands

### Phase 2: Admin -- Users & Groups
**Goal**: Users can manage org users and groups entirely from the terminal, replacing the slow Zoho web UI for everyday admin tasks
**Depends on**: Phase 1
**Requirements**: ADMIN-USR-01, ADMIN-USR-02, ADMIN-USR-03, ADMIN-USR-04, ADMIN-USR-05, ADMIN-USR-06, ADMIN-GRP-01, ADMIN-GRP-02, ADMIN-GRP-03, ADMIN-GRP-04, ADMIN-GRP-05, ADMIN-GRP-06
**Success Criteria** (what must be TRUE):
  1. User can list all org users with pagination and get detailed info for any user by ID or email
  2. User can create, update, activate/deactivate, and delete users in the organization
  3. User can list groups, view group details with members, and create/update/delete groups
  4. User can add and remove members from any group
  5. All admin commands produce correctly formatted output in all three modes (JSON, plain, rich)
**Plans**: TBD

Plans:
- [ ] 02-01: Admin API client layer, pagination abstraction, and user list/get commands
- [ ] 02-02: User create/update/activate/deactivate/delete commands
- [ ] 02-03: Group management commands (list, get, create, update, delete, member add/remove)

### Phase 3: Admin -- Domains & Audit
**Goal**: Users can manage domains (including DNS verification) and access audit/security logs without touching the Zoho web console
**Depends on**: Phase 2
**Requirements**: ADMIN-DOM-01, ADMIN-DOM-02, ADMIN-DOM-03, ADMIN-DOM-04, ADMIN-DOM-05, ADMIN-AUD-01, ADMIN-AUD-02, ADMIN-AUD-03, ADMIN-AUD-04
**Success Criteria** (what must be TRUE):
  1. User can list all domains with their verification status and view detailed domain settings
  2. User can add a new domain and see the required DNS records for verification
  3. User can view login audit logs and admin action logs filtered by date range
  4. User can list active sessions/devices for users and view security policy settings (2FA, password policies)
**Plans**: TBD

Plans:
- [ ] 03-01: Domain management commands (list, get, add, verify, update settings)
- [ ] 03-02: Audit and security commands (login logs, admin logs, sessions, security policies)

### Phase 4: Mail -- Read Operations
**Goal**: Users can read, search, and organize email entirely from the terminal
**Depends on**: Phase 1
**Requirements**: MAIL-READ-01, MAIL-READ-02, MAIL-READ-03, MAIL-READ-04, MAIL-READ-05, MAIL-READ-06, MAIL-READ-07
**Success Criteria** (what must be TRUE):
  1. User can list messages in any folder with pagination and read a specific message (headers, body, metadata)
  2. User can search messages by query (subject, sender, date range) and view threaded conversations
  3. User can list mail folders and labels/tags
  4. User can download attachments from a message to local disk
**Plans**: TBD

Plans:
- [ ] 04-01: Mail API client layer, folder/label listing, and message list/get commands
- [ ] 04-02: Message search, thread view, and attachment download

### Phase 5: Mail -- Send, Settings & Admin
**Goal**: Users can compose and send email (with attachments), manage mail settings, and administer mail policies from the terminal
**Depends on**: Phase 4
**Requirements**: MAIL-SEND-01, MAIL-SEND-02, MAIL-SEND-03, MAIL-SEND-04, MAIL-SEND-05, MAIL-SET-01, MAIL-SET-02, MAIL-SET-03, MAIL-SET-04, MAIL-ADM-01, MAIL-ADM-02, MAIL-ADM-03, MAIL-ADM-04
**Success Criteria** (what must be TRUE):
  1. User can compose and send a new email with to/cc/bcc, subject, and plain text or HTML body
  2. User can reply, reply-all, and forward messages, with optional file attachments
  3. User can view and update email signatures, vacation auto-reply, display name/aliases, and forwarding settings
  4. User can view and update retention policies, spam filter settings, allowlists/blocklists, and delivery logs
**Plans**: TBD

Plans:
- [ ] 05-01: Send email commands (compose, reply, forward) with attachment support
- [ ] 05-02: Mail settings commands (signatures, vacation, display name, forwarding)
- [ ] 05-03: Mail admin commands (retention, spam, allowlists/blocklists, delivery logs)

### Phase 6: CLI Polish & Power User UX
**Goal**: Power users get shortcuts, scripting flags, and shell integration that make zoh fast and composable in pipelines
**Depends on**: Phase 2, Phase 4
**Requirements**: UX-03, UX-05, UX-06, UX-07, UX-08, UX-10, UX-11
**Success Criteria** (what must be TRUE):
  1. User can use action-first desire-path shortcuts (`zoh send`, `zoh ls users`) alongside the full service hierarchy
  2. User can use `--results-only` to strip JSON envelope, `--no-input` to disable prompts, and `--force` to skip confirmations
  3. User can use `--dry-run` on any mutating command to preview what would happen without executing
  4. User can run `zoh schema [command]` to get a machine-readable command tree as JSON
  5. Shell completion works for bash, zsh, and fish
**Plans**: TBD

Plans:
- [ ] 06-01: Desire-path shortcuts and action-first command aliases
- [ ] 06-02: Scripting flags (--results-only, --no-input, --force, --dry-run) and schema command
- [ ] 06-03: Shell completion generation (bash, zsh, fish)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6
(Note: Phase 4 depends on Phase 1, not Phase 3 -- mail read can begin after auth is complete, in parallel with admin domains/audit if desired.)

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation & Authentication | 3/3 | ✓ Complete | 2026-02-14 |
| 2. Admin -- Users & Groups | 0/3 | Not started | - |
| 3. Admin -- Domains & Audit | 0/2 | Not started | - |
| 4. Mail -- Read Operations | 0/2 | Not started | - |
| 5. Mail -- Send, Settings & Admin | 0/3 | Not started | - |
| 6. CLI Polish & Power User UX | 0/3 | Not started | - |
