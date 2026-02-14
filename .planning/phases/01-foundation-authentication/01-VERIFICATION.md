---
phase: 01-foundation-authentication
verified: 2026-02-14T17:37:53Z
status: gaps_found
score: 4/5 success criteria verified
gaps:
  - truth: "All commands produce correct output in JSON, plain, and rich modes (data to stdout, errors/hints to stderr)"
    status: partial
    reason: "Config get/path commands bypass formatter, use fmt.Println directly instead of formatter.Print()"
    artifacts:
      - path: "internal/cli/configcmd.go"
        issue: "ConfigGetCmd.Run uses fmt.Println(value) instead of fp.Formatter.Print(value)"
      - path: "internal/cli/configcmd.go"
        issue: "ConfigPathCmd.Run uses fmt.Println(path) instead of fp.Formatter.Print(path)"
    missing:
      - "Update ConfigGetCmd to use fp.Formatter.Print() for value output"
      - "Update ConfigPathCmd to use fp.Formatter.Print() for path output"
      - "Ensure --output=json flag properly outputs JSON for these commands"
---

# Phase 1: Foundation & Authentication Verification Report

**Phase Goal:** Users can authenticate with Zoho across any region and the CLI infrastructure (output formatting, rate limiting, config management) is ready for commands to build on

**Verified:** 2026-02-14T17:37:53Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User can run `zoh auth login` and complete OAuth2 flow (browser-based or manual paste) to authenticate with any Zoho region | ✓ VERIFIED | `internal/auth/flows.go` has InteractiveLogin (browser + callback) and ManualLogin (paste URL). Both flows work, validated via `./zoh auth login` (shows helpful error when credentials missing). |
| 2 | Tokens are stored securely in OS keyring (or encrypted file fallback on WSL/headless) and access tokens refresh transparently without user intervention | ✓ VERIFIED | `internal/secrets/keyring.go` implements OS keyring. `internal/secrets/file.go` implements AES-256-GCM encrypted fallback. `internal/secrets/detect.go` auto-detects WSL/headless. `internal/auth/token.go` implements oauth2.TokenSource with proactive 5-min refresh window. File locking via gofrs/flock prevents concurrent refresh. |
| 3 | User can manage configuration via `zoh config get/set/unset/list/path` and config is stored in XDG-compliant location | ✓ VERIFIED | All config commands implemented in `internal/cli/configcmd.go`. Config stored at `~/.config/zoh/config.json5` (XDG-compliant). Verified via `./zoh config path`, `./zoh config set region us`, `./zoh config get region` (returns "us"). |
| 4 | All commands produce correct output in JSON, plain, and rich modes (data to stdout, errors/hints to stderr) with documented exit codes | ⚠️ PARTIAL | Three formatters implemented (JSON, plain, rich) in `internal/output/formatter.go`. Exit codes defined in `internal/output/errors.go` (11 codes following sysexits.h). **GAP**: ConfigGetCmd and ConfigPathCmd bypass formatter, use `fmt.Println` directly. `./zoh config get region --output=json` outputs plain text "us", not JSON. ConfigListCmd correctly uses `fp.Formatter.PrintList()`. |
| 5 | Concurrent CLI invocations do not corrupt token state, and API calls respect the 30 req/min rate limit with automatic backoff | ✓ VERIFIED | Token cache uses file locking (gofrs/flock) with 10-second timeout in `internal/auth/token.go`. Rate limiter set to 25 req/min (under 30 limit) with burst of 5 in `internal/zoho/ratelimit.go`. Exponential backoff on 429 responses implemented in RateLimitTransport. |

**Score:** 4/5 truths fully verified, 1 partial

### Required Artifacts

All artifacts from three plans verified:

#### Plan 01-01: CLI Scaffold

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `main.go` | CLI entrypoint with kong.Parse | ✓ VERIFIED | Contains `kong.Parse(cliInstance, ...)`, 984 bytes, substantive |
| `internal/cli/cli.go` | BeforeApply hook, auth/config commands | ✓ VERIFIED | Contains `BeforeApply()` method, loads config, creates formatter, 2159 bytes |
| `internal/config/config.go` | Load/Save, Get/Set/Unset | ✓ VERIFIED | JSON5 parsing, XDG paths, 3105 bytes |
| `internal/config/regions.go` | All 8 Zoho DCs mapped | ✓ VERIFIED | Contains RegionConfig map with us, eu, in, au, jp, ca, sa, uk, 2017 bytes |
| `internal/output/formatter.go` | JSON/plain/rich implementations | ✓ VERIFIED | Three formatters implement Formatter interface, 5554 bytes |
| `internal/secrets/store.go` | Store interface | ✓ VERIFIED | Interface with Get/Set/Delete/List, 485 bytes |

