# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** Fast, reliable access to Zoho Admin and Mail operations from the terminal
**Current focus:** Phase 2 complete -- Ready for Phase 3

## Current Position

Phase: 5 of 6 (Mail Send, Settings, Admin)
Plan: 2 of 3 in current phase
Status: Complete
Last activity: 2026-02-14 -- Phase 5 Plan 2 complete (mail settings management)

Progress: [██████░░░░] 66.7% (12/18 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 12
- Average duration: 4.3 min
- Total execution time: 52 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 3/3 | 14 min | 4.7 min |
| 02 | 3/3 | 15 min | 5.0 min |
| 03 | 2/2 | 10 min | 5.0 min |
| 04 | 2/2 | 7 min | 3.5 min |
| 05 | 2/3 | 6 min | 3.0 min |

**Recent Executions:**

| Phase-Plan | Duration | Tasks | Files | Date |
|------------|----------|-------|-------|------|
| 05-02 | 4 min | 2 | 4 | 2026-02-14 |
| 05-01 | 2 min | 2 | 4 | 2026-02-14 |
| 04-02 | 4 min | 2 | 4 | 2026-02-14 |
| 04-01 | 3 min | 2 | 5 | 2026-02-14 |
| 03-02 | 5 min | 2 | 5 | 2026-02-14 |

**Recent Trend:**
- Last 3 plans: 3.3 min average
- Trend: Excellent velocity

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Roadmap: 6 phases derived from 64 requirements (auth -> admin users/groups -> admin domains/audit -> mail read -> mail send/settings/admin -> CLI polish)
- Roadmap: UX infrastructure (output modes, exit codes, rate limiter) placed in Phase 1 since all commands depend on it
- Roadmap: Mail read (Phase 4) depends only on Phase 1, not Phase 3 -- can parallelize with admin domains/audit
- 01-01: Go 1.24 required for lipgloss v2 (auto-upgraded from 1.22)
- 01-01: FormatterProvider wrapper for Kong interface binding (Kong can't bind interfaces directly)
- 01-01: Empty region default resolves to 'us' in BeforeApply (CLI flag > config > us)
- 01-02: 99designs/keyring for OS credential storage (macOS, Linux, Windows support)
- 01-02: AES-256-GCM encrypted file fallback for WSL/headless (sha256 key derivation, future: scrypt/argon2)
- 01-02: Zoho OAuth2 quirks: comma-separated scopes, access_type=offline, prompt=consent
- 01-02: gofrs/flock for file-locked token cache (prevents concurrent refresh stampede)
- 01-02: 5-minute proactive token refresh window (reduces auth errors during API calls)
- [Phase 01-03]: 25 req/min rate limit budget (under Zoho's 30 req/min limit) — Safety margin for API calls
- [Phase 01-03]: Global --region flag instead of command-specific override — Avoids duplicate flags, cleaner UX
- [Phase 02-01]: AdminClient caches organization ID on initialization — Zoho admin APIs require zoid in URLs, fetching once avoids redundant API calls
- [Phase 02-01]: Generic PageIterator with type parameter — Go 1.24 generics enable reusable pagination for users, groups, and future resources
- [Phase 02-01]: GetUserByEmail iterates all users — Zoho API lacks email-based lookup, PageIterator makes this efficient
- [Phase 02-01]: newAdminClient helper in admin_users.go — Mirrors auth.go pattern (secrets store → token cache → client)
- [Phase 02-01]: ZUID vs email auto-detection in CLI — Better UX, users can use "zoh admin users get 12345" or "user@example.com" without flags
- [Phase 02-02]: resolveUserID returns both ZUID and User object — Commands need ZUID for API calls AND user email for confirmation messages, single lookup instead of two API calls
- [Phase 02-02]: os.Stderr for confirmation messages — Matches pattern from admin_groups.go and configcmd.go, status messages to stderr, data to stdout
- [Phase 02-02]: Delete requires --confirm flag — Deletion is permanent, prevent accidental data loss
- [Phase 02-03]: Batch size of 50 for AddGroupMembers — Provides safety margin for Zoho API bulk operation limits while maintaining efficiency
- [Phase 02-03]: ShowMembers default true in groups get — Members are core to group utility, better UX to show by default with opt-out flag
- [Phase 02-03]: Required --confirm for group deletion — Kong's required flag ensures explicit user intent for permanent destructive action
- [Phase 03-01]: Domain list does not use pagination — API returns all domains in single response (typical orgs have <10 domains)
- [Phase 03-01]: Boolean fields display raw true/false values — output.Column doesn't support Transform field (planned feature not yet implemented)
- [Phase 03-01]: User-friendly flag mappings — CLI flags (txt/cname/html, enable-hosting/etc) mapped to API mode values in command layer
- [Phase 03-01]: Verification codes printed to stderr — Ensures users see critical DNS setup info regardless of output mode
- [Phase 03-01]: Validation in AdminClient methods — Method/mode validation happens before API call for better error messages
- [Phase 03-02]: Display structs for timestamp formatting — Column type doesn't support Transform field, pre-formatting is cleaner than reflection-based changes
- [Phase 03-02]: 90-day validation in AdminClient — Login history API limitation, fail fast with clear error vs cryptic API response
- [Phase 03-02]: Informational commands for sessions/security — No documented API endpoints, web console redirect provides better UX than "not implemented"
- [Phase 03-02]: Empty string enum default — Kong requires default value for optional enums, allows --search-by to be truly optional
- [Phase 04-01]: MailClient caches primary accountID on initialization (mirrors AdminClient's cached zoid pattern)
- [Phase 04-01]: All mail requests use DoMail (MailBase URL) instead of Do (APIBase URL) - fundamental architectural difference from AdminClient
- [Phase 04-01]: Display structs for timestamp formatting (unix ms to human-readable) - Column doesn't support Transform field
- [Phase 04-01]: Three-tier message retrieval: GetMessageMetadata + GetMessageContent = complete view (two API calls)
- [Phase 04-02]: SearchQuery builder uses method chaining for fluent API construction
- [Phase 04-02]: GetThread uses client-side filtering with configurable scan limit (no dedicated API endpoint)
- [Phase 04-02]: DownloadAttachment streams binary response with io.Copy (no memory buffering)
- [Phase 04-02]: Auto-filename detection for downloads via ListAttachments lookup when --output-path not specified
- [Phase 05-01]: Attachment upload uses application/octet-stream Content-Type (not multipart/form-data) - bypasses DoMail to avoid automatic JSON header
- [Phase 05-01]: Two-step attachment workflow: upload first, get reference, include in send request
- [Phase 05-01]: Action field in SendEmailRequest determines operation type (reply/replyall/forward)
- [Phase 05-01]: Reply-all combines original ToAddress and CcAddress into new CcAddress field
- [Phase 05-02]: Mode-based PUT operations use updateAccountSettings helper for consistent request handling
- [Phase 05-02]: Vacation date format validated in CLI before API call (MM/DD/YYYY HH:MM:SS layout)
- [Phase 05-02]: Forwarding is read-only (research confidence LOW for update operations)
- [Phase 05-02]: VacationResponse and ForwardDetails use json.RawMessage for flexible nested object parsing

### Pending Todos

None yet.

### Blockers/Concerns

- Research flag (RESOLVED): Phase 2 research confirmed all admin ops use Zoho Mail API — no separate Directory API needed
- Research flag (RESOLVED): Phase 5 attachment upload - confirmed application/octet-stream Content-Type works, not multipart/form-data
- Plan 03-02 runtime bug: AdminAuditSMTPLogsCmd.SearchBy enum panic ("enum value is only valid if it is either required or has a valid default value") — Kong validation issue, needs fix before audit commands are usable

## Session Continuity

Last session: 2026-02-14T20:55:58Z
Stopped at: Completed 05-02-PLAN.md
Resume file: None
Next: Phase 5 Plan 3 (mail administration operations)
