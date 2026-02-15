# zohod-cli Development Guide

## Architecture

Go CLI wrapping Zoho Mail & Admin APIs, built with [Kong](https://github.com/alecthomas/kong).

```
cmd/zoh/main.go          → CLI entry point
internal/
├── auth/                → OAuth2 flows, token cache, scopes
├── cli/                 → Command implementations + ServiceProvider
├── config/              → Config loading, region maps (8 data centers)
├── output/              → Formatters (JSON/plain/rich), CLIError, table utils
├── secrets/             → Keyring + encrypted file store (Store interface)
└── zoho/                → API clients, types, pagination, rate limiting
pkg/browser/             → System browser opener
```

## Dependency Injection

Commands receive `*ServiceProvider` (not `*config.Config`) for API access:

```go
func (cmd *SomeCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
    admin, err := sp.Admin()  // Returns zoho.AdminService (lazy, cached)
    // ...
}
```

- `ServiceProvider` uses `sync.Once` for thread-safe lazy init
- `Admin()` → `zoho.AdminService`, `Mail()` → `zoho.MailService`
- Config-only commands (auth, config) keep `cfg *config.Config` signatures
- `mail_admin.go` uses separate `MailAdminClient` — not yet in ServiceProvider

### Kong Binding

Dependencies are bound in `CLI.BeforeApply()` via `ctx.Bind()`. Kong injects them into `Run()` methods by matching parameter types. `FormatterProvider` wraps the `output.Formatter` interface because Kong can't bind interfaces directly.

## Testing

### Running Tests

```bash
go test ./... -v -race          # All tests
go test ./internal/zoho -cover  # Coverage for a package
```

### Patterns

- **Table-driven tests** with `testify/assert`
- Test files are same-package (not `_test` suffix package) to access unexported functions
- Pure function tests require zero mocks — see `types_test.go`, `regions_test.go`

### Naming

| Concept | Convention | Example |
|---------|-----------|---------|
| Interface | `*Service` | `AdminService`, `MailService` |
| Implementation | `*Client` | `AdminClient`, `MailClient` |
| Command | `*Cmd` | `AdminUsersListCmd` |

## Zoho API Quirks

**`accountId` is a quoted JSON string** — all other ID fields are raw numbers:
```json
{"accountId": "123456789", "zuid": 987654321}
```
This is verified by `TestUserJSONUnmarshal` in `types_test.go`.

**Response envelope**: All responses use `{"status": {"code": 200, ...}, "data": {...}}`.

**Scopes**: Zoho uses comma-separated scopes, not space-separated (standard OAuth2).

## Regions

8 data centers: `us`, `eu`, `in`, `au`, `jp`, `ca`, `sa`, `uk`. Default: `us`. Resolution order: CLI flag → config → default.
