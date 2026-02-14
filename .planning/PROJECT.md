# zoh — Zoho CLI

## What This Is

A command-line interface for managing Zoho Admin and Mail operations, built in Go. Designed for power users who are frustrated with Zoho's scattered API docs and clunky web UI — the same problem Google Cloud Console has, except Zoho has no internal CLI tooling. Follows patterns from [gogcli](https://github.com/steipete/gogcli) for architecture, auth flows, and output formatting.

## Core Value

**Fast, reliable access to Zoho Admin and Mail operations from the terminal** — bypassing the slow, fragmented web UI entirely.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Full Zoho Admin operations: user management, domain/org settings, groups & roles, audit & security
- [ ] Full Zoho Mail operations: account settings, filters & routing, read/send email, mail admin (retention, spam, allowlists/blocklists)
- [ ] OAuth2 authentication with refresh token auto-refresh (currently proven via bash wrapper)
- [ ] Multi-region support (.com, .eu, .in, .com.au, .jp)
- [ ] Three output modes: JSON (scripting), plain (piping), rich (human terminal)
- [ ] XDG-compliant config + OS keyring for secrets (like gogcli)
- [ ] Dual command style: service-first hierarchy (`zoh admin users list`) AND action-first shortcuts (`zoh ls users`)
- [ ] Extensible architecture for community to add more Zoho services

### Out of Scope

- Zoho CRM — separate service, add via community contribution later
- Zoho Desk — separate service, add via community contribution later
- Zoho Projects — separate service, add via community contribution later
- All other Zoho apps — v1 focuses on Admin + Mail only
- Distribution packaging (Homebrew, etc.) — figure it out after core works

## Context

- **Current tooling:** Bash wrapper script (`zoho-mail`) that auto-refreshes OAuth tokens, caches in `/tmp/`, and passes through curl args. Credentials stored in `~/secrets/zoho-mail.env` with client_id, client_secret, refresh_token, API domain, accounts URL, org_id, account_id.
- **Zoho API landscape:** Scattered docs, multiple API domains per region, separate auth endpoints. EU instance (zoho.eu) is the primary development target but all regions should work.
- **Reference project:** gogcli — Go CLI using Kong framework, 99designs/keyring, muesli/termenv, with clean separation of auth/config/api/output concerns. ~92k LOC, production-grade patterns.
- **Open source intent:** Community-driven extensibility. Users can request features via issues, vote on priorities, or contribute PRs for additional Zoho services.

## Constraints

- **Tech stack**: Go, Kong CLI framework, OS keyring for secrets — matching gogcli patterns
- **API dependency**: Zoho API stability and documentation quality are external constraints
- **Auth model**: OAuth2 with refresh tokens — Zoho's self-client flow for initial token generation
- **Single org**: Designed for single-org use initially (user's current setup)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go (like gogcli) | Single binary, strong typing, proven CLI patterns in reference project | — Pending |
| Kong over Cobra | gogcli uses it successfully, struct-tag-based command definitions are clean | — Pending |
| XDG + OS keyring | Security best practice, cross-platform, same as gogcli | — Pending |
| Multi-region from start | Zoho has 5+ regional endpoints, designing this in is cheaper than retrofitting | — Pending |
| Command name: `zoh` | Short, fast to type, distinctive | — Pending |
| Dual command style | Service-first for discoverability, action-first for speed (gogcli pattern) | — Pending |
| Admin + Mail v1 only | Focused scope, extensible architecture allows community to add services | — Pending |

---
*Last updated: 2026-02-14 after initialization*