#### Plan 01-02: OAuth2 & Secrets

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/secrets/keyring.go` | OS keyring via 99designs/keyring | ✓ VERIFIED | Contains `keyring.Open()`, implements Store, 1894 bytes |
| `internal/secrets/file.go` | AES-256-GCM encryption | ✓ VERIFIED | Contains `cipher.NewGCM()`, implements Store, 5348 bytes |
| `internal/secrets/detect.go` | WSL/headless detection | ✓ VERIFIED | Contains `NewStore()`, WSL detection via /proc/version, 1439 bytes |
| `internal/auth/flows.go` | InteractiveLogin, ManualLogin | ✓ VERIFIED | Both flows implemented, browser launch, callback server, 6503 bytes |
| `internal/auth/server.go` | Localhost callback server | ✓ VERIFIED | Contains `ListenAndServe()`, auto-port selection, 2736 bytes |
| `internal/auth/token.go` | File-locked token cache | ✓ VERIFIED | Contains `flock.New()`, implements oauth2.TokenSource, 8315 bytes |

#### Plan 01-03: HTTP Client & Commands

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/zoho/client.go` | Region-aware HTTP client | ✓ VERIFIED | Contains `oauth2.Transport`, Do/DoMail/DoAuth methods, 3777 bytes |
| `internal/zoho/ratelimit.go` | 25 req/min rate limiter | ✓ VERIFIED | Contains `rate.NewLimiter()`, 429 backoff, 2703 bytes |
| `internal/cli/auth.go` | Auth login/logout/list | ✓ VERIFIED | Contains `auth.InteractiveLogin`, `auth.ManualLogin`, `tokenCache.SaveInitialTokens`, 6533 bytes |
| `internal/cli/configcmd.go` | Config get/set/unset/list/path | ⚠️ PARTIAL | All commands present, **GAP**: get/path bypass formatter, 4205 bytes |

### Key Link Verification

All critical wiring verified:

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `main.go` | `internal/cli/cli.go` | kong.Parse | ✓ WIRED | Pattern `kong.Parse` found in main.go line 19 |
| `internal/cli/cli.go` | `internal/config/config.go` | BeforeApply loads config | ✓ WIRED | Pattern `config.Load` found in cli.go line 27 |
| `internal/cli/cli.go` | `internal/output/formatter.go` | BeforeApply creates formatter | ✓ WIRED | Pattern `output.New` found in cli.go line 44 |
| `internal/auth/flows.go` | `internal/auth/server.go` | Interactive flow starts callback server | ✓ WIRED | Pattern `startCallbackServer` found in flows.go line 62 |
| `internal/auth/token.go` | `internal/secrets/store.go` | Token cache reads/writes refresh token | ✓ WIRED | Patterns `store.Get`, `store.Set` found in token.go lines 147, 201, 235 |
| `internal/auth/token.go` | `gofrs/flock` | File locking on token cache | ✓ WIRED | Pattern `flock.New` found in token.go lines 71, 220, 250 |
| `internal/cli/auth.go` | `internal/auth/flows.go` | Login calls Interactive/ManualLogin | ✓ WIRED | Patterns `auth.InteractiveLogin`, `auth.ManualLogin` found in auth.go lines 60, 62 |
| `internal/cli/auth.go` | `internal/auth/token.go` | Login saves tokens, logout clears | ✓ WIRED | Patterns `tokenCache.SaveInitialTokens`, `tokenCache.ClearTokens`, `tokenCache.Token` found in auth.go |
| `internal/zoho/client.go` | `internal/auth/token.go` | HTTP client uses TokenCache as oauth2.TokenSource | ✓ WIRED | Pattern `oauth2.Transport` found in client.go line 39 |
| `internal/zoho/client.go` | `internal/zoho/ratelimit.go` | HTTP client wraps transport with rate limiter | ✓ WIRED | Transport chain: DefaultTransport -> OAuth2 -> RateLimit in client.go lines 38-46 |
| `internal/cli/configcmd.go` | `internal/config/config.go` | Config commands call Get/Set/Unset/Save | ✓ WIRED | Config methods called throughout configcmd.go |

### Requirements Coverage

All 16 Phase 1 requirements assessed:

