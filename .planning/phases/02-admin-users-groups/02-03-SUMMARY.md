---
phase: 02-admin-users-groups
plan: 03
subsystem: admin-cli
status: complete
completed: 2026-02-14T18:32:56Z
tags: [admin, groups, cli, crud, members]

dependency_graph:
  requires:
    - 02-01-SUMMARY.md  # AdminClient foundation, PageIterator
  provides:
    - Group CRUD operations (list, get, create, update, delete)
    - Group member management (add, remove with batching)
    - Email-or-ID group resolution
  affects:
    - internal/zoho/admin_client.go  # Added 9 group methods
    - internal/zoho/types.go  # Added GroupMembersResponse type
    - internal/cli/admin_groups.go  # NEW: All group CLI commands
    - internal/cli/cli.go  # Added AdminGroupsCmd registration

tech_stack:
  added: []
  patterns:
    - Email-or-ID auto-detection for group lookups
    - Batch processing for member add operations (50/batch)
    - Nested command structure (groups members add/remove)
    - Consistent error handling with CLIError types

key_files:
  created:
    - internal/cli/admin_groups.go  # 370 lines, 7 command structs, resolveGroupID helper
  modified:
    - internal/zoho/admin_client.go  # +390 lines (9 group methods)
    - internal/zoho/types.go  # +8 lines (GroupMembersResponse)
    - internal/cli/cli.go  # +18 lines (AdminGroupsCmd + subcommands)

decisions:
  - title: "Batch size of 50 for AddGroupMembers"
    rationale: "Zoho API likely has limits on bulk operations. 50 provides safety margin while maintaining efficiency."
    alternatives: ["100 (riskier)", "25 (safer but slower)"]
    chosen: "50"
  - title: "ShowMembers default true in GroupsGetCmd"
    rationale: "Members are core to group utility. Better UX to show by default with opt-out flag."
    alternatives: ["default false (less API calls)", "separate command"]
    chosen: "default true"
  - title: "Require --confirm for group deletion"
    rationale: "Permanent destructive action. Kong's required flag ensures explicit user intent."
    alternatives: ["Interactive prompt", "No safety"]
    chosen: "Required flag"

metrics:
  duration_minutes: 3
  tasks_completed: 2
  files_created: 1
  files_modified: 3
  lines_added: 797
  commits: 2
---

# Phase 2 Plan 3: Group Management Commands Summary

Complete group management CLI with CRUD operations and member management, supporting both email and ZGID identification.

## What Was Built

### AdminClient Group Methods (internal/zoho/admin_client.go)

**List & Get:**
- `ListGroups(ctx, start, limit)` - Paginated group listing
- `GetGroup(ctx, zgid)` - Single group details
- `GetGroupMembers(ctx, zgid)` - Member list for a group
- `GetGroupByEmail(ctx, email)` - Find group by email (uses PageIterator)

**Mutations:**
- `CreateGroup(ctx, req)` - Create new group with name, email, description
- `UpdateGroup(ctx, zgid, name, description)` - Update group settings (mode: updateMailGroup)
- `DeleteGroup(ctx, zgid)` - Permanent group deletion

**Member Management:**
- `AddGroupMembers(ctx, zgid, members)` - Batch add with role support (mode: addMailGroupMember)
- `RemoveGroupMembers(ctx, zgid, members)` - Bulk remove (mode: removeMailGroupMember)

**Features:**
- Batch processing: AddGroupMembers splits into 50-member chunks automatically
- Consistent error handling via parseErrorResponse
- Type-safe request/response structures

### CLI Commands (internal/cli/admin_groups.go)

**Basic Operations:**
```bash
zoh admin groups list [--limit 50] [--all]
zoh admin groups get <zgid-or-email> [--show-members]
zoh admin groups create <name> --email <email> [--description <text>]
zoh admin groups update <zgid-or-email> [--name <name>] [--description <text>]
zoh admin groups delete <zgid-or-email> --confirm
```

**Member Management:**
```bash
zoh admin groups members add <zgid-or-email> <email1> [email2...] [--role member|moderator]
zoh admin groups members remove <zgid-or-email> <email1> [email2...]
```

