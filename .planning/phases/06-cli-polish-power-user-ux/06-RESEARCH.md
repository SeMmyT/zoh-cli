# Phase 6: CLI Polish & Power User UX - Research

**Researched:** 2026-02-14
**Domain:** CLI UX patterns, Kong framework capabilities, shell completion
**Confidence:** HIGH

## Summary

Phase 6 adds power-user features to make the zoh CLI scriptable, composable in pipelines, and faster for experts. The research reveals that Kong provides the foundation through aliases, hooks, and introspection, but key features like dry-run, schema generation, and shell completion require custom implementation layers.

Kong's `aliases` tag enables desire-path shortcuts without code duplication. Shell completion is well-supported through dedicated libraries (kongplete, kong-completion). The challenge lies in implementing consistent dry-run semantics across mutating commands and generating machine-readable schema from Kong's command tree.

**Primary recommendation:** Build a three-layer approach: (1) Use Kong's native `aliases` tag for command shortcuts, (2) Implement scripting flags through global flags and BeforeApply hooks for validation, and (3) Add shell completion via kongplete library with custom predictors for dynamic completions.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/alecthomas/kong | v1.14.0 | CLI parser | Already in use, provides aliases, hooks, introspection |
| github.com/willabides/kongplete | v0.4.0+ | Shell completion | Best-maintained Kong completion library, supports all 3 shells |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/posener/complete | Latest | Completion predictor engine | Required by kongplete for dynamic completions |
| encoding/json | stdlib | Schema generation | Serialize command tree to JSON |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| kongplete | jotaen/kong-completion | kong-completion is simpler but less actively maintained, fewer predictors |
| Custom schema | Kong's built-in help | Built-in help is human-readable only, not machine-parsable JSON |

**Installation:**
```bash
go get github.com/willabides/kongplete@latest
go get github.com/posener/complete@latest
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── cli/
│   ├── cli.go              # Root CLI with Globals
│   ├── globals.go          # Global flags (add scripting flags here)
│   ├── shortcuts.go        # NEW: Shortcut command mappings
│   ├── schema.go           # NEW: Schema introspection command
│   └── completion.go       # NEW: Shell completion setup
├── dryrun/
│   ├── context.go          # NEW: DryRun context propagation
│   └── validator.go        # NEW: Validation helpers
```

### Pattern 1: Desire-Path Shortcuts with Kong Aliases

**What:** Use Kong's `aliases` tag to provide action-first shortcuts alongside hierarchical commands.

**When to use:** When power users want `zoh send` instead of `zoh mail send compose`.

**Example:**
```go
// In cli.go
type CLI struct {
    Globals

    // Hierarchical commands (full paths)
    Mail    MailCmd    `cmd:"" help:"Mail operations"`
    Admin   AdminCmd   `cmd:"" help:"Admin operations"`

    // Desire-path shortcuts (action-first)
    Send    MailSendComposeCmd `cmd:"" aliases:"compose" help:"Send an email (shortcut for mail send compose)"`
    Ls      LsCmd              `cmd:"" help:"List resources (users, groups, folders)"`
}

// Ls subcommand with resource-specific shortcuts
type LsCmd struct {
    Users   AdminUsersListCmd   `cmd:"" help:"List users (shortcut for admin users list)"`
    Groups  AdminGroupsListCmd  `cmd:"" help:"List groups (shortcut for admin groups list)"`
    Folders MailFoldersListCmd  `cmd:"" help:"List folders (shortcut for mail folders list)"`
}
```

**Key insight:** Kong resolves aliases at parse time, so shortcuts share the same Run() implementation as full paths. No code duplication.

### Pattern 2: Scripting Flags via Global Flags and Hooks

**What:** Add global flags for automation (`--results-only`, `--no-input`, `--force`, `--dry-run`) and validate them in BeforeApply hooks.

**When to use:** When users need deterministic, non-interactive behavior for scripts and CI/CD pipelines.

