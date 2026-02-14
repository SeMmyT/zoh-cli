# Project Research Summary

**Project:** zoh — Zoho Admin and Mail CLI
**Domain:** Go CLI wrapping Zoho REST APIs (Admin + Mail operations)
**Researched:** 2026-02-14
**Confidence:** HIGH

## Executive Summary

The research confirms that building a production-ready Zoho CLI requires navigating a complex multi-datacenter OAuth2 architecture, aggressive rate limits, and inconsistent API documentation. The recommended approach follows the proven Kong CLI framework pattern (as demonstrated by gogcli), with a layered internal-package design that cleanly separates API client logic from command implementations. Go 1.23+ with Kong for CLI structure, golang.org/x/oauth2 for auth, and 99designs/keyring for secure credential storage forms the core stack.

The most critical risks center on multi-region support and token management. Zoho operates 10 independent datacenters with region-specific auth servers and API endpoints. Hardcoding URLs or mishandling region-specific client secrets will break for non-US users. Additionally, Zoho's strict OAuth token limits (10 access tokens per 10 minutes, 30 API requests per minute) require proactive token caching with file-based locking and built-in rate limiting from day one. The 30 req/min rate limit is particularly aggressive for a CLI tool where users may script operations — this must shape the entire HTTP client architecture.

The recommended build path prioritizes auth and config infrastructure in Phase 1, implements simpler admin operations in Phase 2 to validate the full stack, then tackles mail operations in Phase 3. This ordering avoids the most severe pitfalls (multi-DC auth, token refresh races, keyring fallback) while proving extensibility early. The architecture supports adding new Zoho services (CRM, Books, etc.) with zero changes to existing code, making this a scalable foundation.

## Key Findings

### Recommended Stack

The stack centers on Go 1.23+ with Kong for CLI framework, oauth2 for authentication, and a carefully selected set of libraries that balance simplicity with production requirements. Full details in [STACK.md](.planning/research/STACK.md).

**Core technologies:**
- **Go 1.23+**: Single binary deployment, strong typing, proven for CLI tools (gogcli validates this choice)
- **Kong (latest)**: Struct-tag based CLI definitions, declarative command tree, used by gogcli — cleaner than Cobra's code-generation approach
- **golang.org/x/oauth2**: Standard OAuth2 library with custom token sources for keyring integration
- **99designs/keyring v1.2.2**: OS keychain abstraction (macOS Keychain, Linux Secret Service, Windows Credential Manager) with encrypted file fallback for headless environments

**Critical supporting libraries:**
- **muesli/termenv + fatih/color + rodaine/table**: Rich terminal output with capability detection
- **yosuke-furukawa/json5**: Config files with comments and trailing commas (human-editable)
- **stretchr/testify v1.11.1**: Essential for TDD workflow with readable assertions and mocks

**What NOT to use:**
- **schmorrison/Zoho**: Minimally maintained, doesn't cover Mail API, last commit June 2024
- **bubbletea/lipgloss**: Overkill for data-oriented CLI — designed for interactive TUIs
- **Viper**: Kong handles flags/env/defaults natively; Viper adds unnecessary complexity

### Expected Features

Feature research mapped the entire Zoho Mail API surface against web UI capabilities and identified notable gaps. Full details in [FEATURES.md](.planning/research/FEATURES.md).

**Must have (table stakes):**
- **User management**: List, add, remove, update role, reset password, enable/disable, manage aliases, toggle protocols (IMAP/POP/SMTP), toggle 2FA
- **Group management**: List, create, delete, add/remove members, update roles, view/moderate queue
- **Domain management**: List, add, verify, configure DKIM, set primary, manage catch-all
- **Org/security**: Get org details, view subscription/storage, manage spam allow/block lists, IP whitelist, audit/login/SMTP logs
- **Mail read/send**: List inbox, read messages, search, send (with attachments), reply, mark read/unread, move, flag, delete, archive
- **Mail settings**: Manage signatures, vacation reply, display name, reply-to, forwarding
- **Folders & labels**: Full CRUD for folders and labels, apply/remove from messages