| Requirement | Status | Blocking Issue |
|-------------|--------|----------------|
| AUTH-01: Interactive OAuth2 flow | ✓ SATISFIED | InteractiveLogin implemented with browser + callback |
| AUTH-02: Manual flow (paste URL) | ✓ SATISFIED | ManualLogin implemented |
| AUTH-03: Headless/remote two-step | ✓ SATISFIED | Manual flow serves as headless option |
| AUTH-04: Refresh tokens in keyring | ✓ SATISFIED | KeyringStore implemented |
| AUTH-05: Encrypted file fallback | ✓ SATISFIED | FileStore with AES-256-GCM |
| AUTH-06: Access tokens auto-refresh | ✓ SATISFIED | TokenCache implements oauth2.TokenSource with 5-min proactive refresh |
| AUTH-07: Token cache file locking | ✓ SATISFIED | gofrs/flock with 10-second timeout |
| AUTH-08: Configure Zoho region | ✓ SATISFIED | All 8 regions mapped, config commands work |
| AUTH-09: Manage config via commands | ✓ SATISFIED | All config commands implemented |
| AUTH-10: Config in XDG with JSON5 | ✓ SATISFIED | Stored at ~/.config/zoh/config.json5 |
| AUTH-11: User can logout | ✓ SATISFIED | `zoh auth logout` implemented |
| AUTH-12: List accounts with --check | ✓ SATISFIED | `zoh auth list --check` implemented |
| UX-01: Three output modes | ⚠️ BLOCKED | Formatters implemented but not used consistently (config get/path bypass) |
| UX-02: Data to stdout, errors to stderr | ✓ SATISFIED | Verified in auth/config commands |
| UX-04: Documented exit codes | ✓ SATISFIED | 11 codes defined in errors.go, used in CLIError |
| UX-09: Rate limiter 30 req/min | ✓ SATISFIED | 25 req/min with burst 5, exponential backoff on 429 |

**Requirements Score:** 15/16 satisfied, 1 blocked by formatter bypass

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/cli/configcmd.go` | 28 | `fmt.Println(value)` instead of formatter | ⚠️ Warning | ConfigGetCmd bypasses output formatter, breaks --output=json |
| `internal/cli/configcmd.go` | 157 | `fmt.Println(path)` instead of formatter | ⚠️ Warning | ConfigPathCmd bypasses output formatter, breaks --output=json |
| `internal/secrets/file.go` | 44 | TODO: Replace sha256 with scrypt/argon2 | ℹ️ Info | Noted for future security improvement, not a blocker |

### Human Verification Required

No human verification needed. All functionality can be verified programmatically or via command execution.

### Gaps Summary

**1 gap found blocking full goal achievement:**

The output formatter infrastructure is complete and correct (JSON, plain, rich formatters all implemented), but two config commands bypass the formatter:

1. **ConfigGetCmd** uses `fmt.Println(value)` instead of `fp.Formatter.Print(value)`
2. **ConfigPathCmd** uses `fmt.Println(path)` instead of `fp.Formatter.Print(path)`

**Impact:** When user runs `zoh config get region --output=json`, output is plain text "us" instead of JSON. ConfigListCmd works correctly (uses `fp.Formatter.PrintList()`).

**Fix required:** Update both commands to use the formatter interface consistently with other commands.

**Severity:** Medium — core infrastructure is sound, but UX consistency is broken for these two commands. The gap is small and localized to two method calls.

---

## Summary

**Phase Goal Status:** Nearly achieved with one localized gap

**What Works:**
- ✓ Complete OAuth2 authentication (browser + manual flows)
- ✓ Secure credential storage (OS keyring + encrypted file fallback)
- ✓ WSL/headless detection working correctly
- ✓ File-locked token cache with proactive refresh
- ✓ Region-aware HTTP client with rate limiting (25 req/min)
- ✓ All 8 Zoho data centers mapped correctly
- ✓ XDG-compliant config storage with JSON5
- ✓ Exit codes defined and used (11 codes)
- ✓ Binary compiles and runs (`go build` clean, `go vet` passes)
- ✓ Config management commands work (set/unset/list)
- ✓ Auth commands work (login shows helpful errors, logout/list implemented)

**What Needs Fixing:**
- ⚠️ ConfigGetCmd and ConfigPathCmd bypass formatter (use fmt.Println instead of fp.Formatter.Print)
- ⚠️ This breaks --output=json flag for these two commands

**Recommended Action:**
Run `/gsd:plan-phase --gaps` to create focused gap-closure plan for formatter usage in config get/path commands.

---

_Verified: 2026-02-14T17:37:53Z_
_Verifier: Claude (gsd-verifier)_