**Example:**
```go
// In globals.go
type Globals struct {
    Region      string `help:"Zoho region" default:"" enum:"us,eu,in,au,jp,ca,sa,uk," env:"ZOH_REGION"`
    Output      string `help:"Output format" default:"auto" enum:"json,plain,rich,auto" short:"o" env:"ZOH_OUTPUT"`
    Verbose     bool   `help:"Verbose output" short:"v" env:"ZOH_VERBOSE"`

    // Scripting flags (Phase 6)
    ResultsOnly bool   `help:"Strip JSON envelope, return data array only" env:"ZOH_RESULTS_ONLY"`
    NoInput     bool   `help:"Disable interactive prompts (fail instead)" env:"ZOH_NO_INPUT"`
    Force       bool   `help:"Skip confirmation prompts for destructive operations" env:"ZOH_FORCE"`
    DryRun      bool   `help:"Preview operation without executing" env:"ZOH_DRY_RUN"`
}

// In cli.go BeforeApply
func (c *CLI) BeforeApply(ctx *kong.Context) error {
    // ... existing config/region/formatter setup ...

    // Validate flag combinations
    if c.Force && c.DryRun {
        return fmt.Errorf("cannot use --force with --dry-run")
    }

    if c.Force && c.NoInput {
        // --force implies skipping confirmations, redundant with --no-input for confirms
        // This is OK, just log if verbose
    }

    // Bind globals so commands can check flags
    ctx.Bind(&c.Globals)

    return nil
}
```

**Key insight:** Global flags propagate to all subcommands automatically. BeforeApply provides a single validation point.

### Pattern 3: DryRun Context Propagation

**What:** Pass dry-run state through context to API clients, preventing actual API calls.

**When to use:** For mutating commands (create, update, delete, send).

**Example:**
```go
// internal/dryrun/context.go
package dryrun

import "context"

type contextKey int

const dryRunKey contextKey = 0

// WithDryRun creates a context with dry-run enabled
func WithDryRun(ctx context.Context, enabled bool) context.Context {
    return context.WithValue(ctx, dryRunKey, enabled)
}

// IsDryRun checks if dry-run is enabled in context
func IsDryRun(ctx context.Context) bool {
    v, ok := ctx.Value(dryRunKey).(bool)
    return ok && v
}

// internal/cli/admin_users.go (example mutation command)
func (cmd *AdminUsersCreateCmd) Run(cfg *config.Config, fp *FormatterProvider, globals *Globals) error {
    ctx := context.Background()

    // Propagate dry-run flag to context
    if globals.DryRun {
        ctx = dryrun.WithDryRun(ctx, true)
        fmt.Fprintf(os.Stderr, "[DRY RUN] Would create user: %s\n", cmd.Email)
    }

    adminClient, err := newAdminClient(cfg)
    if err != nil {
        return err
    }

    // AdminClient.CreateUser checks dryrun.IsDryRun(ctx) internally
    user, err := adminClient.CreateUser(ctx, &zoho.CreateUserRequest{
        Email:     cmd.Email,
        FirstName: cmd.FirstName,
        LastName:  cmd.LastName,
    })

    if globals.DryRun {
        fmt.Fprintf(os.Stderr, "[DRY RUN] User would be created with ID: %s\n", user.ZUID)
        return nil
    }

    return fp.Formatter.Print(user)
}

// internal/zoho/admin_client.go
func (c *AdminClient) CreateUser(ctx context.Context, req *CreateUserRequest) (*User, error) {
    if dryrun.IsDryRun(ctx) {
        // Return a mock response, don't call API
        return &User{
            ZUID:      "MOCK-ZUID-" + req.Email,
            Email:     req.Email,
            FirstName: req.FirstName,
            LastName:  req.LastName,
            Status:    "active",
        }, nil
    }

    // Normal API call...
}
```

**Key insight:** Context propagation keeps dry-run logic centralized. API clients control whether to mock or execute.

### Pattern 4: Results-Only Output Filtering

**What:** Strip JSON envelope when `--results-only` is set, returning raw data array.

**When to use:** For pipeline composition where users pipe output to jq or other tools.