**Should have (differentiators):**
- **Bulk user operations**: CSV/JSON import for mass onboarding/offboarding (GAM's killer feature)
- **Output format flexibility**: JSON, CSV, TSV, rich table for every command
- **Pipe-friendly design**: Every list command outputs IDs that pipe into action commands
- **Domain health check**: Single command validating DNS, SPF, DKIM, DMARC, MX for all domains
- **Onboarding/offboarding workflows**: Single command orchestrating multi-step user lifecycle operations
- **Multi-datacenter awareness**: Auto-detect correct API base URL per region (10 DCs)

**Defer (v2+):**
- **Tasks, Notes, Bookmarks**: APIs exist but out of scope for admin+mail focus
- **Calendar/contacts**: Separate product concern
- **Interactive TUI**: Nice-to-have, not core value proposition

**Anti-features (explicitly NOT building):**
- **Mail filter/rule management**: No API exists — would require fragile web scraping
- **eDiscovery/retention config**: No API exists, compliance-critical (use web UI)
- **SSO/SAML configuration**: No API exists, security-critical (use web UI)
- **HTML email rendering in terminal**: Poor UX — show plain text or open in browser

### Architecture Approach

The architecture follows a layered internal-package design with strict dependency flow and interface boundaries for testing. Full details in [ARCHITECTURE.md](.planning/research/ARCHITECTURE.md).

**Recommended structure:**
```
internal/
  cli/         Kong root struct, global flags, dependency injection hooks
  cmd/admin/   Admin command implementations
  cmd/mail/    Mail command implementations
  zoho/        API client layer (region-aware, auth transport)
    admin/     Admin API typed methods
    mail/      Mail API typed methods
  auth/        OAuth flow orchestration
  config/      XDG config, region, org settings
  secrets/     OS keyring abstraction (interface for testability)
  output/      JSON/plain/rich formatters
  ui/          Terminal colors, spinners, prompts
```

**Major components:**
1. **Region-aware base client** — Single `zoho.Client` that resolves correct regional base URL (10 DCs) and wraps http.Client with oauth2.Transport for automatic token refresh
2. **Persistent token source** — Custom oauth2.TokenSource reading refresh tokens from OS keyring, auto-refreshing, persisting back to keyring
3. **Service registration pattern** — Each Zoho service (Admin, Mail, future: CRM) is self-contained package with interface boundary — adding new services requires zero changes to existing code
4. **Output formatter interface** — Commands receive injected `output.Formatter`, call `Print(data)`, formatter handles JSON/TSV/rich table based on global flag
5. **Kong dependency injection** — `BeforeApply` hook initializes shared dependencies, `kong.Bind()` injects into command `Run()` methods

**Critical patterns:**
- **Interface at every boundary** for testing (Service interfaces, Keyring interface, Formatter interface)
- **No config threading** — Resolve config into concrete values during init, pass only what's needed
- **Region logic centralized** — Resolved once in `zoho.NewClient()`, never scattered across packages

### Critical Pitfalls

Research identified 16 pitfalls across critical/moderate/minor severity. Top 5 that require Phase 1 mitigation in [PITFALLS.md](.planning/research/PITFALLS.md):

1. **Multi-DC region URL management** — Zoho operates 10 independent datacenters with region-specific auth servers AND API base URLs. Hardcoding `zoho.com` breaks for EU/IN/AU/JP/CA/SA/UK/CN/AE users. Prevention: `RegionConfig` struct mapping region identifiers to all three URL types (accounts, API, mail), resolved once during client initialization. Must be in Phase 1 foundation.

2. **OAuth token refresh race conditions** — Zoho limits: 10 access tokens per 10 min per refresh token, 5 refresh tokens per min per user. Concurrent CLI invocations (pipelines, scripts) trigger simultaneous refresh attempts, exhausting token budget within seconds. Prevention: File-locked token cache so only one process refreshes at a time, proactive refresh when < 5 min remaining, exponential backoff on rate limit. Must be in auth layer foundation.

3. **Multi-DC client secret mismatch** — Each datacenter gets unique client secret by default unless "Use same credentials for all DCs" is explicitly selected during OAuth client setup. Prevention: Document self-client setup clearly, optionally store per-region secrets. Must be addressed in config schema from start.

4. **30 requests/minute rate limit with undisclosed lockout** — Zoho Mail API enforces 30 req/min globally across all endpoints. Exceeding triggers blocking period of undisclosed duration (not publicly documented for security). Scripts iterating over messages hit limit within first minute. Prevention: Built-in token bucket rate limiter in HTTP client (cap at ~25 req/min), progress feedback for long operations, batch endpoints where possible. Must be in HTTP client layer from Phase 1.

5. **Keyring fails on headless Linux/WSL** — `99designs/keyring` uses D-Bus Secret Service on Linux. Headless servers, WSL, containers have no D-Bus session bus or keyring daemon. Prevention: Ordered backend fallback (try Secret Service, fall back to encrypted file), WSL detection via `/proc/version`, `--keyring-backend` flag, clear error messages, support `ZOH_TOKEN` env var for CI/CD. Must be in secrets layer foundation.

## Implications for Roadmap

Based on combined research, the recommended phase structure prioritizes risk mitigation (auth complexity, rate limits, multi-DC) while validating architecture extensibility early.

### Phase 1: Foundation & Authentication
**Rationale:** Auth is the blocking dependency for all API calls. Multi-DC support, token management, and keyring fallback are the highest-risk architectural decisions. Getting these wrong forces rewrites later. Building this foundation first allows all subsequent phases to assume working auth.

**Delivers:**
- Multi-region config system (10 Zoho datacenters)
- OAuth self-client flow with browser-based auth
- Keyring-backed token storage with file fallback for WSL/headless
- Region-aware HTTP client with oauth2 transport
- Rate limiter (30 req/min) built into HTTP client
- File-locked token cache preventing refresh races
- Output formatter framework (JSON/plain/rich)

**Addresses pitfalls:**
- Pitfall 1: Multi-DC URLs (RegionConfig struct)
- Pitfall 2: Token refresh races (file-locked cache)
- Pitfall 3: Per-DC client secrets (config schema)
- Pitfall 4: Rate limits (token bucket in HTTP client)
- Pitfall 5: WSL/headless keyring (backend fallback)
- Pitfall 13: Signal handling (context plumbing)
- Pitfall 15: 200 OK with error body (unified response parser)

**Stack elements:**
- Kong CLI framework setup
- oauth2 + keyring integration
- XDG config (adrg/xdg)
- termenv for output capability detection

**Research flag:** Skip research-phase — auth patterns are well-documented in oauth2 docs and gogcli reference implementation.

---

### Phase 2: Core Admin Operations
**Rationale:** Admin operations have simpler API surface (mostly list/get/update) compared to mail (attachments, encoding). Implementing admin first validates the full request/response pipeline without complex edge cases. User/group management is the primary pain point (Zoho web UI is slow for repetitive tasks) — delivering this early proves CLI value.

**Delivers:**
- User management (list, add, remove, update, enable/disable, aliases, protocols, 2FA)
- Group management (list, create, delete, members, roles)
- Organization details (info, subscription, storage)
- Spam management (allow/block lists)
- IP whitelist management

**Implements:**
- `internal/zoho/admin/` service package with full interface
- `internal/cmd/admin/` command implementations
- Pagination abstraction (handle Zoho's inconsistent pagination)
- Output formatting integration (table/JSON/CSV)

**Addresses:**
- Features: Table stakes admin operations from FEATURES.md
- Pitfall 7: Inconsistent pagination (iterator pattern)
- Pitfall 11: API coverage gaps (conduct per-endpoint audit at phase start)

**Stack elements:**
- rodaine/table for rich output
- testify for TDD assertions

**Research flag:** Conduct API endpoint audit at phase start — verify each planned admin command has documented API endpoint before implementation. Some operations may require Zoho Directory API instead of Mail API.

---

### Phase 3: Domain Management & Logs
**Rationale:** Domain operations build on Phase 2's admin service patterns but add complexity (DNS verification, DKIM setup, multi-step workflows). Logs provide visibility into org activity. Both are admin-oriented but less frequently used than user/group ops.

**Delivers:**
- Domain management (list, add, verify, DKIM config, primary domain, catch-all)
- Audit/login/SMTP logs
- Domain health check (validate DNS, SPF, DKIM, DMARC, MX)

**Implements:**
- Extended admin service with domain/logs endpoints
- Health check aggregating multiple domain detail calls
- Log filtering and formatting

**Addresses:**
- Features: Domain admin from FEATURES.md
- Differentiator: Domain health check

**Research flag:** Skip research-phase — domain verification is standard DNS pattern, logs are read-only endpoints.

---

### Phase 4: Mail Operations (Read)
**Rationale:** Mail operations are the second major value driver (after admin). Splitting into read (Phase 4) and send (Phase 5) isolates complexity — reading has simpler error modes than sending with attachments/encoding. Many users need read-only mail access for monitoring/search.

**Delivers:**
- List inbox/folder messages
- Read message content (handle HTML vs plain text)
- Search emails
- Mark read/unread, flag, move, archive, delete
- Folder management (CRUD)
- Label management (CRUD, apply/remove)

**Implements:**
- `internal/zoho/mail/` service package
- `internal/cmd/mail/` commands
- Folder/label operations
- Message list pagination

**Addresses:**
- Features: Mail read operations (table stakes)
- Pitfall 10: Email encoding (UTF-8 normalization, charset detection)

**Stack elements:**
- Consider jaytaylor/html2text for HTML email rendering

**Research flag:** Skip research-phase — message list/read are standard REST operations.

---

### Phase 5: Mail Operations (Send & Settings)
**Rationale:** Sending email has the most complex edge cases (attachments, encoding, HTML escaping for JSON). Settings management (signatures, vacation, forwarding) has multi-step verification flows. Tackling these after read operations are stable reduces risk.

**Delivers:**
- Send email (compose, reply, CC/BCC, subject, body)
- Send with attachments (upload flow)
- Download attachments
- Signature management (CRUD)
- Vacation reply
- Display name, reply-to, forwarding config

**Implements:**
- Extended mail service with send/settings endpoints
- Attachment upload (Content-Type: application/octet-stream)
- JSON escaping for HTML content
- Multi-step forwarding verification flow

**Addresses:**
- Features: Mail send + settings (table stakes)
- Pitfall 6: Attachment Content-Type (manual endpoint verification)
- Pitfall 10: Email encoding (UTF-8 for send, proper HTML escaping)

**Research flag:** Phase needs deeper research for attachment upload — API docs are sparse, community reports indicate Content-Type gotchas. Test manually with curl before implementing.

---

### Phase 6: Power User Features
**Rationale:** After core admin + mail operations are stable, layer on differentiators that make power users love the tool. These features compound value of existing commands (bulk ops scale user management, reports aggregate data, workflows orchestrate multiple commands).

**Delivers:**
- Bulk user operations (CSV import for onboarding/offboarding)
- User audit report (aggregate user details, groups, login history, storage)
- Storage reports (org-wide usage with top consumers)
- Onboarding/offboarding workflows
- Policy management (create, assign, templates)
- Shell completions (Bash/Zsh/Fish)

**Implements:**
- CSV/JSON parsing for bulk ops
- Multi-call aggregation with rate limit awareness
- Policy templates (restrictive/moderate/open presets)
- Kong completion generation

**Addresses:**
- Features: Differentiators from FEATURES.md
- Pitfall 4: Rate limits (aggregation commands need progress bars)

**Research flag:** Skip research-phase — these compose existing endpoints, no new API surface.

---

### Phase Ordering Rationale

**Dependency-driven:**
- Auth (Phase 1) blocks all API calls → must be first
- Admin operations (Phase 2) validate full stack with simpler API surface → prove architecture before complex mail ops
- Mail read (Phase 4) before send (Phase 5) isolates attachment/encoding complexity
- Power features (Phase 6) depend on stable core operations

**Risk mitigation:**
- Multi-DC auth, token races, rate limits (highest-risk pitfalls) addressed in Phase 1 foundation
- API coverage gaps audited before Phase 2 implementation
- Attachment upload gotchas researched/tested at Phase 5 start
- Extensibility validated by Phase 2 — adding mail service in Phase 4 should require zero architecture changes

**Value delivery:**
- Phase 2 delivers immediate value (user/group admin — primary pain point)
- Phases 2-3 complete full admin story before mail
- Phases 4-5 complete mail story
- Phase 6 adds differentiators after core value is proven

### Research Flags

**Phases needing deeper research during planning:**
- **Phase 2 (Admin Ops):** Conduct per-endpoint API audit with curl to verify each planned command has working API endpoint. Some operations may require Zoho Directory API or Admin Console API instead of Mail API. Document deviations from docs in "docs vs reality" reference.
- **Phase 5 (Mail Send):** Attachment upload has sparse docs and community-reported Content-Type gotchas. Manual verification with curl required before implementation. Email encoding edge cases need test fixtures with Japanese/Chinese/Arabic/emoji content.

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Auth):** Well-documented OAuth2 patterns, gogcli provides reference implementation
- **Phase 3 (Domains/Logs):** Standard DNS patterns, read-only log endpoints
- **Phase 4 (Mail Read):** Standard REST list/get operations
- **Phase 6 (Power Features):** Composes existing endpoints, no new API surface

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Go 1.23+ with Kong is proven by gogcli. All libraries verified from official docs/releases. Keyring fallback pattern documented in 99designs/keyring. |
| Features | HIGH | Complete API surface mapped from official Zoho Mail API index. API coverage gaps clearly identified (mail rules, eDiscovery, SSO — no APIs exist). Rate limits and OAuth scopes verified from official docs. |
| Architecture | HIGH | Layered internal-package design follows official Go module layout guidance. All patterns verified against gogcli reference implementation and Kong framework docs. Service registration extensibility validated. |
| Pitfalls | HIGH | Critical pitfalls (multi-DC, token limits, rate limits) verified from official Zoho OAuth/API docs. Keyring issues verified from 99designs/keyring GitHub issues. Attachment Content-Type gotcha verified from community blog post and API docs. |

**Overall confidence:** HIGH

Research drew primarily from official Zoho documentation (OAuth, Mail API, multi-DC, rate limits), official library documentation (Kong, oauth2, keyring), and the gogcli reference implementation (real-world Go CLI wrapping Google APIs in same domain). Community sources used only to identify pitfalls, then verified against official docs where possible.

### Gaps to Address

**API endpoint verification (Phase 2 blocker):**
- Gap: Zoho Mail API index lists categories (Organization, Users, Groups, Domains) but per-endpoint documentation has varying detail. Some operations may only be available via Zoho Directory API or Admin Console API.
- Resolution: Conduct manual curl audit for each planned admin command at Phase 2 start. Document which API service (Mail vs Directory vs Admin Console) each operation requires. Flag any operations with no working endpoint for web UI redirect.

**Attachment upload specifics (Phase 5 risk):**
- Gap: Zoho Mail API docs show Java example for attachment upload but do not explicitly state Content-Type requirement. Community blog post indicates `application/octet-stream` required, but docs don't confirm.
- Resolution: Manual endpoint testing with curl at Phase 5 start. Create golden test fixtures from successful requests. Document Content-Type + request format deviations from typical REST patterns.

**Pagination inconsistency (Phase 2+ risk):**
- Gap: Zoho Mail API documentation does not specify pagination parameters for list endpoints. Pagination behavior must be discovered experimentally. Other Zoho services use different pagination schemes (page-based vs offset-based vs index-count).
- Resolution: Test each list endpoint with >1 page of results to discover pagination parameters. Build pagination abstraction in Phase 2 that accepts strategy parameter so each endpoint can declare its pagination style.

**OAuth scope completeness (Phase 1 risk):**
- Gap: Complete list of Zoho Mail scopes not documented in single location. Scope format verified (`ServiceName.scopeName.OperationType`) but full enumeration for Mail API unclear.
- Resolution: Build scope list incrementally during implementation. When a command hits `OAUTH_SCOPE_MISMATCH`, document required scope. Provide scope profiles (mail-read-only, mail-full, admin-read-only, admin-full) in auth setup wizard.

**Rate limit specifics (Phase 1 assumption):**
- Gap: 30 req/min rate limit verified from official Zoho Mail rates-and-limits page. Lockout duration explicitly not disclosed. Unclear if limit is per-token, per-user, or per-org.
- Resolution: Implement conservative token bucket rate limiter (25 req/min) with configurable override. Test limit behavior during Phase 1 integration testing. Add telemetry to track rate limit hits and adjust if needed.

## Sources

### Primary (HIGH confidence)
- [Zoho Mail API Index](https://www.zoho.com/mail/help/api/) — Complete endpoint inventory across Organization, Users, Groups, Domains, Messages, Accounts, Folders, Labels, Logs APIs
- [Zoho OAuth Multi-DC Documentation](https://www.zoho.com/accounts/protocol/oauth/multi-dc.html) — Region-specific auth servers, API domains, client secret handling
- [Zoho OAuth Self-Client Flow](https://www.zoho.com/accounts/protocol/oauth/self-client/authorization-code-flow.html) — Grant token generation, 3-minute expiry, token exchange
- [Zoho Mail API Getting Started](https://www.zoho.com/mail/help/api/getting-started-with-api.html) — Base URLs, auth headers, response envelope structure
- [Zoho Mail Rates and Limits](https://www.zoho.com/mail/help/adminconsole/rates-and-limits.html) — 30 req/min limit, undisclosed lockout period
- [Zoho CRM API Limits](https://www.zoho.com/crm/developer/docs/api/v8/api-limits.html) — OAuth token limits: 10 access tokens per 10 min, 5 refresh tokens per min, 20 refresh tokens per org
- [Kong CLI Framework](https://github.com/alecthomas/kong) — Struct-tag command definitions, dependency injection via Bind, hooks
- [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) — TokenSource interface, Transport pattern, refresh logic
- [99designs/keyring](https://github.com/99designs/keyring) — v1.2.2 release, backend options (Secret Service, KWallet, Keychain, Pass, file)
- [Go Module Layout](https://go.dev/doc/modules/layout) — Official guidance on internal/ package usage

### Secondary (MEDIUM confidence)
- [gogcli Reference Implementation](https://github.com/steipete/gogcli) — Real-world Go CLI wrapping Google APIs, Kong-based, output formatting, keyring integration
- [Daniel Michaels Kong Patterns](https://danielms.site/zet/2024/how-i-write-golang-cli-tools-today-using-kong/) — Practical Kong usage, verified against official docs
- [99designs/keyring Issue #106](https://github.com/99designs/keyring/issues/106) — Headless Linux / WSL Secret Service failures, backend fallback patterns
- [99designs/keyring Issue #23](https://github.com/99designs/keyring/issues/23) — CGo cross-compilation issues, macOS Keychain dependencies
- [Zoho Mail Attachment Upload Gotchas](https://pebblesrox.wordpress.com/2021/03/28/zoho-mail-api-how-to-upload-an-attachment/) — Content-Type: application/octet-stream requirement, deviates from multipart/form-data

### Tertiary (LOW confidence)
- [schmorrison/Zoho Go Library](https://github.com/schmorrison/Zoho) — Minimally maintained (last commit June 2024), Mail API not implemented. Useful as reference for request patterns but not recommended as dependency.
- Community reports on Zoho OAuth refresh issues, pagination inconsistencies, 200 OK with error body — verify during implementation.

---
*Research completed: 2026-02-14*
*Ready for roadmap: yes*
