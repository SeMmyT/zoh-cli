---
phase: 06-cli-polish-power-user-ux
plan: 02
subsystem: CLI UX
tags: [dry-run, force-flag, preview, scripting-safety]
dependencies:
  requires: [internal/cli/globals.go, internal/cli/admin_*.go, internal/cli/mail_*.go]
  provides: [dry-run-preview, force-bypass]
  affects: [all-mutating-commands]
tech-stack:
  added: []
  patterns: [preview-before-execute, confirmation-bypass]
key-files:
  created: []
  modified:
    - internal/cli/admin_users.go
    - internal/cli/admin_groups.go
    - internal/cli/admin_domains.go
    - internal/cli/mail_send.go
    - internal/cli/mail_settings.go
    - internal/cli/mail_admin.go
decisions:
  - title: "Dry-run output to stderr with [DRY RUN] prefix"
    rationale: "Keeps stdout clean for piping actual data; clear visual indicator; matches industry standards (kubectl, terraform)"
    alternatives: ["stdout output", "no prefix", "different prefix"]
    chosen: "stderr with [DRY RUN] prefix"
  - title: "--force bypasses --confirm on delete operations"
    rationale: "Power users in scripts need non-interactive deletion; --force conveys destructive intent; validates before API calls"
    alternatives: ["--yes flag", "environment variable", "config setting"]
    chosen: "--force global flag"
  - title: "Dry-run before client initialization"
    rationale: "Fastest preview path; no authentication needed; validates parameters only; reduces API calls to zero"
    alternatives: ["dry-run after auth", "API dry-run mode"]
    chosen: "client-side preview before auth"
metrics:
  duration: 6min
  completed: 2026-02-14
---

# Phase 06 Plan 02: Dry-Run Preview and Force Bypass Summary

**One-liner:** Global --dry-run preview for all 21 mutating commands with stderr output, and --force flag to bypass --confirm on delete operations for scripted workflows.

## Objective Achieved

Added --dry-run preview to all mutating CLI commands across admin and mail operations, enabling safe operation verification before execution. Added --force bypass for delete confirmations to support scripted workflows.

**Purpose:** Power users can preview destructive operations before executing them (--dry-run), and skip confirmation prompts in automation scripts (--force), improving safety and automation support.

**Output:** 21 mutating commands now check globals.DryRun and print preview to stderr without making API calls. Delete commands accept --force as alternative to --confirm.

## Tasks Completed

| Task | Description | Commit | Key Changes |
|------|-------------|--------|-------------|
| 1 | Admin dry-run and force | a0f9ece | 13 admin commands: users (5), groups (5), domains (3); force bypass for 2 delete commands |
| 2 | Mail dry-run | 02c0927 | 8 mail commands: send (3), settings (4), admin (1) |

## Key Implementation Details

### Dry-Run Pattern (Tasks 1 & 2)

**Consistent implementation across all mutating commands:**

```go
func (cmd *SomeCmd) Run(cfg *config.Config, globals *Globals) error {
    // Dry-run preview
    if globals.DryRun {
        fmt.Fprintf(os.Stderr, "[DRY RUN] Would <action>: <details>\n")
        return nil
    }

    // Actual implementation...
}
```

**Key characteristics:**
- Check happens BEFORE client initialization (no auth, no API calls)
- Output goes to stderr (stdout stays clean for piping)
- `[DRY RUN]` prefix for clear visual indication
- Returns nil immediately (no execution)

### Admin Commands Dry-Run (Task 1)

**User operations (admin_users.go):**
1. `AdminUsersCreateCmd` - "Would create user: {email} (firstName={FirstName}, lastName={LastName})"
2. `AdminUsersUpdateCmd` - "Would update user {identifier}: role={Role}"
3. `AdminUsersActivateCmd` - "Would activate user: {identifier}"
4. `AdminUsersDeactivateCmd` - "Would deactivate user: {identifier}"
5. `AdminUsersDeleteCmd` - "Would permanently delete user: {identifier}"

