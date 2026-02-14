# Domain Pitfalls

**Domain:** Go CLI for Zoho Admin and Mail API integration
**Researched:** 2026-02-14

---

## Critical Pitfalls

Mistakes that cause rewrites or major issues.

---

### Pitfall 1: Hardcoding Region URLs Instead of Dynamic DC Resolution

**What goes wrong:** Developers hardcode `zoho.com` or a single region's URLs throughout the codebase, then discover that Zoho operates 8+ data centers (US, EU, IN, AU, JP, CA, SA, UK) with completely separate authentication servers AND API base URLs. Each region pair is different:

| Region | Auth Server | API Domain |
|--------|------------|------------|
| US | `accounts.zoho.com` | `www.zohoapis.com` |
| EU | `accounts.zoho.eu` | `www.zohoapis.eu` |
| IN | `accounts.zoho.in` | `www.zohoapis.in` |
| AU | `accounts.zoho.com.au` | `www.zohoapis.com.au` |
| JP | `accounts.zoho.jp` | `www.zohoapis.jp` |
| CA | `accounts.zohocloud.ca` | `www.zohoapis.ca` |
| SA | `accounts.zoho.sa` | `www.zohoapis.sa` |
| UK | `accounts.zoho.uk` | `www.zohoapis.uk` |

Additionally, the Zoho Mail API uses yet another URL pattern: `mail.zoho.{tld}` (not `www.zohoapis.{tld}`).

**Why it happens:** Most Zoho API examples show `.com` URLs. The EU/IN/AU differences are documented separately and easy to miss. The Mail API uses a different base URL pattern than other Zoho services.

**Consequences:** Users on non-US regions get authentication failures (`invalid_client` errors), API 404s, or silently hit wrong-region endpoints. Retrofitting multi-region support after the fact requires touching every HTTP call.

**Prevention:** Design a `RegionConfig` struct from day one that maps a region identifier to all three URL types (accounts server, API domain, mail API domain). Every HTTP client must receive its base URL from this config, never from a hardcoded string. Use the `api_domain` returned in the OAuth token response to validate.

**Detection:** Any hardcoded `zoho.com` string in non-test code is a red flag. Tests should exercise at least two regions.

**Confidence:** HIGH -- verified from official Zoho multi-DC documentation at https://www.zoho.com/accounts/protocol/oauth/multi-dc.html

**Phase:** Must be addressed in Phase 1 (foundation/auth).

---

### Pitfall 2: OAuth2 Token Refresh Race Conditions and Limit Exhaustion

**What goes wrong:** Zoho enforces strict token generation limits that are easy to hit:

- Maximum **10 access tokens per 10 minutes** per refresh token
- Maximum **5 refresh tokens per minute** per user
- Maximum **20 refresh tokens** per user per organization
- Maximum **30 active access tokens** per refresh token at any time

If two CLI invocations run concurrently (e.g., `zoh mail list` piped to `zoh mail read`), both detect an expired token simultaneously, both attempt refresh, and you burn through your token budget. Worse: if the CLI is used in scripts with `xargs -P` or `parallel`, you can exhaust 10 access tokens in seconds and get locked out for the remainder of the 10-minute window.

**Why it happens:** CLI tools are invoked per-command. Unlike a long-running server that refreshes once, a CLI may spawn dozens of processes in a pipeline. Naive "check expiry, refresh if needed" logic without coordination leads to thundering herd.

**Consequences:** API calls fail with `invalid_code` or throttling errors. Users in scripting workflows get intermittent failures. The 10-minute lockout window is not documented with a clear error message, so users blame the CLI.

