# Architecture Patterns

**Domain:** Go CLI tool wrapping Zoho Admin + Mail REST APIs
**Researched:** 2026-02-14
**Overall Confidence:** HIGH (established Go patterns, verified against official Zoho docs and reference implementations)

## Recommended Architecture

The architecture follows a layered internal-package design modeled after the
[gogcli](https://github.com/steipete/gogcli) reference architecture, adapted for
Zoho's multi-region API surface and Kong's struct-based command routing. The
guiding principle is **one responsibility per package, interfaces at boundaries,
structs for everything else**.

```
zoh                              (module root: github.com/user/zoh)
+-- main.go                      (3-line bootstrap: parse + run)
+-- internal/
|   +-- cli/                     (Kong root struct + globals + hooks)
|   +-- cmd/                     (command implementations, one file per resource)
|   |   +-- admin/               (admin service commands)
|   |   +-- mail/                (mail service commands)
|   +-- zoho/                    (Zoho API client layer)
|   |   +-- client.go            (base HTTP client, region-aware)
|   |   +-- auth.go              (OAuth2 token source, refresh, multi-DC)
|   |   +-- admin/               (Admin API typed methods)
|   |   +-- mail/                (Mail API typed methods)
|   +-- auth/                    (OAuth flow orchestration: login, token lifecycle)
|   +-- config/                  (XDG config file, region, org settings)
|   +-- secrets/                 (OS keyring abstraction)
|   +-- output/                  (JSON / plain / rich formatters)
|   +-- ui/                      (terminal colors, spinners, prompts)
+-- go.mod
+-- go.sum
+-- Makefile
```

### Why This Layout

- **`internal/`** enforces that nothing outside the module imports these packages,
  giving freedom to refactor without SemVer concerns (per
  [Go official layout guidance](https://go.dev/doc/modules/layout)).
- **Flat-ish hierarchy** -- two levels max inside `internal/`. Go idiom is shallow
  packages; deep nesting is an anti-pattern.
- **Separate `cli/` from `cmd/`** -- `cli/` owns the Kong struct tree and global
  flags; `cmd/` owns the `Run()` implementations. This prevents the root CLI
  struct from becoming a 2000-line monolith.
- **`zoho/` as a standalone API client layer** -- has zero knowledge of CLI
  concerns. Can be extracted to a separate module later if others want a Go SDK.

---

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| `main.go` | Bootstrap Kong parser, call `ctx.Run()` | `cli/` |
| `cli/` | Define Kong root struct, global flags, hooks (`BeforeApply`/`AfterApply`), bind dependencies | `cmd/*`, `config/`, `auth/`, `output/` |
| `cmd/admin/` | Admin service command `Run()` methods (org, users, groups, domains) | `zoho/admin/`, `output/` |
| `cmd/mail/` | Mail service command `Run()` methods (messages, folders, labels, etc.) | `zoho/mail/`, `output/` |
| `zoho/client.go` | Region-aware base HTTP client with auth transport | `zoho/auth.go`, `net/http` |
| `zoho/auth.go` | Custom `oauth2.TokenSource` wrapping refresh tokens, persists to keyring | `secrets/`, `config/` |
| `zoho/admin/` | Typed Go methods for Zoho Admin/Organization REST endpoints | `zoho/client.go` |
| `zoho/mail/` | Typed Go methods for Zoho Mail REST endpoints | `zoho/client.go` |
| `auth/` | High-level OAuth login flow (browser open, local callback, device code) | `zoho/auth.go`, `secrets/`, `config/`, `ui/` |
| `config/` | Read/write XDG config files, resolve region, store org ID / account defaults | `adrg/xdg` |
| `secrets/` | Keyring get/set/delete behind an interface (real + mock backends) | `zalando/go-keyring` |
| `output/` | Format any result as JSON, plain (TSV), or rich table | `cmd/*` (called by commands) |
| `ui/` | Terminal helpers: colors, spinners, interactive prompts | `cmd/*`, `auth/` |

---

### Data Flow

The data flows in a strict top-down direction. No lower layer ever imports a
higher one.

```
User types command
       |
       v
  [main.go]  kong.Parse(&cli)
       |
       v
  [cli/]     BeforeApply hook: loads config, resolves region,
             initializes auth client, binds deps via kong.Bind()
       |
       v
  [cmd/*]    Selected command's Run() receives bound dependencies:
             - *zoho.Client (already authenticated, region-correct)
             - *output.Formatter
             - *config.Config
       |
       v
  [zoho/*]   Command calls typed API method, e.g.:
             zoho/admin.ListUsers(ctx, orgID) -> []User
             Internally: builds URL, calls client.Do(req), decodes JSON
       |
       v
  [zoho/client.go]   http.Client with custom oauth2.Transport
                     Auto-refreshes token on 401, selects regional base URL
       |
       v
  [Zoho REST API]    https://mail.zoho.{region}/api/...
       |
       v
  [Response]  JSON decoded into typed Go structs
       |
       v
  [output/]   Command passes result to formatter
             formatter.Print(result) -> JSON / TSV / rich table to stdout
       |
       v
  User sees output
```

**Error flow:** Errors propagate upward via `error` returns. The command layer
decides whether to retry, surface to the user, or exit. The `zoho/` layer wraps
HTTP errors into typed `*zoho.APIError` structs that include status code,
Zoho error code, and description.

---

## Patterns to Follow

### Pattern 1: Kong Struct Tree with Embedded Globals

**What:** Define the entire CLI as nested Go structs. Kong uses struct tags to
build the command tree. Global flags live in an embedded struct accessible to
all commands.

**When:** Always -- this is how Kong works.

**Confidence:** HIGH (verified via [Kong docs](https://github.com/alecthomas/kong)
and [Daniel Michaels' Kong patterns](https://danielms.site/zet/2024/how-i-write-golang-cli-tools-today-using-kong/))

**Example:**

```go
// internal/cli/cli.go

type Globals struct {
    Region  string `help:"Zoho region (us,eu,in,au,jp,ca,sa,uk)" default:"us" enum:"us,eu,in,au,jp,ca,sa,uk,cn,ae" env:"ZOH_REGION"`
    Output  string `help:"Output format" default:"rich" enum:"json,plain,rich" short:"o"`
    Verbose bool   `help:"Enable verbose output" short:"v"`
    Config  string `help:"Config file path" type:"path" env:"ZOH_CONFIG"`
}

type CLI struct {
    Globals

    // Service-first commands
    Admin admin.Cmd `cmd:"" help:"Zoho Admin / Organization management"`
    Mail  mail.Cmd  `cmd:"" help:"Zoho Mail operations"`

    // Action-first shortcuts (aliases into service commands)
    Users    admin.UsersCmd    `cmd:"" help:"List/manage users (shortcut for admin users)" hidden:""`
    Send     mail.SendCmd      `cmd:"" help:"Send an email (shortcut for mail send)" hidden:""`

    // Meta
    Login    auth.LoginCmd     `cmd:"" help:"Authenticate with Zoho"`
    Config   config.ConfigCmd  `cmd:"" help:"View/edit configuration"`
    Version  kong.VersionFlag  `name:"version" help:"Show version"`
}
```

### Pattern 2: Interface-Based API Client for Testing

**What:** Define an interface for each Zoho service client. The real
implementation calls HTTP; tests inject a mock.

**When:** Every service boundary.

**Confidence:** HIGH (standard Go testing pattern, verified via
[testify/mockery best practices](https://www.buanacoding.com/2025/10/how-to-use-mock-testing-in-go-with-testify-and-mockery.html))

**Example:**

```go
// internal/zoho/admin/admin.go

type Service interface {
    GetOrganization(ctx context.Context) (*Organization, error)
    ListUsers(ctx context.Context, orgID string) ([]User, error)
    AddUser(ctx context.Context, orgID string, req AddUserRequest) (*User, error)
    ListGroups(ctx context.Context, orgID string) ([]Group, error)
    ListDomains(ctx context.Context, orgID string) ([]Domain, error)
    // ... expanded per Zoho Mail Organization API surface
}

type service struct {
    client *zoho.Client
}

func New(client *zoho.Client) Service {
    return &service{client: client}
}

func (s *service) ListUsers(ctx context.Context, orgID string) ([]User, error) {
    var resp usersResponse
    err := s.client.Get(ctx, "/api/organization/"+orgID+"/users", &resp)
    if err != nil {
        return nil, fmt.Errorf("list users: %w", err)
    }
    return resp.Data.Users, nil
}
```

### Pattern 3: Region-Aware Base Client with OAuth Transport

**What:** A single `zoho.Client` that resolves the correct regional base URL
and wraps `http.Client` with an `oauth2.Transport` for automatic token refresh.

**When:** All API calls go through this client.

**Confidence:** HIGH (Go `x/oauth2` Transport pattern is [official](https://pkg.go.dev/golang.org/x/oauth2);
Zoho multi-DC [documented](https://www.zoho.com/accounts/protocol/oauth/multi-dc.html))

**Example:**

```go
// internal/zoho/client.go

type Client struct {
    httpClient *http.Client
    baseURL    string    // e.g., "https://mail.zoho.eu"
    accountsURL string  // e.g., "https://accounts.zoho.eu"
}

// RegionConfig maps region codes to Zoho base URLs
var regions = map[string]RegionEndpoints{
    "us": {Accounts: "https://accounts.zoho.com",    Mail: "https://mail.zoho.com"},
    "eu": {Accounts: "https://accounts.zoho.eu",     Mail: "https://mail.zoho.eu"},
    "in": {Accounts: "https://accounts.zoho.in",     Mail: "https://mail.zoho.in"},
    "au": {Accounts: "https://accounts.zoho.com.au", Mail: "https://mail.zoho.com.au"},
    "jp": {Accounts: "https://accounts.zoho.jp",     Mail: "https://mail.zoho.jp"},
    "ca": {Accounts: "https://accounts.zohocloud.ca",Mail: "https://mail.zohocloud.ca"},
    "sa": {Accounts: "https://accounts.zoho.sa",     Mail: "https://mail.zoho.sa"},
    "uk": {Accounts: "https://accounts.zoho.uk",     Mail: "https://mail.zoho.uk"},
    "cn": {Accounts: "https://accounts.zoho.com.cn", Mail: "https://mail.zoho.com.cn"},
    "ae": {Accounts: "https://accounts.zoho.ae",     Mail: "https://mail.zoho.ae"},
}

func NewClient(region string, tokenSource oauth2.TokenSource) (*Client, error) {
    endpoints, ok := regions[region]
    if !ok {
        return nil, fmt.Errorf("unknown region: %s", region)
    }
    return &Client{
        httpClient: oauth2.NewClient(context.Background(), tokenSource),
        baseURL:    endpoints.Mail,
        accountsURL: endpoints.Accounts,
    }, nil
}
```

### Pattern 4: Persistent Token Source with Keyring Backend

**What:** A custom `oauth2.TokenSource` that reads the refresh token from the OS
keyring, uses it to obtain access tokens, and persists refreshed tokens back to
the keyring. Wraps `x/oauth2`'s built-in refresh logic.

**When:** Token lifecycle management.

**Confidence:** HIGH (Go `oauth2.TokenSource` is a one-method interface; keyring
persistence is the pattern used by gogcli and gh CLI)

**Example:**

```go
// internal/zoho/auth.go

type KeyringTokenSource struct {
    config   *oauth2.Config
    keyring  secrets.Keyring   // interface for testability
    service  string            // keyring service name
    account  string            // keyring account key
}

func (k *KeyringTokenSource) Token() (*oauth2.Token, error) {
    // 1. Load stored token from keyring
    stored, err := k.keyring.Get(k.service, k.account)
    if err != nil {
        return nil, fmt.Errorf("no stored token: %w", err)
    }
    token := deserializeToken(stored)

    // 2. If valid, return it
    if token.Valid() {
        return token, nil
    }

    // 3. If expired, refresh using the refresh token
    src := k.config.TokenSource(context.Background(), token)
    newToken, err := src.Token()
    if err != nil {
        return nil, fmt.Errorf("token refresh failed: %w", err)
    }

    // 4. Persist the refreshed token back to keyring
    if err := k.keyring.Set(k.service, k.account, serializeToken(newToken)); err != nil {
        return nil, fmt.Errorf("persist token: %w", err)
    }

    return newToken, nil
}
```

### Pattern 5: Service Registration for Extensibility

**What:** Each Zoho service (Admin, Mail, future: CRM, Books, etc.) is a
self-contained package under `internal/zoho/` with its own types, and a
corresponding command package under `internal/cmd/`. Adding a new service
means adding two packages -- no existing code changes.

**When:** Adding any new Zoho service.

**Confidence:** HIGH (follows Go's package-level modularity; same pattern as
gogcli which wraps 15+ Google services)

**How to add a new service (e.g., CRM):**

```
1. Create internal/zoho/crm/        -- API client methods + types
   - crm.go                         -- Service interface + implementation
   - types.go                       -- Request/response structs

2. Create internal/cmd/crm/         -- Command implementations
   - crm.go                         -- Cmd struct (Kong sub-commands)
   - leads.go                       -- Run() for lead commands
   - deals.go                       -- Run() for deal commands

3. Register in internal/cli/cli.go:
   CRM crm.Cmd `cmd:"" help:"Zoho CRM operations"`

Done. No other files touched.
```

### Pattern 6: Output Formatter Interface

**What:** Commands receive an `output.Formatter` and call `Print(data)`. The
formatter decides JSON, TSV, or rich table based on global flag.

**When:** Every command that produces output.

**Confidence:** HIGH (directly from gogcli's `outfmt` package pattern)

**Example:**

```go
// internal/output/formatter.go

type Formatter interface {
    Print(v any) error
    PrintTable(headers []string, rows [][]string) error
    PrintError(err error)
}

type formatter struct {
    mode   string     // "json", "plain", "rich"
    writer io.Writer  // usually os.Stdout
}

func New(mode string, w io.Writer) Formatter {
    return &formatter{mode: mode, writer: w}
}

func (f *formatter) Print(v any) error {
    switch f.mode {
    case "json":
        return json.NewEncoder(f.writer).Encode(v)
    case "plain":
        return printTSV(f.writer, v)
    case "rich":
        return printRichTable(f.writer, v)
    default:
        return fmt.Errorf("unknown output mode: %s", f.mode)
    }
}
```

### Pattern 7: Kong Dependency Injection via Bind + Hooks

**What:** Use Kong's `BeforeApply` hook on the root CLI struct to initialize
shared dependencies (config, auth client, formatter), then `kong.Bind()` them
so command `Run()` methods receive them as parameters.

**When:** Wiring dependencies at startup.

**Confidence:** HIGH (documented Kong pattern, used in
[gogcli](https://github.com/steipete/gogcli) and
[Kong README](https://github.com/alecthomas/kong))

**Example:**

```go
// main.go

func main() {
    var cli cli.CLI
    ctx := kong.Parse(&cli,
        kong.Name("zoh"),
        kong.Description("Zoho Admin & Mail CLI"),
        kong.DefaultEnvars("ZOH"),
        kong.Vars{"version": version},
    )
    err := ctx.Run(&cli.Globals)
    ctx.FatalIfErrorf(err)
}

// internal/cmd/admin/users.go

type UsersListCmd struct{}

func (c *UsersListCmd) Run(globals *cli.Globals, admin admin.Service, fmt output.Formatter) error {
    users, err := admin.ListUsers(context.Background(), globals.OrgID)
    if err != nil {
        return err
    }
    return fmt.Print(users)
}
```

---

## Anti-Patterns to Avoid

### Anti-Pattern 1: God Package

**What:** Putting all command implementations, API calls, and formatting in one
package (e.g., everything in `cmd/`).

**Why bad:** Untestable, unmaintainable, impossible for contributors to navigate.
A single `commands.go` file with 3000 lines is where CLI projects go to die.

**Instead:** One package per concern. Commands in `cmd/`, API clients in `zoho/`,
formatting in `output/`. Each package testable in isolation.

### Anti-Pattern 2: Passing Config Everywhere

**What:** Threading a `*Config` struct through every function signature.

**Why bad:** Creates tight coupling; every function depends on the config shape.
Changes to config propagate through the entire codebase.

**Instead:** Resolve config into concrete values during initialization (in the
`BeforeApply` hook). Pass only what each component needs: the API client gets
a region string and token source, commands get a pre-configured client and
formatter. No package below `cli/` should import `config/` directly.

### Anti-Pattern 3: Region Logic Scattered Across Packages

**What:** Each API method building its own base URL from a region string.

**Why bad:** Duplicated logic, easy to miss a region, hard to test.

**Instead:** Region resolution happens once in `zoho.NewClient()`. The `Client`
struct holds the resolved base URL. All service packages use `client.Get()`
/ `client.Post()` which prepends the base URL automatically.

### Anti-Pattern 4: Hardcoded HTTP Client

**What:** Using `http.DefaultClient` or creating `http.Client{}` inside API
methods.

**Why bad:** Cannot inject test servers, cannot add auth transport, cannot set
timeouts.

**Instead:** Accept `*http.Client` or `*zoho.Client` via constructor injection.
The client is configured once with OAuth transport, timeouts, and retry logic.

### Anti-Pattern 5: Testing Against Live APIs

**What:** Integration tests that call real Zoho endpoints.

**Why bad:** Slow, flaky, requires credentials in CI, rate-limited.

**Instead:** Interface-based mocks for unit tests. `httptest.Server` for
integration tests that verify HTTP request construction. Live API tests only
as a separate manual/nightly suite.

---

## Detailed Component Design

### Config Layer (`internal/config/`)

```go
// Config represents the persistent configuration stored at
// $XDG_CONFIG_HOME/zoh/config.json
type Config struct {
    DefaultRegion string `json:"default_region"`
    DefaultOrgID  string `json:"default_org_id"`
    DefaultOutput string `json:"default_output"`
    Accounts      map[string]AccountConfig `json:"accounts"`
}

type AccountConfig struct {
    Region   string `json:"region"`
    OrgID    string `json:"org_id"`
    ClientID string `json:"client_id"`
    // client_secret and tokens stored in keyring, NOT here
}
```

**Key decisions:**
- Config file at `$XDG_CONFIG_HOME/zoh/config.json` (via `adrg/xdg`)
- Secrets (client_secret, refresh_token) NEVER in config file -- always in keyring
- Support multiple accounts with named profiles
- Config is read-only after initialization; commands do not mutate it

### Secrets Layer (`internal/secrets/`)

```go
// Keyring abstracts OS keyring operations for testability
type Keyring interface {
    Get(service, account string) (string, error)
    Set(service, account, password string) error
    Delete(service, account string) error
}

// OSKeyring wraps zalando/go-keyring
type OSKeyring struct{}

func (k *OSKeyring) Get(service, account string) (string, error) {
    return keyring.Get(service, account)
}

// MockKeyring for testing (also available via keyring.MockInit())
type MockKeyring struct {
    store map[string]string
}
```

**Key decisions:**
- Interface wrapping `zalando/go-keyring` so tests never touch real keyring
- Service name: `"zoh"`, account keys: `"{profile}-refresh-token"`,
  `"{profile}-client-secret"`
- Keyring size limits: macOS ~3KB, Windows ~2.5KB -- tokens fit easily

### Auth Flow (`internal/auth/`)

This package orchestrates the initial OAuth login flow:

1. User runs `zoh login`
2. Open browser to `https://accounts.zoho.{region}/oauth/v2/auth?...`
3. Start local HTTP server on `localhost:PORT` for redirect
4. Receive authorization code via redirect
5. Exchange code for access + refresh tokens via `/oauth/v2/token`
6. Store refresh token in keyring, client ID/secret in keyring
7. Store region + org ID in config file
8. Handle `other_dc` error: detect correct region, retry against right DC

**Zoho-specific auth notes (from [Zoho OAuth docs](https://www.zoho.com/accounts/protocol/oauth/web-server-applications.html)):**
- Access tokens valid for 1 hour
- Authorization codes valid for 2 minutes
- Multi-DC: if user is in different DC than app, get `other_dc` error with
  correct location -- must retry
- Auth header format: `Authorization: Zoho-oauthtoken {token}` (NOT `Bearer`)
- Client type: "Non-browser based application" for CLI use

### Zoho API Client (`internal/zoho/`)

```
zoho/
+-- client.go      Base client: region URLs, HTTP methods, error handling
+-- auth.go        KeyringTokenSource (oauth2.TokenSource implementation)
+-- errors.go      Typed API errors with Zoho status codes
+-- regions.go     Region-to-URL mapping (10 data centers)
+-- admin/         Admin/Organization service
|   +-- service.go     Service interface + constructor
|   +-- organization.go  Org endpoints
|   +-- users.go       User endpoints
|   +-- groups.go      Group endpoints
|   +-- domains.go     Domain endpoints
|   +-- types.go       Request/response structs
+-- mail/          Mail service
    +-- service.go     Service interface + constructor
    +-- messages.go    Message endpoints
    +-- folders.go     Folder endpoints
    +-- labels.go      Label endpoints
    +-- accounts.go    Account settings endpoints
    +-- types.go       Request/response structs
```

**Response handling pattern (from [Zoho Mail API docs](https://www.zoho.com/mail/help/api/getting-started-with-api.html)):**

```go
// Zoho wraps all responses in a status envelope
type Response[T any] struct {
    Status struct {
        Code        int    `json:"code"`
        Description string `json:"description"`
    } `json:"status"`
    Data T `json:"data"`
}
```

**Key identifiers required by Zoho APIs:**
- `zoid` -- Zoho Organization ID (stored in config)
- `zuid` -- Zoho User ID (derived from auth)
- `accountId` -- Mail account ID (fetched via accounts API)

### Command Layer (`internal/cmd/`)

```
cmd/
+-- admin/
|   +-- cmd.go           Cmd struct with sub-commands
|   +-- org.go           zoh admin org [get|update]
|   +-- users.go         zoh admin users [list|add|delete|update]
|   +-- groups.go        zoh admin groups [list|create|delete|members]
|   +-- domains.go       zoh admin domains [list|add|verify]
+-- mail/
    +-- cmd.go           Cmd struct with sub-commands
    +-- messages.go      zoh mail messages [list|read|send|reply|delete]
    +-- folders.go       zoh mail folders [list|create|rename|delete]
    +-- labels.go        zoh mail labels [list|create|delete]
    +-- accounts.go      zoh mail accounts [list|get]
```

**Each command file contains:**
- A Kong-tagged struct defining flags/args for that command
- A `Run()` method that receives injected dependencies
- Zero business logic -- delegates entirely to the service interface

---

## Testing Architecture

### Unit Tests (fast, no I/O)

| Layer | What to Test | How |
|-------|-------------|-----|
| `zoho/admin/` | Request construction, response parsing | Mock `zoho.Client` via interface |
| `zoho/mail/` | Request construction, response parsing | Mock `zoho.Client` via interface |
| `zoho/auth.go` | Token refresh logic, keyring persistence | Mock `secrets.Keyring` |
| `cmd/*` | Command orchestration, error handling | Mock service interfaces |
| `output/` | JSON/TSV/table rendering | Golden file comparisons |
| `config/` | Config read/write, defaults | Temp directory + file I/O |

### Integration Tests (medium speed, `httptest`)

| Layer | What to Test | How |
|-------|-------------|-----|
| `zoho/client.go` | Full HTTP request/response cycle | `httptest.NewServer` with canned responses |
| `zoho/admin/` | End-to-end service call through real HTTP | `httptest.NewServer` |
| `auth/` | OAuth flow with redirect capture | `httptest.NewServer` as fake Zoho accounts |

### Interface Definitions for Mocking

Every external boundary gets an interface:

```go
// Keyring (secrets/)     -- mock for keyring ops
// Service (zoho/admin/)  -- mock for API calls
// Service (zoho/mail/)   -- mock for API calls
// Formatter (output/)    -- mock/capture for output verification
// Client (zoho/)         -- mock for HTTP layer (optional, httptest preferred)
```

Use `mockery` to generate mocks from interfaces automatically.

---

## Suggested Build Order

The build order is determined by dependency relationships. Each layer depends
only on layers below it.

```
Phase 1: Foundation (no Zoho API calls possible yet)
  config/ -> secrets/ -> zoho/regions.go -> zoho/errors.go

Phase 2: Auth (can authenticate but not call APIs)
  zoho/auth.go -> zoho/client.go -> auth/ (login flow)

Phase 3: First Service (prove the full stack works end-to-end)
  zoho/admin/ -> cmd/admin/ -> cli/ (wire it up) -> main.go

Phase 4: Output Formatting (make it usable)
  output/ -> integrate into cmd/admin/

Phase 5: Second Service (validate extensibility)
  zoho/mail/ -> cmd/mail/ -> register in cli/

Phase 6: Polish
  ui/ (colors, spinners) -> action-first shortcuts -> shell completions
```

**Why this order:**

1. **Config + Secrets first** because everything depends on knowing the region
   and having access to stored credentials.
2. **Auth second** because you cannot test any API call without a valid token.
   Auth is the "unblocking" dependency.
3. **Admin service third** (not Mail) because Admin endpoints are simpler
   (mostly list/get operations) and prove the full request/response pipeline.
4. **Output fourth** because raw JSON output is usable for testing even before
   formatters exist.
5. **Mail service fifth** to validate that adding a second service requires
   zero changes to existing code (proves the extensibility pattern).
6. **UI polish last** because it is cosmetic and does not affect functionality.

---

## Scalability Considerations

This is a CLI tool, not a server, so "scalability" means handling growing API
surfaces and contributor count, not request throughput.

| Concern | Now (2 services) | Later (5 services) | Eventually (15+ services) |
|---------|-------------------|--------------------|----|
| Package count | ~15 packages | ~25 packages | ~50 packages |
| Build time | Sub-second | ~2 seconds | ~5 seconds |
| Contributor onboarding | Read any one `cmd/` + `zoho/` pair | Same pattern, different directory | Same pattern, auto-generated stubs possible |
| Binary size | ~10-15 MB | ~20 MB | ~30 MB (still single binary) |
| Command tree depth | `zoh admin users list` | Same | May need `zoh crm leads list` grouping |

**When to consider splitting the module:** If a standalone Zoho Go SDK becomes
valuable to external consumers, extract `internal/zoho/` into a separate
`github.com/user/zoho-go` module. The CLI then imports it as a dependency.

---

## Zoho API Surface Mapping

Based on the [Zoho Mail API Index](https://www.zoho.com/mail/help/api/), the
full API surface is larger than "Admin + Mail" suggests. The Mail API includes:

| API Category | CLI Scope | Priority |
|-------------|-----------|----------|
| Organization API | Admin service | Phase 3 |
| Domain API | Admin service | Phase 3 |
| Users API | Admin service | Phase 3 |
| Groups API | Admin service | Phase 3 |
| Mail Policy API | Admin service | Phase 5+ |
| Accounts API | Mail service | Phase 5 |
| Folders API | Mail service | Phase 5 |
| Labels API | Mail service | Phase 5 |
| Email Messages API | Mail service | Phase 5 |
| Threads API | Mail service | Phase 5+ |
| Signatures API | Mail service | Phase 5+ |
| Tasks API | Separate service? | Defer |
| Bookmarks API | Separate service? | Defer |
| Notes API | Separate service? | Defer |
| Logs API | Admin service | Phase 5+ |

---

## Sources

### HIGH Confidence
- [Go official module layout](https://go.dev/doc/modules/layout) -- official Go team guidance
- [Kong CLI framework](https://github.com/alecthomas/kong) -- official README and pkg.go.dev
- [Zoho OAuth2 documentation](https://www.zoho.com/accounts/protocol/oauth/web-server-applications.html) -- official Zoho docs
- [Zoho Multi-DC support](https://www.zoho.com/accounts/protocol/oauth/multi-dc.html) -- official Zoho docs
- [Zoho Mail API Getting Started](https://www.zoho.com/mail/help/api/getting-started-with-api.html) -- official Zoho docs
- [Zoho Mail API Index](https://www.zoho.com/mail/help/api/) -- official Zoho docs
- [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) -- official Go package
- [zalando/go-keyring](https://github.com/zalando/go-keyring) -- official README + API docs
- [adrg/xdg](https://github.com/adrg/xdg) -- official README + API docs

### MEDIUM Confidence
- [gogcli reference architecture](https://github.com/steipete/gogcli) -- real-world Go CLI wrapping Google APIs, same domain
- [Kong patterns by Daniel Michaels](https://danielms.site/zet/2024/how-i-write-golang-cli-tools-today-using-kong/) -- practical guide, verified against Kong docs
- [Kong patterns by Miek Gieben](https://miek.nl/2024/november/01/kong-go-cli/) -- custom mapper patterns, verified
- [testify/mockery best practices](https://www.buanacoding.com/2025/10/how-to-use-mock-testing-in-go-with-testify-and-mockery.html) -- community guide, consistent with official testify docs
- [Go project structure by Glukhov](https://www.glukhov.org/post/2025/12/go-project-structure/) -- recent community guidance

### LOW Confidence
- [schmorrison/Zoho Go library](https://github.com/schmorrison/Zoho) -- community library, Mail support incomplete; useful as reference but not a dependency
- Zoho rate limits -- documented as "per-endpoint" but specific values not found in public docs; needs validation during implementation
