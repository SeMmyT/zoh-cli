# Stack Research

**Domain:** Go CLI wrapping Zoho REST APIs (Admin + Mail)
**Researched:** 2026-02-14
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.23+ | Language | Single binary, strong typing, proven for CLI tools. gogcli uses Go 1.25 |
| Kong | latest (tag-based, no releases) | CLI framework | Struct-tag command definitions, used by gogcli. Declarative, type-safe, introspectable |
| golang.org/x/oauth2 | latest | OAuth2 client | Standard Go OAuth2 library, handles token refresh, custom token sources |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| 99designs/keyring | v1.2.2 | Secure credential storage | OS keychain abstraction — macOS Keychain, Linux Secret Service, Windows Credential Manager, encrypted file fallback |
| muesli/termenv | latest | Terminal capability detection | Color support detection, TTY detection, respects NO_COLOR |
| rodaine/table | latest | Table output | Rich mode terminal tables, works with ANSI colors |
| fatih/color | latest | Terminal colors | ANSI color output for rich mode |
| yosuke-furukawa/json5 | latest | Config parsing | JSON5 config files (comments, trailing commas) — human-editable config |
| stretchr/testify | v1.11.1 | Testing toolkit | Assertions (assert/require), mocks, test suites — essential for TDD |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| GoReleaser v2.13+ | Cross-platform builds & releases | CGO_ENABLED=1 for macOS Keychain, =0 for Linux/Windows |
| golangci-lint | Linting | Multi-linter runner, gogcli uses 47+ linters |
| goimports | Import formatting | Auto-organize imports with local prefix |
| gofumpt | Code formatting | Stricter than gofmt |
| lefthook | Git hooks | Pre-commit/pre-push automation |

## Installation

```bash
# Initialize Go module
go mod init github.com/<user>/zohcli

# Core dependencies
go get github.com/alecthomas/kong
go get golang.org/x/oauth2
go get github.com/99designs/keyring

# Terminal UI
go get github.com/muesli/termenv
go get github.com/rodaine/table
go get github.com/fatih/color

# Config
go get github.com/yosuke-furukawa/json5

# Testing
go get github.com/stretchr/testify

# Dev tools (pinned in .tools/)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest
go install mvdan.cc/gofumpt@latest
```

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| Kong | Cobra + Viper | Cobra is more popular but requires more boilerplate. Kong's struct-tag approach is cleaner and matches gogcli patterns. Cobra's code-generation approach is heavier |
| 99designs/keyring | zalando/go-keyring | zalando is simpler but lacks file-based fallback for headless environments. 99designs supports more backends (D-Bus, KWallet, Pass, file) |
| muesli/termenv | fatih/color alone | termenv provides terminal capability detection (true color, 256 color) that fatih/color lacks. Use together — termenv for detection, fatih for coloring |
| rodaine/table | olekukonko/tablewriter | rodaine/table is simpler and works better with ANSI colors. tablewriter has more features but heavier API |
| json5 | encoding/json | JSON5 supports comments and trailing commas — essential for human-editable config files |
| testify | stdlib only | stdlib testing is fine for simple tests but testify's assertions and mock package are essential for TDD workflow with readable test failures |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| schmorrison/Zoho | Minimally maintained (last commit Jun 2024), doesn't cover Mail, requires Go 1.13, uses vendoring | Build our own thin API client — Zoho's REST API is simple enough |
| Zoho official Go SDK | Only exists for Analytics, not Mail or Admin | Build our own |
| bubbletea/lipgloss | Overkill for a data-oriented CLI — designed for interactive TUIs | rodaine/table + fatih/color for output formatting |
| Viper | Kong handles flags/env/defaults natively. Viper adds unnecessary complexity | Kong struct tags + json5 config file |
| go-resty | Heavy HTTP client with features we don't need | net/http with custom transport for auth — keeps dependency surface small |

## Stack Patterns by Variant

**If adding interactive features (prompts, selection):**
- Use `github.com/erikgeiser/promptkit` or `github.com/charmbracelet/huh`
- Only for `auth login` flow and destructive confirmations

**If email body rendering needed (rich mode):**
- Use `github.com/jaytaylor/html2text` for HTML email → terminal
- Only for `mail get` with HTML bodies

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| Kong | Go 1.18+ | Uses generics in newer versions |
| 99designs/keyring v1.2.2 | Go 1.18+ | CGO required for macOS Keychain backend |
| testify v1.11.1 | Go 1.20+ | Latest release requires newer Go |
| GoReleaser v2.13+ | Go 1.22+ | Build tool, not a dependency |

## Sources

- [Kong GitHub](https://github.com/alecthomas/kong) — CLI framework, no formal releases (uses Go module tags)
- [99designs/keyring releases](https://github.com/99designs/keyring/releases) — v1.2.2 (Dec 2024)
- [testify releases](https://github.com/stretchr/testify/releases) — v1.11.1 (Aug 2024)
- [schmorrison/Zoho](https://github.com/schmorrison/Zoho) — Minimally maintained, Mail NOT implemented
- [gogcli](https://github.com/steipete/gogcli) — Reference architecture, Go 1.25, Kong-based
- [GoReleaser](https://goreleaser.com/) — v2.13+ (Dec 2025)
- [Zoho Mail API docs](https://www.zoho.com/mail/help/api/) — REST API, OAuth2 auth

---
*Stack research for: Zoho Admin + Mail CLI*
*Researched: 2026-02-14*