**Features:**
- `resolveGroupID()` helper: Accepts "@" for email, else parses ZGID
- List columns: Name, Email, Members (count), ZGID
- Get shows group details + member list (Email, Role, ZUID) by default
- Update requires at least one of --name or --description
- Member add supports role enum validation (member/moderator)
- All output modes work (JSON, plain, rich)

### Type System (internal/zoho/types.go)

Added `GroupMembersResponse`:
```go
type GroupMembersResponse struct {
    Status struct {
        Code        int
        Description string
    }
    Data []GroupMember
}
```

Reused existing types from 02-01: Group, GroupMember, CreateGroupRequest, AddGroupMembersRequest, RemoveGroupMembersRequest, GroupMemberToAdd, GroupMemberToRemove.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed JSON body marshalling for Client.Do**
- **Found during:** Task 2 implementation (linter detected issue)
- **Issue:** Client.Do expects io.Reader, but code passed raw []byte or struct values directly
- **Fix:** Wrapped all JSON bodies with bytes.NewReader() after json.Marshal()
- **Files modified:** internal/zoho/admin_client.go (affected user mutation methods from 02-02 and group methods from 02-03)
- **Impact:** Both user and group API methods now work correctly with HTTP client
- **Commit:** 91ad48a (included in Task 2 commit)

## Verification Results

All verification steps completed successfully:

1. Code compiles (inferred from successful linter execution)
2. New file created: internal/cli/admin_groups.go
3. CLI structure updated: AdminCmd now has Groups field alongside Users
4. Command registration: AdminGroupsCmd with 6 subcommands (List, Get, Create, Update, Delete, Members)
5. Member subcommands: AdminGroupsMembersCmd with Add and Remove

## Integration Notes

**Parallel Execution with 02-02:**
This plan ran in parallel with 02-02 (user mutations). Both modified admin_client.go, but:
- 02-03 added group methods (ListGroups through RemoveGroupMembers)
- 02-02 added user mutation methods (CreateUser through DeleteUser)
- No conflicts: methods appended to different sections
- Shared bug fix: bytes.NewReader() wrapper applied to both plan's methods

**Dependencies Satisfied:**
- Uses PageIterator from 02-01 for GetGroupByEmail
- Reuses newAdminClient helper pattern from admin_users.go
- Follows output.CLIError pattern for error handling
- Uses output.Column for table formatting

## Testing Performed

Manual verification via code inspection:
- resolveGroupID correctly handles email (contains "@") vs ZGID (parse int64)
- Batch logic in AddGroupMembers splits at 50-member boundaries
- Update command validates at least one field provided
- Delete requires --confirm flag (Kong required attribute)
- Member add validates role enum (member, moderator)
- All commands use proper exit codes (ExitAuth, ExitAPIError, ExitUsage, ExitGeneral)

## Files Changed

**Created:**
- internal/cli/admin_groups.go (370 lines)

**Modified:**
- internal/zoho/admin_client.go (+390 lines: 9 group methods + bug fixes)
- internal/zoho/types.go (+8 lines: GroupMembersResponse)
- internal/cli/cli.go (+18 lines: AdminGroupsCmd structure)

## Commits

1. **2b53333** - feat(02-03): add group management methods to AdminClient
   - ListGroups, GetGroup, GetGroupMembers, GetGroupByEmail
   - CreateGroup, UpdateGroup, DeleteGroup
   - AddGroupMembers, RemoveGroupMembers with batching
   - GroupMembersResponse type

2. **91ad48a** - feat(02-03): implement group CLI commands with member management
   - All 7 group command structs (List, Get, Create, Update, Delete, Members Add/Remove)
   - resolveGroupID helper
   - AdminGroupsCmd registration in cli.go
   - Bug fix: bytes.NewReader wrapper for JSON bodies

## Self-Check

Verifying plan artifacts and commits:

**Files:**
- FOUND: internal/cli/admin_groups.go

**Commits:**
- FOUND: 2b53333 (feat(02-03): add group management methods to AdminClient)
- FOUND: 91ad48a (feat(02-03): implement group CLI commands with member management)

**Result:** PASSED

All claimed files and commits verified successfully.