**Example:**
```go
// internal/output/formatter.go
type jsonFormatter struct {
    resultsOnly bool  // NEW: Set from globals
}

func (f *jsonFormatter) PrintList(items any, columns []Column) error {
    if f.resultsOnly {
        // Skip envelope, print raw array
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(items)
    }

    // Default: wrap in envelope
    envelope := map[string]any{
        "data":  items,
        "count": reflect.ValueOf(items).Len(),
    }
    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(envelope)
}

// In cli.go BeforeApply
func (c *CLI) BeforeApply(ctx *kong.Context) error {
    // ... existing config/region setup ...

    // Create formatter with results-only setting
    var formatter output.Formatter
    if c.ResolvedOutput() == "json" {
        formatter = output.NewJSON(c.ResultsOnly)
    } else {
        formatter = output.New(c.ResolvedOutput())
    }

    ctx.Bind(&FormatterProvider{Formatter: formatter})
    // ...
}
```

**Key insight:** Results-only is an output formatter concern, not a command concern. Centralize in formatter.

### Pattern 5: Schema Introspection with Kong's Node API

**What:** Generate machine-readable JSON from Kong's command tree using the Node structure.

**When to use:** For tooling, documentation generation, or API clients that need the full command structure.

**Example:**
```go
// internal/cli/schema.go
package cli

import (
    "encoding/json"
    "os"

    "github.com/alecthomas/kong"
)

type SchemaCmd struct {
    Command string `arg:"" optional:"" help:"Specific command to show schema for (empty = full tree)"`
}

type SchemaNode struct {
    Name        string                 `json:"name"`
    Type        string                 `json:"type"`  // "command" | "flag" | "arg"
    Help        string                 `json:"help,omitempty"`
    Required    bool                   `json:"required,omitempty"`
    Default     string                 `json:"default,omitempty"`
    Enum        []string               `json:"enum,omitempty"`
    Aliases     []string               `json:"aliases,omitempty"`
    Children    map[string]*SchemaNode `json:"children,omitempty"`
    Flags       map[string]*SchemaNode `json:"flags,omitempty"`
}

func (cmd *SchemaCmd) Run(ctx *kong.Context) error {
    rootNode := ctx.Model.Node

    if cmd.Command != "" {
        // Find specific command node
        selectedNode := findNodeByPath(rootNode, cmd.Command)
        if selectedNode == nil {
            return fmt.Errorf("command not found: %s", cmd.Command)
        }
        rootNode = selectedNode
    }

    schema := buildSchemaNode(rootNode)

    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(schema)
}

func buildSchemaNode(n *kong.Node) *SchemaNode {
    schema := &SchemaNode{
        Name: n.Name,
        Type: nodeTypeString(n.Type),
        Help: n.Help,
    }

    // Extract flags
    flagGroups := n.AllFlags(false)
    if len(flagGroups) > 0 {
        schema.Flags = make(map[string]*SchemaNode)
        for _, group := range flagGroups {
            for _, flag := range group {
                schema.Flags[flag.Name] = &SchemaNode{
                    Name:     flag.Name,
                    Type:     "flag",
                    Help:     flag.Help,
                    Required: flag.Required,
                    Default:  flag.Default,
                    Aliases:  flag.Aliases,
                }
            }
        }
    }

    // Recurse into children
    if len(n.Children) > 0 {
        schema.Children = make(map[string]*SchemaNode)
        for _, child := range n.Children {
            schema.Children[child.Name] = buildSchemaNode(child)
        }
    }

    return schema
}

func nodeTypeString(t kong.NodeType) string {
    switch t {
    case kong.ApplicationNode:
        return "application"
    case kong.CommandNode:
        return "command"
    case kong.ArgumentNode:
        return "argument"
    default:
        return "unknown"
    }
}

func findNodeByPath(root *kong.Node, path string) *kong.Node {
    parts := strings.Split(path, " ")
    current := root

    for _, part := range parts {
        found := false
        for _, child := range current.Children {
            if child.Name == part {
                current = child
                found = true
                break
            }
        }
        if !found {
            return nil
        }
    }

    return current
}
```

