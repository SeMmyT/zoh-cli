# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** Fast, reliable access to Zoho Admin and Mail operations from the terminal
**Current focus:** Phase 2 complete -- Ready for Phase 3

## Current Position

Phase: 2 of 6 (Admin -- Users & Groups)
Plan: 3 of 3 in current phase
Status: Complete
Last activity: 2026-02-14 -- Phase 2 complete (all admin user & group commands)

Progress: [███░░░░░░░] 33.3% (6/18 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 6
- Average duration: 5.0 min
- Total execution time: 29 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 3/3 | 14 min | 4.7 min |
| 02 | 3/3 | 15 min | 5.0 min |

**Recent Executions:**

| Phase-Plan | Duration | Tasks | Files | Date |
|------------|----------|-------|-------|------|
| 02-02 | 8 min | 2 | 2 | 2026-02-14 |
| 02-03 | 3 min | 2 | 4 | 2026-02-14 |
| 02-01 | 4 min | 2 | 5 | 2026-02-14 |
| 01-03 | 5 min | 2 | 5 | 2026-02-14 |
| 01-02 | 4 min | 2 | 8 | 2026-02-14 |

**Recent Trend:**
- Last 3 plans: 5.0 min average
- Trend: Stable velocity

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

### Pending Todos

None yet.

### Blockers/Concerns

- Research flag (RESOLVED): Phase 2 research confirmed all admin ops use Zoho Mail API — no separate Directory API needed
- Research flag: Phase 5 needs attachment upload testing -- sparse docs, Content-Type gotchas reported by community

## Session Continuity

Last session: 2026-02-14T19:30:00Z
Stopped at: Phase 2 complete (all 3 plans executed, verification passed)
Resume file: None
Next: Phase 2 complete. Ready for Phase 3 planning (Admin Domains & Audit)