**Prevention:**
1. **File-based token cache with locking:** Store the current access token + expiry in a file. Use `flock` (or Go's `os.File` lock) so only one process refreshes at a time. Others wait and read the refreshed token.
2. **Proactive refresh:** Refresh when the token has < 5 minutes remaining (not when it is already expired), reducing the window where multiple processes see "expired."
3. **Exponential backoff on refresh failure:** If refresh returns an error indicating rate limiting, back off rather than retry immediately.
4. **Single-token design:** Cache the access token to disk/memory so concurrent invocations share one token rather than each generating their own.

**Detection:** Test with `for i in $(seq 20); do zoh mail list & done` and verify only one refresh occurs.

**Confidence:** HIGH -- token limits verified from Zoho CRM API docs (https://www.zoho.com/crm/developer/docs/api/v8/access-refresh.html) and Zoho Recruit docs. Mail-specific limits may differ but the pattern is consistent across Zoho services.

**Phase:** Must be addressed in Phase 1 (auth layer). Get this wrong and every subsequent feature is unreliable.

---

### Pitfall 3: Multi-DC Client Secret Mismatch

**What goes wrong:** When Zoho's multi-DC support is enabled for an OAuth client, **each data center gets a unique client secret by default**. The Client ID is shared across all DCs, but the Client Secret is not. If the CLI stores a single client_secret and the user is on EU while the secret is for US, all token operations fail with `invalid_client`.

**Why it happens:** Zoho's default behavior when enabling multi-DC is to generate separate secrets per DC. The option to "Use the same OAuth credentials for all data centers" exists but is opt-in and not the default. Developers who test only on one region never encounter this.

**Consequences:** Users on non-primary regions cannot authenticate. The error message (`invalid_client`) does not indicate the real cause is a DC-specific secret mismatch.

**Prevention:**
1. During OAuth client registration, explicitly select "Use the same OAuth credentials for all data centers" to simplify deployment.
2. If per-DC secrets are needed for security, store the client secret per-region in the config and select the correct one based on the user's configured region.
3. Document the self-client setup flow clearly, including which DC to generate the grant token from.
4. The CLI setup wizard should ask for region first, then validate the client secret against that region's auth server.

**Detection:** Integration test that attempts token refresh against at least two different region auth servers.

**Confidence:** HIGH -- verified from https://www.zoho.com/accounts/protocol/oauth/multi-dc.html

**Phase:** Phase 1 (auth/config). The config schema must account for this from the start.

---

### Pitfall 4: Zoho Mail API's 30 Requests/Minute Rate Limit with Undisclosed Lockout

**What goes wrong:** The Zoho Mail API enforces a hard limit of **30 API requests per minute**. Exceeding this triggers a blocking period of undisclosed duration. The documentation explicitly states: "the locking period is not publicly disclosed for security reasons." This means:

- You cannot predict or communicate the cooldown to users
- There is no `Retry-After` header documented
- The limit applies globally across all Mail API endpoints, not per-endpoint

For a CLI tool where a user might run `zoh mail list` (1 call) then `zoh mail read <id>` for 30 messages in a script, they hit the limit within the first minute.

**Why it happens:** The 30/min limit is buried in the rates-and-limits page, not in the API endpoint documentation. Most developers discover it in production.

**Consequences:** Scripts that iterate over mailboxes or messages fail silently or with opaque errors partway through. Users cannot predict when they will be unblocked.

**Prevention:**
1. **Built-in rate limiter:** Implement a token bucket or sliding window rate limiter in the HTTP client layer, capped at ~25 requests/minute (leaving headroom).
2. **`--rate-limit` flag:** Let users override the default if Zoho changes limits or for enterprise accounts with higher limits.
3. **Batch operations where possible:** Use bulk endpoints (thread operations, batch delete) instead of per-message calls.
4. **Progress feedback:** For operations that will take multiple minutes due to rate limiting, show a progress bar with ETA.
5. **Retry with exponential backoff:** When a 429-like response is detected (Zoho may return various status codes), implement backoff.

**Detection:** Warning signs include any loop that calls a Zoho API endpoint per-item without rate limiting. Code review should flag `for range items { client.Get(...) }` patterns.

**Confidence:** HIGH -- rate limit verified from https://www.zoho.com/mail/help/adminconsole/rates-and-limits.html

**Phase:** Must be in the HTTP client layer from Phase 1. Every API call flows through the rate limiter.

---

### Pitfall 5: Keyring Fails on Headless Linux / WSL Without Secret Service

**What goes wrong:** The `99designs/keyring` library uses the D-Bus Secret Service API on Linux (backed by GNOME Keyring or KDE Wallet). On headless servers, WSL, containers, and minimal Linux installations, there is no D-Bus session bus or keyring daemon running. The library throws:

```
No such interface 'org.freedesktop.DBus.Properties' on object at path /
```

WSL specifically has no "default" collection defined, so the keystore does not work out of the box. The automatic keyring unlock steps from GNOME also do not work on WSL because it bypasses TTY login.

**Why it happens:** Linux desktop environments start these services automatically. Server environments, CI/CD, Docker containers, and WSL do not. The `99designs/keyring` library does support a file-based backend as a fallback, but it must be explicitly configured.

**Consequences:** The CLI crashes on first use for any Linux user not running a full desktop environment. Since the target audience includes server admins and automation users, this likely affects a majority of users.

**Prevention:**
1. **Ordered backend fallback:** Try Secret Service first, fall back to file-based encrypted storage if Secret Service is unavailable. Do this detection at runtime, not compile time.
2. **`--keyring-backend` flag:** Let users explicitly choose: `secret-service`, `kwallet`, `file`, `pass` (password-store).
3. **Graceful error message:** If no backend works, print a clear message explaining the issue and how to configure an alternative, not a raw D-Bus error.
4. **WSL detection:** Check for WSL via `/proc/version` containing "Microsoft" or "WSL" and default to file-based backend.
5. **CI/CD mode:** Support `ZOH_TOKEN` environment variable for automation contexts where no keyring is available.

**Detection:** Test the CLI in a fresh Docker container with no desktop services. If it panics or shows a D-Bus error, the fallback is broken.

**Confidence:** HIGH -- verified from https://github.com/99designs/keyring/issues/106 and multiple community reports.

**Phase:** Phase 1 (config/auth). The keyring backend selection must be in the initial config layer.

---

## Moderate Pitfalls

---

### Pitfall 6: Zoho Mail API Documentation Lies About Content-Type for Attachments

**What goes wrong:** The Zoho Mail API documentation for attachment upload states "The binary file is mandatory and has to be sent in the Request Body" but provides only a Java example using Apache HttpClient. The actual required Content-Type header is `application/octet-stream`, not `multipart/form-data` as most developers assume. Sending `multipart/form-data` returns HTTP 415 (Unsupported Media Type). The documentation does not explicitly state this requirement.

**Why it happens:** Zoho's API documentation is generated per-service with inconsistent quality. The Mail API docs are notably sparser than CRM docs. The attachment endpoint breaks conventions that most REST APIs follow.

**Prevention:**
1. **Manual testing against each endpoint before implementing:** Do not trust the docs for request format. Use `curl` to verify the actual required headers.
2. **Record and replay tests:** Capture successful `curl` requests and encode them as golden test fixtures.
3. **Document deviations:** Maintain a "docs vs reality" reference in the codebase for endpoints where behavior differs from documentation.

**Confidence:** MEDIUM -- verified from developer blog post (https://pebblesrox.wordpress.com/2021/03/28/zoho-mail-api-how-to-upload-an-attachment/) and Zoho's own API docs showing `application/octet-stream` in the Java example.

**Phase:** Phase 3+ (mail operations). Flag each mail endpoint for manual verification during implementation.

---

### Pitfall 7: Inconsistent Pagination Across Zoho API Endpoints

**What goes wrong:** Different Zoho services use completely different pagination schemes:

| Service | Pagination Style | Parameters |
|---------|-----------------|------------|
| Zoho CRM | Concurrency-based with offset | `page`, `per_page` |
| Zoho Books/Inventory | Page-number based | `page`, `per_page`, `page_context` node |
| Zoho Assist | Index-count based | `index`, `count` |
| Zoho Mail | Undocumented | Not specified in API docs |

Even within Zoho Mail, the API documentation does not specify pagination parameters for list endpoints. This means you must discover pagination behavior through experimentation.

**Why it happens:** Zoho's services were built by different teams at different times. There is no unified API design standard.

**Prevention:**
1. **Abstract pagination into an iterator pattern:** Create a generic `PageIterator` that accepts a pagination strategy (offset-based, cursor-based, page-number-based) so each endpoint can declare its pagination style.
2. **Default to conservative page sizes:** Start with small page sizes (e.g., 20) and let users configure via `--limit` flag.
3. **Test pagination with real data:** Pagination bugs often surface only with > 1 page of results. Integration tests need enough test data to trigger pagination.

**Confidence:** MEDIUM -- pagination inconsistency verified across Zoho services from official docs. Zoho Mail's specific pagination is LOW confidence (undocumented).

**Phase:** Phase 2 (API client layer). The pagination abstraction must be in place before implementing list commands.

---

### Pitfall 8: Self-Client Grant Token Expires in 3 Minutes

**What goes wrong:** The Zoho self-client OAuth flow generates a grant token (authorization code) that expires in **3 minutes** (configurable, but very short). The user must:

1. Go to Zoho API Console
2. Select Self-Client
3. Enter scopes, generate the code
4. Copy the code
5. Paste it into the CLI setup
6. The CLI exchanges it for tokens

If the user takes more than 3 minutes (common when they are reading docs, selecting scopes, or dealing with an unfamiliar UI), the grant token expires and they must start over. The error message from Zoho is `invalid_code`, which does not explain that the code simply expired.

**Why it happens:** Zoho designed the self-client flow for developers who know exactly what scopes to request. First-time CLI users fumble through the process.

**Consequences:** Frustrating first-run experience. Users may attempt the flow 3-4 times before succeeding, each time generating new refresh tokens (max 20 per org).

**Prevention:**
1. **Guided setup wizard:** The CLI `zoh auth setup` command should print the exact scopes needed, the exact URL to visit, and step-by-step instructions before prompting for the code. Minimize the time between code generation and input.
2. **Pre-compute scope string:** Print the comma-separated scope string so users can copy-paste it directly into the Zoho console.
3. **Clear error message:** If token exchange fails with `invalid_code`, tell the user "The authorization code has expired (they last 3 minutes). Please generate a new one."
4. **Consider device flow:** Zoho supports the OAuth device flow for non-browser apps, which may be a better UX (user visits URL, enters code displayed by CLI).

**Confidence:** HIGH -- 3-minute expiry verified from https://www.zoho.com/accounts/protocol/oauth/self-client/authorization-code-flow.html

**Phase:** Phase 1 (auth setup). The setup wizard must guide users through this carefully.

---

### Pitfall 9: Scope Explosion Across Admin and Mail APIs

**What goes wrong:** Zoho uses fine-grained OAuth scopes with the format `ServiceName.scopeName.OperationType` (e.g., `ZohoMail.messages.READ`, `ZohoMail.accounts.UPDATE`). The CLI needs scopes across many API categories:

- `ZohoMail.messages.ALL` -- email CRUD
- `ZohoMail.accounts.ALL` -- account settings
- `ZohoMail.folders.ALL` -- folder management
- `organization.domains.ALL` -- domain management
- `organization.groups.ALL` -- group management
- `organization.accounts.ALL` -- user management (admin)
- `organization.subscriptions.READ` -- subscription info
- `organization.spam.ALL` -- spam settings

Requesting all scopes at once creates a wall of permissions during authorization. Requesting too few means users hit `OAUTH_SCOPE_MISMATCH` errors when they try a command they do not have scope for.

**Why it happens:** Zoho's scope model is per-module, per-operation. A CLI that covers both admin and mail needs dozens of scopes. The complete scope list for Zoho Mail is not comprehensively documented in one place.

**Consequences:** Users either over-authorize (security concern) or under-authorize (broken commands). Scope mismatch errors from Zoho do not tell you which scope is missing.

**Prevention:**
1. **Scope profiles:** Define scope bundles: "mail-read-only", "mail-full", "admin-read-only", "admin-full", "everything". Let users choose during setup.
2. **Incremental authorization:** Zoho supports scope enhancement (https://www.zoho.com/accounts/protocol/oauth/incremental-auth/scope-enhance-request.html). If a command needs a scope the user has not granted, prompt to re-authorize with the additional scope.
3. **Per-command scope documentation:** Each CLI command should document which scope it requires. When a scope mismatch error occurs, the CLI should suggest which scope to add.
4. **Scope validation on setup:** After obtaining tokens, call a lightweight endpoint to verify which scopes were actually granted.

**Confidence:** MEDIUM -- scope format verified from official docs. The complete list of Zoho Mail scopes is not fully documented in one place (LOW confidence on completeness).

**Phase:** Phase 1 (auth). Scope management design affects every subsequent feature.

---

### Pitfall 10: Email Encoding and Character Set Handling

**What goes wrong:** The Zoho Mail API supports 13 encoding formats for email content (UTF-8, ISO-8859-1, Shift_JIS, Big5, GB2312, etc.). When sending email, the `encoding` parameter must match the actual content encoding. When reading email, the API may return content in any of these encodings, and the response does not always clearly indicate which encoding was used for the body.

Additionally, HTML email content with special characters can cause JSON parse errors when sent via the API. The Zoho community forums document cases where HTML content breaks the JSON request body.

**Why it happens:** Email is a 40-year-old protocol with layers of encoding complexity. Zoho must handle all legacy encodings. The API's JSON wrapper adds another encoding layer on top.

**Prevention:**
1. **Always use UTF-8 for sending:** Normalize all outgoing content to UTF-8. It is the default and handles all characters.
2. **Content-Type detection on read:** When reading emails, check the Content-Type charset header from the original email and decode accordingly.
3. **HTML escaping for JSON:** When sending HTML email via JSON body, properly escape all special characters. Use Go's `encoding/json` marshaler rather than manual string concatenation.
4. **Test with non-ASCII content:** Include test fixtures with Japanese, Chinese, Arabic, and emoji content.

**Confidence:** MEDIUM -- encoding options verified from Zoho Mail API docs. JSON parse issue reported in community forums (LOW confidence on specifics).

**Phase:** Phase 3 (mail send/read operations).

---

### Pitfall 11: Zoho Admin API Coverage Gaps

**What goes wrong:** The Zoho Mail Organization API does not provide direct CRUD endpoints for all admin operations. Based on API documentation review:

- **User management:** Limited to storage allocation; no documented endpoints for creating/deleting users via the Mail API specifically.
- **Domain management:** Listed as an API category but detailed CRUD operations are sparse.
- **Group management:** Members and roles are manageable, but advanced settings may be undocumented.
- **Audit logs:** The Logs API provides login history and SMTP logs, but granularity varies.

Some admin operations may require the Zoho Admin Console API (separate from Zoho Mail API) or the Zoho People/Directory APIs, which have different base URLs and authentication scopes.

**Why it happens:** Zoho Mail's API was designed primarily for mail operations. Organization management was added incrementally, and some operations may only be available through the web UI or different Zoho service APIs.

**Consequences:** The CLI promises "full admin operations" but discovers at implementation time that certain operations have no API endpoint. This leads to scope creep as you hunt for the right API, or feature gaps that disappoint users.

**Prevention:**
1. **API audit before roadmap finalization:** For each planned admin command, verify the exact API endpoint exists by making a test call with `curl`. Do not rely solely on the API index page.
2. **Feature flag unknown endpoints:** Mark commands as "experimental" or "unverified" when their API endpoint is not fully documented.
3. **Multiple API integration:** Be prepared to call different Zoho services (Mail API + Directory API + Admin Console API) for a complete admin feature set. Each may have different auth scopes and rate limits.

**Confidence:** MEDIUM -- API categories verified from https://www.zoho.com/mail/help/api/. Specific endpoint coverage is LOW confidence (requires per-endpoint verification).

**Phase:** Phase 2 (admin operations). Conduct API audit at the start of this phase.

---

## Minor Pitfalls

---

### Pitfall 12: Go Kong Framework Validation Requires Type Wrappers

**What goes wrong:** Kong validates CLI arguments via a `Validate() error` method on types. Since most arguments are basic types (`string`, `int`), you must create wrapper types for each validated argument. This leads to boilerplate:

```go
type EmailAddress string
func (e EmailAddress) Validate() error { /* ... */ }

type OrgID string
func (o OrgID) Validate() error { /* ... */ }
```

**Prevention:** Accept this as a Kong design decision. Create a `types/` package early with all validated types. This is actually a feature, not a bug -- it provides type safety for Zoho IDs, email addresses, region codes, etc.

**Confidence:** MEDIUM -- reported by Kong users and in documentation.

**Phase:** Phase 1 (CLI skeleton).

---

### Pitfall 13: Signal Handling for Long-Running Operations

**What goes wrong:** A user runs `zoh mail export --all` (potentially minutes of API calls with rate limiting), then presses Ctrl+C. Without proper signal handling, the process dies mid-request, potentially leaving:

- An incomplete export file
- A dangling HTTP connection
- An access token refresh in progress (corrupting the cached token)

**Prevention:**
1. Use `signal.NotifyContext` (Go 1.16+) to create a cancellable context. Pass it through all HTTP calls.
2. On first SIGINT: cancel in-flight requests, flush partial results, clean up temp files.
3. On second SIGINT: force-exit immediately.
4. Protect token refresh with a mutex so a signal during refresh does not corrupt the token file.

**Confidence:** HIGH -- standard Go pattern, well-documented.

**Phase:** Phase 1 (HTTP client layer). The context plumbing must be in place from the start.

---

### Pitfall 14: Cross-Compilation Breaks with Keyring CGo Dependencies

**What goes wrong:** The `99designs/keyring` library depends on C bindings for macOS Keychain access (`CoreFoundation` headers). Cross-compiling from Linux to macOS (or vice versa) with `CGO_ENABLED=1` fails with missing header errors. Even `CGO_ENABLED=0` may break keyring backends that require CGo.

**Prevention:**
1. **Build per-platform in CI:** Use GitHub Actions matrix builds with native runners for each OS.
2. **Build tag separation:** Use Go build tags to select keyring backends per OS, ensuring the macOS backend is only compiled on macOS.
3. **Test the release pipeline early:** Do not wait until v1.0 to try cross-platform builds.

**Confidence:** HIGH -- verified from https://github.com/99designs/keyring/issues/23

**Phase:** Phase 1 (project setup / CI). Set up CI matrix builds early.

---

### Pitfall 15: Zoho API Returns 200 OK with Error Body

**What goes wrong:** Some Zoho API endpoints return HTTP 200 with an error in the JSON body, rather than using appropriate HTTP status codes (4xx/5xx). This is documented behavior for Zoho CRM's GraphQL API and has been reported across other Zoho services. If your HTTP client only checks `resp.StatusCode`, it will miss these "successful" errors.

**Prevention:**
1. **Always parse the response body:** Check for `status.code` field in JSON responses, not just HTTP status.
2. **Unified error extraction:** Build a response parser that checks both HTTP status code and Zoho's JSON error structure before returning data.
3. **Define error types:** Create Go error types for Zoho-specific errors (`ZohoAPIError`) that carry the status code, error code, and message.

**Confidence:** MEDIUM -- verified for CRM GraphQL API. Likely applies to Mail API but not explicitly documented.

**Phase:** Phase 1 (HTTP client). The response parser must handle this from the start.

---

### Pitfall 16: Testing with Real Zoho APIs is Slow and Fragile

**What goes wrong:** Integration tests against real Zoho APIs are slow (rate limited to 30/min), flaky (dependent on Zoho uptime), require real credentials, and modify real data (sending test emails, creating test groups).

**Prevention:**
1. **Three test tiers:**
   - **Unit tests:** Mock the HTTP layer with `httptest.Server`. Test business logic, argument parsing, output formatting. Fast, no credentials needed.
   - **Contract tests:** Record real API responses once (`go-vcr` or similar), replay them in CI. Validates response parsing without hitting Zoho.
   - **Integration tests:** Run against real Zoho (tagged `//go:build integration`), only in CI with secrets, not on every commit.
2. **Configurable base URL:** The HTTP client must accept a base URL parameter so tests can point to `httptest.Server` instead of `mail.zoho.eu`.
3. **Test organization:** Use a dedicated Zoho test organization (free tier) to avoid polluting production data.
4. **Rate limit awareness in integration tests:** Add delays between API calls in integration tests to avoid hitting the 30/min limit.

**Confidence:** HIGH -- standard Go testing patterns, verified rate limit.

**Phase:** Phase 1 (project foundation). The test architecture must be established before writing any API-calling code.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Auth/Config (Phase 1) | Multi-DC client secret mismatch (Pitfall 3) | "Use same credentials for all DCs" or per-region secret storage |
| Auth/Config (Phase 1) | Token refresh race condition (Pitfall 2) | File-locked token cache with proactive refresh |
| Auth/Config (Phase 1) | Self-client grant expiry (Pitfall 8) | Guided setup wizard with pre-computed scopes |
| Auth/Config (Phase 1) | Keyring fails on WSL/headless (Pitfall 5) | Runtime backend detection with file fallback |
| Auth/Config (Phase 1) | Scope explosion (Pitfall 9) | Scope profiles with incremental authorization |
| HTTP Client (Phase 1) | 30 req/min rate limit (Pitfall 4) | Token bucket rate limiter in HTTP client |
| HTTP Client (Phase 1) | 200 OK with error body (Pitfall 15) | Unified response parser checks both HTTP status and JSON body |
| HTTP Client (Phase 1) | Region URL management (Pitfall 1) | RegionConfig struct, no hardcoded URLs |
| Admin Operations (Phase 2) | API coverage gaps (Pitfall 11) | Per-endpoint curl audit before implementation |
| Admin Operations (Phase 2) | Inconsistent pagination (Pitfall 7) | Abstract pagination iterator pattern |
| Mail Operations (Phase 3) | Attachment Content-Type lies (Pitfall 6) | Manual endpoint verification, golden test fixtures |
| Mail Operations (Phase 3) | Email encoding complexity (Pitfall 10) | UTF-8 normalization, charset detection on read |
| Testing (All Phases) | Real API tests are slow/fragile (Pitfall 16) | Three-tier test strategy: unit, contract, integration |
| Build/Release (Phase 1) | CGo cross-compilation (Pitfall 14) | Per-platform CI builds from the start |
| CLI Framework (Phase 1) | Kong validation boilerplate (Pitfall 12) | Dedicated types package, accept the design |
| CLI Framework (Phase 1) | Signal handling (Pitfall 13) | Context plumbing from day one |

---

## Sources

### Official Zoho Documentation (HIGH confidence)
- [Zoho OAuth Multi-DC Support](https://www.zoho.com/accounts/protocol/oauth/multi-dc.html)
- [Zoho OAuth Token Refresh](https://www.zoho.com/accounts/protocol/oauth/web-apps/access-token-expiry.html)
- [Zoho Self-Client Auth Code Flow](https://www.zoho.com/accounts/protocol/oauth/self-client/authorization-code-flow.html)
- [Zoho OAuth Scopes](https://www.zoho.com/accounts/protocol/oauth/scope.html)
- [Zoho Mail API Getting Started](https://www.zoho.com/mail/help/api/getting-started-with-api.html)
- [Zoho Mail API Index (all endpoints)](https://www.zoho.com/mail/help/api/)
- [Zoho Mail Rates and Limits](https://www.zoho.com/mail/help/adminconsole/rates-and-limits.html)
- [Zoho Mail Email API](https://www.zoho.com/mail/help/api/email-api.html)
- [Zoho Mail Send Email with Attachments](https://www.zoho.com/mail/help/api/post-send-email-attachment.html)
- [Zoho Mail Organization API](https://www.zoho.com/mail/help/api/organization-api.html)
- [Zoho CRM API Limits](https://www.zoho.com/crm/developer/docs/api/v8/api-limits.html)
- [Zoho CRM Access/Refresh Tokens](https://www.zoho.com/crm/developer/docs/api/v8/access-refresh.html)
- [Zoho Incremental Authorization](https://www.zoho.com/accounts/protocol/oauth/incremental-auth/scope-enhance-request.html)

### Library Documentation (HIGH confidence)
- [99designs/keyring GitHub](https://github.com/99designs/keyring)
- [99designs/keyring headless Linux issue #106](https://github.com/99designs/keyring/issues/106)
- [99designs/keyring cross-compilation issue #23](https://github.com/99designs/keyring/issues/23)
- [Kong CLI framework](https://github.com/alecthomas/kong)

### Community Reports (MEDIUM confidence)
- [Zoho Mail API Attachment Upload Gotchas](https://pebblesrox.wordpress.com/2021/03/28/zoho-mail-api-how-to-upload-an-attachment/)
- [n8n Community: Zoho Mail OAuth2 refresh token fails](https://community.n8n.io/t/zoho-mail-oauth2-refresh-token-fails/214296)
- [Zoho Integration Dos and Don'ts (Zenatta)](https://zenatta.com/the-dos-and-donts-of-zoho-integrations/)
- [Common Zoho Developer Mistakes (Expertia)](https://www.expertia.ai/career-tips/common-mistakes-to-avoid-as-a-zoho-developer-87927m)

### Go Best Practices (HIGH confidence)
- [Go Graceful Shutdown Patterns](https://victoriametrics.com/blog/go-graceful-shutdown/)
- [signal.NotifyContext](https://henvic.dev/posts/signal-notify-context/)
- [Go httptest Best Practices](https://speedscale.com/blog/testing-golang-with-httptest/)