**Source:** Based on Kong pkg.go.dev documentation of Node structure and methods.

### Pattern 6: Shell Completion with Kongplete

**What:** Generate shell-specific completion scripts using kongplete's InstallCompletions command.

**When to use:** To enable tab completion for bash, zsh, and fish shells.

**Example:**
```go
// internal/cli/cli.go
import (
    "github.com/willabides/kongplete"
    "github.com/posener/complete"
)

type CLI struct {
    Globals

    // ... existing commands ...

    // Completion command
    InstallCompletions kongplete.InstallCompletions `cmd:"" help:"Install shell completions"`
}

// main.go
func main() {
    cliInstance := &cli.CLI{}

    parser := kong.Must(cliInstance,
        kong.Name("zoh"),
        kong.Description("Zoho CLI for Admin and Mail operations"),
        kong.UsageOnError(),
        kong.ConfigureHelp(kong.HelpOptions{
            Compact: true,
        }),
        kong.Vars{
            "version": version,
        },
    )

    // Add completion support BEFORE parsing
    kongplete.Complete(parser,
        kongplete.WithPredictor("file", complete.PredictFiles("*")),
        kongplete.WithPredictor("email", complete.PredictAnything), // Custom predictor for emails
    )

    ctx := parser.Parse(os.Args[1:])

    err := ctx.Run()
    // ... error handling ...
}
```

**Custom predictor example:**
```go
// Predictor for user identifiers (email or zuid)
var userPredictor = complete.PredictFunc(func(args complete.Args) []string {
    // Could fetch recent users from cache or API
    // For now, return empty (no predictions)
    return []string{}
})

kongplete.Complete(parser,
    kongplete.WithPredictor("user", userPredictor),
)

// In command struct
type AdminUsersGetCmd struct {
    Identifier string `arg:"" help:"User ID or email" predictor:"user"`
}
```

**Installation:**
```bash
# User runs once to install completions
zoh install-completions bash
# Output: Add this to your ~/.bashrc:
# source <(zoh install-completions bash --print)

# Or install directly
zoh install-completions bash --install
```

**Source:** Based on kongplete pkg.go.dev examples.

### Anti-Patterns to Avoid

- **Duplicating command structs for shortcuts:** Use Kong's `aliases` tag instead. Duplication creates maintenance burden and bloats the binary.
- **Implementing dry-run at the command level:** Context propagation to API clients is cleaner and ensures consistency.
- **Parsing Kong help text for schema:** Use Kong's Node introspection API directly for accurate, stable schema generation.
- **Building custom completion logic:** Use kongplete. Shell completion is complex with many edge cases.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Shell completion | Custom bash/zsh/fish scripts | kongplete + posener/complete | Handles quoting, escaping, dynamic predictions, multi-shell support |
| Command tree introspection | Text parsing of --help | Kong's Node API (ctx.Model.Node) | Type-safe, accurate, includes aliases and metadata |
| Dry-run API mocking | Per-command mock logic | Context-based flag + centralized client logic | Prevents duplication, ensures consistency |
| Results filtering | Custom JSON munging per command | Formatter-level envelope control | Single source of truth for output format |

**Key insight:** Kong provides introspection primitives (Node, Flags, AllFlags, Visit). Build on these, don't reimplement.

## Common Pitfalls

### Pitfall 1: Alias Name Collisions

**What goes wrong:** Defining an alias that conflicts with an existing command or flag name causes Kong parse errors.

**Why it happens:** Kong validates that all command/flag names and aliases are unique at parse time.

**How to avoid:**
- Namespace shortcuts under a parent command (e.g., `zoh ls users` instead of `zoh users`)
- Test thoroughly with `go test` to catch collisions
- Document aliases in --help output

**Warning signs:**
- Parse errors like "duplicate command name"
- Completions suggesting the same command twice

### Pitfall 2: Dry-Run Not Respecting Side Effects