**Group operations (admin_groups.go):**
1. `AdminGroupsCreateCmd` - "Would create group: {name} (email={EmailId})"
2. `AdminGroupsUpdateCmd` - "Would update group {groupId}: name={Name}"
3. `AdminGroupsDeleteCmd` - "Would permanently delete group: {groupId}"
4. `AdminGroupsMembersAddCmd` - "Would add {N} member(s) to group {groupId}"
5. `AdminGroupsMembersRemoveCmd` - "Would remove {N} member(s) from group {groupId}"

**Domain operations (admin_domains.go):**
1. `AdminDomainsAddCmd` - "Would add domain: {DomainName}"
2. `AdminDomainsVerifyCmd` - "Would verify domain {DomainName} using method={Method}"
3. `AdminDomainsUpdateCmd` - "Would update domain {DomainName}: mode={Mode}"

**All commands:**
- Added `globals *Globals` parameter to Run() methods that lacked it
- Commands that previously only took `cfg *config.Config` now take both

### Force Bypass for Delete Commands (Task 1)

**Changed delete commands in admin_users.go and admin_groups.go:**

**Before:**
```go
type AdminUsersDeleteCmd struct {
    Confirm bool `help:"..." required:""`  // Required flag
}
```

**After:**
```go
type AdminUsersDeleteCmd struct {
    Confirm bool `help:"..."`  // Optional flag
}

func (cmd *AdminUsersDeleteCmd) Run(..., globals *Globals) error {
    // Check confirmation requirement
    if !cmd.Confirm && !globals.Force && !globals.DryRun {
        return &output.CLIError{
            Message:  "Deletion requires --confirm or --force flag",
            ExitCode: output.ExitUsage,
        }
    }
    // ... rest of delete logic
}
```

