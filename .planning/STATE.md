# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-14)

**Core value:** Fast, reliable access to Zoho Admin and Mail operations from the terminal
**Current focus:** Phase 2 -- Admin User & Group Operations

## Current Position

Phase: 2 of 6 (Admin User & Group Operations)
Plan: 1 of 5 in current phase
Status: Complete
Last activity: 2026-02-14 -- Completed 02-01 (Admin API client and user commands)

Progress: [██░░░░░░░░] 22.2% (4/18 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 4
- Average duration: 4.5 min
- Total execution time: 18 min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 3/3 | 14 min | 4.7 min |
| 02 | 1/5 | 4 min | 4.0 min |

**Recent Executions:**

| Phase-Plan | Duration | Tasks | Files | Date |
|------------|----------|-------|-------|------|
| 02-01 | 4 min | 2 | 5 | 2026-02-14 |
| 01-03 | 5 min | 2 | 5 | 2026-02-14 |
| 01-02 | 4 min | 2 | 8 | 2026-02-14 |
| 01-01 | 5 min | 2 | 10 | 2026-02-14 |

**Recent Trend:**
- Last 3 plans: 4.3 min average
- Trend: Consistent velocity

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

### Pending Todos

None yet.

### Blockers/Concerns

- Research flag: Phase 2 needs API endpoint audit (curl verification) at phase start -- some admin ops may require Zoho Directory API instead of Mail API
- Research flag: Phase 5 needs attachment upload testing -- sparse docs, Content-Type gotchas reported by community

## Session Continuity

Last session: 2026-02-14T18:25:36Z
Stopped at: Completed 02-01-PLAN.md (Admin API client and user commands)
Resume file: None
Next: Continue Phase 2 with plan 02-02 (User create/update/delete operations)