**What goes wrong:** Dry-run shows preview but still modifies state (e.g., incrementing counters, creating temp files, opening network connections).

**Why it happens:** Only the final API call is skipped, but validation logic still executes.

**How to avoid:**
- Check `dryrun.IsDryRun(ctx)` at ALL side-effect points (file I/O, network, etc.)
- Use mock clients in tests to verify no side effects
- Document which validations run during dry-run (e.g., "checks credentials, does not send email")

**Warning signs:**
- Test failures when running with --dry-run
- Users reporting "dry-run still did something"

### Pitfall 3: Results-Only Breaking Non-List Commands

**What goes wrong:** `--results-only` strips envelope on single-object commands, making output ambiguous.

**Why it happens:** Results-only is designed for lists but gets applied globally.

**How to avoid:**
- Only apply `--results-only` to commands that return arrays/lists
- Document that `--results-only` is ignored for single-object commands
- Alternative: make `--results-only` error on non-list commands

**Warning signs:**
- User confusion about when `--results-only` applies
- Inconsistent output structure

### Pitfall 4: Force and NoInput Interaction

**What goes wrong:** `--force` and `--no-input` have overlapping semantics. Users don't know which to use.

**Why it happens:** Both affect prompts but in different contexts.

**How to avoid:**
- Define clear semantics:
  - `--no-input`: Fail if ANY input would be requested (includes missing required args)
  - `--force`: Skip confirmation prompts for destructive operations, but still error on missing args
- Document the difference in --help
- Test combinations to ensure predictable behavior

**Warning signs:**
- Users always use both flags together
- Confusion in issues/support requests

### Pitfall 5: Incomplete Schema Coverage

**What goes wrong:** Generated schema is missing flags, aliases, or validation rules.

**Why it happens:** Kong's Node API doesn't expose all metadata, or traversal logic is incomplete.

**How to avoid:**
- Test schema generation against ALL commands
- Validate generated schema with a JSON schema validator
- Document known limitations (e.g., "custom validators not included")

**Warning signs:**
- Third-party tools fail to parse schema
- Schema doesn't match actual CLI behavior

### Pitfall 6: Shell Completion Performance

**What goes wrong:** Completion becomes slow when fetching dynamic predictions (e.g., listing all users from API).

**Why it happens:** Shell completion runs on every tab press. API calls add latency.

**How to avoid:**
- Cache dynamic completions with TTL
- Timeout completions after 100ms, fall back to empty predictions
- Document that some completions require network access

**Warning signs:**
- Users complain about slow tab completion
- Network errors during completion

## Code Examples

Verified patterns from official sources:

### Kong Aliases (Command Shortcuts)
```go
// Source: https://github.com/alecthomas/kong/issues/66
type CLI struct {
    List    ListCmd `cmd:"" aliases:"ls" help:"List resources"`
    Remove  RmCmd   `cmd:"" aliases:"rm,delete" help:"Remove resource"`
}
```

### BeforeApply Hook for Validation
```go
// Source: https://pkg.go.dev/github.com/alecthomas/kong
type CLI struct {
    Globals
}

func (c *CLI) BeforeApply(cfg *Config, logger *log.Logger) error {
    // Validate flag combinations
    if c.Force && c.DryRun {
        return fmt.Errorf("cannot use --force with --dry-run")
    }
    return nil
}
```

### Node Traversal for Schema
```go
// Source: https://pkg.go.dev/github.com/alecthomas/kong
func traverseCommands(node *kong.Node) {
    for _, child := range node.Children {
        fmt.Println(child.Name, child.Help)
        traverseCommands(child)
    }
}
```