**Benefits:**
- `--force` bypasses --confirm requirement (scripting support)
- `--dry-run` also bypasses confirmation (preview doesn't need it)
- Still requires EITHER --confirm OR --force (safety net maintained)
- Clear error message guides users

### Mail Commands Dry-Run (Task 2)

**Send operations (mail_send.go):**

1. `MailSendComposeCmd` - Detailed email preview:
   ```
   [DRY RUN] Would send email:
     To: user@example.com
     Cc: other@example.com
     Subject: Test Email
     Attachments: 2 file(s)
   ```

2. `MailSendReplyCmd` - "Would reply to message {MessageId} (reply-all={ReplyAll})"

3. `MailSendForwardCmd` - "Would forward message {MessageId} to {To}"

**Settings operations (mail_settings.go):**

1. `MailSettingsSignaturesCreateCmd` - "Would create signature: name={Name}"
2. `MailSettingsVacationSetCmd` - "Would enable vacation reply: subject={Subject}"
3. `MailSettingsVacationDisableCmd` - "Would disable vacation auto-reply"
4. `MailSettingsDisplayNameSetCmd` - "Would update display name to: {Name}"

**Admin operations (mail_admin.go):**

1. `MailAdminSpamUpdateCmd` - "Would update {Category}: add {len(Addresses)} address(es)"

**All commands:**
- Added `globals *Globals` parameter (mail commands previously only took `cfg *config.Config`)
- Compose command shows multi-line detailed preview (To/Cc/Bcc/Subject/Attachments)

## Verification Results

All verification steps passed:

1. ✓ `go build ./...` - compiles without errors
2. ✓ `go vet ./...` - passes with no issues
3. ✓ 23 dry-run checks found across all command files (13 admin + 8 mail + 2 force checks)
4. ✓ Delete commands have force bypass validation
5. ✓ Read-only commands (list, get, search) have NO dry-run logic (correctly ignored)

**Grep verification:**
- `grep -rn "globals\.DryRun"` → 23 matches
- `grep -rn "globals\.Force"` → 2 matches (both delete commands)

## Deviations from Plan

None - plan executed exactly as written.

**All specified functionality implemented:**
- 21 mutating commands check globals.DryRun (13 admin + 8 mail)
- Dry-run output to stderr with [DRY RUN] prefix
- Delete commands accept --force OR --confirm
- Read-only commands silently ignore --dry-run (no logic added)
- Project compiles and passes vet

## Code Quality

**Standards met:**
- Zero compiler errors/warnings
- Consistent dry-run pattern across all commands
- Clear, descriptive preview messages
- Proper parameter validation before API calls
- stderr for status messages, stdout for data (established pattern)

**Design patterns:**
- Early return on dry-run (guard clause pattern)
- Validation before execution (fail fast)
- Consistent error messaging
- No code duplication (same pattern everywhere)

## Integration Points

**Affects:**
- All 21 mutating commands now support --dry-run flag
- 2 delete commands now support --force flag
- No changes to read-only commands (correct behavior)

**Future enhancements enabled:**
- Dry-run can be extended to show API request payloads
- Force flag can be used for other confirmation-required operations
- Pattern established for future mutating commands

## Testing Notes

**Manual verification scenarios:**

1. **Dry-run works without authentication:**
   - `zoh admin users create test@example.com --dry-run` → No auth needed, prints preview

2. **Force bypasses confirm:**
   - `zoh admin users delete user@example.com --force` → Works without --confirm

3. **Dry-run bypasses confirm:**
   - `zoh admin users delete user@example.com --dry-run` → No confirm needed for preview

4. **Validation enforces safety:**
   - `zoh admin users delete user@example.com` → Error: "requires --confirm or --force"

5. **Mail commands preview correctly:**
   - `zoh mail send compose --to user@example.com --subject Test --body "Hi" --dry-run` → Multi-line preview

## Files Changed

**Modified:**
- internal/cli/admin_users.go (added dry-run to 5 commands, force bypass to delete)
- internal/cli/admin_groups.go (added dry-run to 5 commands, force bypass to delete, globals param to 5 commands)
- internal/cli/admin_domains.go (added dry-run to 3 commands, globals param to 3 commands)
- internal/cli/mail_send.go (added dry-run to 3 commands, globals param to 3 commands)
- internal/cli/mail_settings.go (added dry-run to 4 commands, globals param to 4 commands)
- internal/cli/mail_admin.go (added dry-run to 1 command, globals param to 1 command)

**Total:** 6 files modified, 21 commands updated, 16 command signatures changed (added globals parameter)

## Performance Impact

**Minimal:**
- Dry-run check is single boolean comparison (negligible)
- Dry-run prevents API calls (actually IMPROVES performance for preview use case)
- No additional allocations or processing in normal execution path
- Force flag check adds one conditional (negligible)

**Benefits:**
- Dry-run eliminates unnecessary API calls during testing
- Users can verify operations before execution (prevents mistakes)
- Scripts can use --force to avoid interactive prompts (faster automation)

## Self-Check

### Files Modified

- [x] internal/cli/admin_users.go has dry-run in 5 commands, force bypass in delete
- [x] internal/cli/admin_groups.go has dry-run in 5 commands, force bypass in delete
- [x] internal/cli/admin_domains.go has dry-run in 3 commands
- [x] internal/cli/mail_send.go has dry-run in 3 commands
- [x] internal/cli/mail_settings.go has dry-run in 4 commands
- [x] internal/cli/mail_admin.go has dry-run in 1 command

### Commits Exist

- [x] a0f9ece (Task 1: admin dry-run and force)
- [x] 02c0927 (Task 2: mail dry-run)

### Functionality Verified

- [x] `go build ./...` compiles successfully
- [x] `go vet ./...` passes
- [x] 23 dry-run checks found (correct count)
- [x] 2 force bypass checks found (delete commands only)
- [x] All mutating commands have dry-run support
- [x] No read-only commands have dry-run logic

## Self-Check Result: PASSED

All files modified correctly, commits are in git history, code compiles and passes vet, correct number of dry-run and force checks implemented.

## Next Steps

Phase 06 Plan 03: Interactive improvements (progress indicators, better error messages, confirmation prompts)
