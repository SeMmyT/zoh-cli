# zoh -- Zoho CLI

## What This Is

A command-line interface for managing Zoho Admin and Mail operations, built in Go. Designed for power users who are frustrated with Zoho's scattered API docs and clunky web UI. Covers the full admin lifecycle (users, groups, domains, audit) and mail operations (read, search, send, settings, admin controls) with scripting-friendly flags, desire-path shortcuts, and shell completion. Follows patterns from [gogcli](https://github.com/steipete/gogcli) for architecture, auth flows, and output formatting.

## Core Value

**Fast, reliable access to Zoho Admin and Mail operations from the terminal** -- bypassing the slow, fragmented web UI entirely.

## Requirements

### Validated

- ✓ Full Zoho Admin operations: user management, domain/org settings, groups & roles, audit & security -- v1.0
- ✓ Full Zoho Mail operations: account settings, read/send email, mail admin (retention, spam, allowlists/blocklists) -- v1.0
- ✓ OAuth2 authentication with refresh token auto-refresh -- v1.0
- ✓ Multi-region support (.com, .eu, .in, .com.au, .jp, .com.cn, .sa, .uk) -- v1.0
- ✓ Three output modes: JSON (scripting), plain (piping), rich (human terminal) -- v1.0
- ✓ XDG-compliant config + OS keyring for secrets (encrypted file fallback for WSL/headless) -- v1.0
- ✓ Dual command style: service-first hierarchy (`zoh admin users list`) AND action-first shortcuts (`zoh ls users`) -- v1.0
- ✓ Power user flags: --results-only, --no-input, --force, --dry-run -- v1.0
- ✓ Machine-readable schema introspection (`zoh schema`) -- v1.0
- ✓ Shell completion (bash, zsh, fish) -- v1.0

### Active

(No active requirements -- v1.0 complete. Define v2 scope with `/gsd:new-milestone`)

### Out of Scope

- Zoho CRM -- separate service, add via community contribution later
- Zoho Desk -- separate service, add via community contribution later
- Zoho Projects -- separate service, add via community contribution later
- All other Zoho apps -- v1 focuses on Admin + Mail only
- Distribution packaging (Homebrew, etc.) -- figure it out after core works
- Email filter/rule management -- NO API exists, would need Zoho to add
- Web UI / TUI dashboard -- this is a CLI, not an interactive app
- Email rendering in terminal -- show raw text/HTML, don't try to render rich email

## Context

Shipped v1.0 with 8,269 LOC Go across 99 files in a single day.
Tech stack: Go 1.24, Kong CLI framework, 99designs/keyring, muesli/termenv, gofrs/flock, kongplete.
Architecture: layered internal packages (cli, zoho, output, auth, config) with AdminClient/MailClient/MailAdminClient API layers.
64 requirements implemented across 6 phases, 16 plans.

## Constraints

- **Tech stack**: Go, Kong CLI framework, OS keyring for secrets -- matching gogcli patterns
- **API dependency**: Zoho API stability and documentation quality are external constraints
- **Auth model**: OAuth2 with refresh tokens -- Zoho's self-client flow for initial token generation
- **Single org**: Designed for single-org use initially (user's current setup)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go (like gogcli) | Single binary, strong typing, proven CLI patterns in reference project | ✓ Good |
| Kong over Cobra | gogcli uses it successfully, struct-tag-based command definitions are clean | ✓ Good |
| XDG + OS keyring | Security best practice, cross-platform, same as gogcli | ✓ Good |
| Multi-region from start | Zoho has 8 regional endpoints, designing this in is cheaper than retrofitting | ✓ Good |
| Command name: `zoh` | Short, fast to type, distinctive | ✓ Good |
| Dual command style | Service-first for discoverability, action-first for speed (gogcli pattern) | ✓ Good |
| Admin + Mail v1 only | Focused scope, extensible architecture allows community to add services | ✓ Good |
| FormatterProvider wrapper | Kong can't bind interfaces directly, wrapper pattern works | ✓ Good |
| AES-256-GCM file fallback | WSL/headless environments lack keyring, encrypted file is secure fallback | ✓ Good |
| Cached org/account IDs | AdminClient caches zoid, MailClient caches accountId -- avoids redundant API calls | ✓ Good |
| Generic PageIterator | Go 1.24 generics enable reusable pagination for all list endpoints | ✓ Good |
| Two-step attachment upload | Upload file first (octet-stream), get reference, include in send request | ✓ Good |
| JSON envelope by default | Lists wrapped in `{"data": [...], "count": N}`, --results-only strips it | ✓ Good |
| Shortcuts hidden from help | Keeps main help clean, discoverable via schema/docs | ✓ Good |
| kongplete for completion | Maintained Kong completion library, handles all 3 shells | ✓ Good |

---
*Last updated: 2026-02-14 after v1.0 milestone*