### Kongplete Integration
```go
// Source: https://pkg.go.dev/github.com/willabides/kongplete
parser := kong.Must(&cli)
kongplete.Complete(parser,
    kongplete.WithPredictor("file", complete.PredictFiles("*")),
)
ctx, err := parser.Parse(os.Args[1:])
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `--dry-run=true/false` boolean flag | `--dry-run=client/server/none` enum (kubectl pattern) | kubectl 1.18 (2020) | Client-side vs server-side preview semantics |
| Manual completion scripts per shell | Unified completion library (kongplete, cobra.Command.GenBashCompletion) | 2020-2021 | One codebase generates all shells |
| Help text parsing for docs | Schema introspection APIs (Kong Node, Cobra Command tree) | 2019-2021 | Reliable programmatic access |
| `--json` flag per command | Global `--output=json` flag with formatter abstraction | 2018-2020 | Consistent across all commands |

**Deprecated/outdated:**
- **Boolean --dry-run flag:** Modern CLIs use enum (`client`, `server`, `none`) to distinguish local vs remote validation. For zoh, boolean is acceptable since we have one execution context.
- **Custom shell scripts:** Libraries like kongplete handle completion generation. Manual scripts are fragile.
- **--quiet + --json combination:** Replaced by `--results-only` or `--output=json` with envelope control.

## Open Questions

1. **Should `--dry-run` work on read operations?**
   - What we know: Read operations don't mutate, so dry-run seems unnecessary
   - What's unclear: Users might expect `--dry-run` to show query parameters without executing
   - Recommendation: Document that `--dry-run` only affects mutating commands (create, update, delete, send). Read operations ignore it.

2. **How should `--results-only` behave with single-object commands?**
   - What we know: Designed for lists to strip `{"data": [...], "count": N}` envelope
   - What's unclear: Should it error, warn, or silently pass through for single objects?
   - Recommendation: Silently pass through (return object as-is). Document that it only affects list commands.

3. **Should shell completion fetch dynamic data from API?**
   - What we know: Tab completion runs frequently, API calls add latency and consume rate limit
   - What's unclear: Users might expect email/name completions for identifiers
   - Recommendation: Start with static completions only (commands, flags). Add cached dynamic completions in a future iteration if requested.

4. **Should shortcuts override existing commands?**
   - What we know: `zoh ls` is shorter than `zoh admin users list`, but `ls` is generic
   - What's unclear: Should `zoh ls` be a parent command with `users`, `groups`, `folders` subcommands, or should each be a top-level shortcut?
   - Recommendation: Use parent command `zoh ls` with resource subcommands. Prevents namespace pollution.

## Sources

### Primary (HIGH confidence)
- [Kong GitHub Repository](https://github.com/alecthomas/kong) - Official docs, README, examples
- [Kong pkg.go.dev](https://pkg.go.dev/github.com/alecthomas/kong) - API documentation for Node, Context, hooks
- [Kongplete pkg.go.dev](https://pkg.go.dev/github.com/willabides/kongplete) - Completion library docs and examples
- [Kong-completion GitHub](https://github.com/jotaen/kong-completion) - Alternative completion library
- [Kong aliases issue #66](https://github.com/alecthomas/kong/issues/66) - Aliases feature discussion and implementation

### Secondary (MEDIUM confidence)
- [GitHub CLI Power Tips (2026)](https://onlyutkarsh.com/posts/2026/github-cli-power-tips/) - Real-world examples of --json, --jq flags
- [Terraform Plan Command Explained](https://spacelift.io/blog/terraform-plan) - Dry-run best practices
- [kubectl dry-run documentation](https://nunoadrego.com/posts/kubectl-dry-run/) - Client/server dry-run patterns
- [Kong BeforeApply hooks issue #124](https://github.com/alecthomas/kong/issues/124) - Hook execution details

### Tertiary (LOW confidence)
- [CLI scripting flags search results](https://helgeklein.com/blog/tips-for-devops-pipeline-automation-bash-scripting/) - General CLI automation patterns, not specific to Go
- [Cobra CLI guides](https://cobra.dev/docs/how-to-guides/working-with-commands/) - Alternative framework, different patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Kong and kongplete are documented, widely used, and stable
- Architecture: HIGH - Patterns verified with official docs and examples
- Pitfalls: MEDIUM - Based on common CLI design issues and some Kong-specific quirks

**Research date:** 2026-02-14
**Valid until:** 2026-03-14 (30 days - Kong is stable, completion libraries mature)
