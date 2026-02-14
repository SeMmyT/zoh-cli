# Feature Landscape

**Domain:** Zoho Mail + Admin CLI tool ("zoh")
**Researched:** 2026-02-14

## API Coverage Map

Before categorizing features, this map shows what Zoho Mail's REST API actually exposes vs what the web UI offers. This drives every feasibility assessment below.

| Web UI Capability | API Available? | API Category | Confidence |
|---|---|---|---|
| Organization details & settings | YES | Organization API | HIGH |
| Subscription/storage management | YES | Organization API | HIGH |
| Spam allowlists/blocklists | YES | Organization API (antispam) | HIGH |
| IP whitelisting | YES | Organization API (allowedIps) | HIGH |
| Domain add/list/verify | YES | Domain API | HIGH |
| Domain DKIM setup | YES | Domain API | HIGH |
| Domain catch-all/notifications | YES | Domain API | HIGH |
| User CRUD (add/list/remove/update) | YES | Users API | HIGH |
| User role changes | YES | Users API | HIGH |
| User password reset | YES | Users API | HIGH |
| User email aliases | YES | Users API | HIGH |
| User protocol toggles (IMAP/POP/SMTP/ActiveSync) | YES | Users API | HIGH |
| User 2FA preference | YES | Users API | HIGH |
| User enable/disable | YES | Users API | HIGH |
| Group CRUD | YES | Groups API | HIGH |
| Group member/role management | YES | Groups API | HIGH |
| Group moderation queue | YES | Groups API | HIGH |
| Mail policies (email/account/access/forward restrictions) | YES | Mail Policy API | HIGH |
| Policy assignment to users/groups | YES | Mail Policy API | HIGH |
| Account settings (display name, reply-to) | YES | Accounts API | HIGH |
| Email forwarding setup | YES | Accounts API | HIGH |
| Vacation replies | YES | Accounts API | HIGH |
| Signatures | YES | Signatures API | HIGH |
| Folder management | YES | Folders API | HIGH |
| Label management | YES | Labels API | HIGH |
| Send/receive/search email | YES | Messages API | HIGH |
| Attachments (upload/download) | YES | Messages API | HIGH |
| Thread management (flag/move/label) | YES | Threads API | HIGH |
| Login history logs | YES | Logs API | HIGH |
| Audit activity logs | YES | Logs API | HIGH |
| SMTP logs | YES | Logs API | HIGH |
| Tasks (CRUD, subtasks, projects) | YES | Tasks API | HIGH |
| Notes (CRUD, books) | YES | Notes API | HIGH |
| Bookmarks | YES | Bookmarks API | HIGH |
| **Incoming mail rules/filters** | **NO** | Not in API | HIGH |
| **Outgoing mail rules/filters** | **NO** | Not in API | HIGH |
| **User-level inbox filters** | **NO** | Not in API | HIGH |
| **eDiscovery / retention config** | **NO** | Not in API | HIGH |
| **eDiscovery search/export** | **NO** | Not in API | MEDIUM |
| **Email routing (advanced)** | **NO** | Not in API | MEDIUM |
| **Password policies (org-level)** | **NO** | Not in API | MEDIUM |
| **SAML/SSO configuration** | **NO** | Not in API | MEDIUM |
| **Custom admin roles** | **NO** | Not in API | LOW |
| **Device management** | **NO** | Not in API | LOW |
| **Migration tools** | **NO** | Not in API | LOW |

**Key insight:** The Zoho Mail API is broad but has notable gaps around mail filtering rules, eDiscovery, advanced routing, and SSO configuration. These are web-UI-only features. The CLI must acknowledge these gaps rather than try to work around them.

---

## Table Stakes

Features users expect from a Zoho admin/mail CLI. Missing any of these = "why would I use this?"

### Admin: User Management

| Feature | Why Expected | Complexity | API Support | Notes |
|---------|-------------|------------|-------------|-------|
| List all users | Core admin task, most common operation | Low | `GET /api/organization/{zoid}/accounts/` | Include role, status, storage usage |
| Get user details | Look up a specific user quickly | Low | `GET /api/organization/{zoid}/accounts/{zuid}` | Show all fields: aliases, protocols, 2FA |
| Add user | Onboarding workflow | Low | `POST /api/organization/{zoid}/accounts/` | Needs first/last name, email, password |
| Remove user | Offboarding workflow | Low | `DELETE /api/organization/{zoid}/accounts` | Require confirmation flag |
| Update user role | Permission management | Low | `PUT /api/organization/{zoid}/accounts` | Admin/user role toggle |
| Reset user password | Most common support request | Low | `PUT /api/organization/{zoid}/accounts` | Generate or set password |
| Enable/disable user account | Suspend without deleting | Low | `PUT /api/organization/{zoid}/accounts` | Separate mail vs full account |
| Manage email aliases | Common identity management | Low | `PUT /api/organization/{zoid}/accounts` | Add/remove alias operations |
| Toggle protocols (IMAP/POP/SMTP/ActiveSync) | Security hardening | Low | `PUT /api/organization/{zoid}/accounts` | Per-user protocol control |
| Toggle 2FA preference | Security enforcement | Low | `PUT /api/organization/{zoid}/accounts` | Enable/disable per user |

### Admin: Group Management

| Feature | Why Expected | Complexity | API Support | Notes |
|---------|-------------|------------|-------------|-------|
| List groups | See what distribution lists exist | Low | `GET /api/organization/{zoid}/groups` | Include member counts |
| Create group | Set up team email addresses | Low | `POST /api/organization/{zoid}/groups` | Name + initial members |
| Delete group | Clean up unused lists | Low | `DELETE /api/organization/{zoid}/groups/{zgid}` | Require confirmation |
| Add/remove members | Most common group operation | Low | `PUT /api/organization/{zoid}/groups/{zgid}` | Bulk add support important |
| Update group roles | Set moderators/owners | Low | `PUT /api/organization/{zoid}/groups/{zgid}` | Role assignment |
| View moderation queue | Manage pending messages | Med | `GET /api/organization/{zoid}/groups/{zgid}/messages` | List + approve/reject |
| Moderate messages | Approve/reject pending emails | Med | `PUT /api/organization/{zoid}/groups/{zgid}/messages` | Approve/reject action |

### Admin: Domain Management

| Feature | Why Expected | Complexity | API Support | Notes |
|---------|-------------|------------|-------------|-------|
| List domains | See configured domains | Low | `GET /api/organization/{zoid}/domains` | Show verification status |
| Get domain details | Check DNS/config status | Low | `GET /api/organization/{zoid}/domains/{domain}` | Verification, DKIM, SPF status |
| Add domain | Expand organization | Med | `POST /api/organization/{zoid}/domains` | Triggers verification flow |
| Verify domain | Complete domain setup | Med | `PUT /api/organization/{zoid}/domains/{domain}` | Different verification methods |
| Configure DKIM | Email authentication | Med | `PUT /api/organization/{zoid}/domains/{domain}` | Generate/validate DKIM keys |
| Set primary domain | Organization identity | Low | `PUT /api/organization/{zoid}/domains/{domain}` | Change primary domain |
| Manage catch-all address | Handle unmatched emails | Low | `PUT /api/organization/{zoid}/domains/{domain}` | Set/remove catch-all |

### Admin: Organization & Security

| Feature | Why Expected | Complexity | API Support | Notes |
|---------|-------------|------------|-------------|-------|
| Get org details | See org configuration | Low | `GET /api/organization/{zoid}` | Name, plan, user/group counts |
| View subscription/storage | Capacity planning | Low | `GET /api/organization/{zoid}/storage` | Org-wide + per-user storage |
| Manage spam allow/block lists | Anti-spam admin | Low | Organization API antispam endpoints | Add/remove/list entries |
| Manage IP whitelist | Access control | Low | Organization API allowedIps endpoints | Add/remove/list IPs |
| View login history | Security monitoring | Low | `GET /api/organization/{zoid}/accounts/reports/loginHistory` | Per-user login records |
| View audit logs | Compliance/investigation | Low | `GET /api/organization/{zoid}/activity` | Admin action tracking |
| View SMTP logs | Mail delivery debugging | Low | `GET /api/organization/{zoid}/smtplogs` | Send/receive troubleshooting |

### Mail: Read & Send

| Feature | Why Expected | Complexity | API Support | Notes |
|---------|-------------|------------|-------------|-------|
| List inbox messages | Core email reading | Low | `GET /api/accounts/{id}/messages/view` | Paginated, folder-based |
| Read message content | View email body | Low | `GET /api/accounts/{id}/folders/{fid}/messages/{mid}/content` | HTML/text content |
| Search emails | Find specific messages | Med | `GET /api/accounts/{id}/messages/search` | Query parameter-based |
| Send email | Core email sending | Med | `POST /api/accounts/{id}/messages` | To, CC, BCC, subject, body |
| Reply to email | Conversation continuation | Med | `POST /api/accounts/{id}/messages` | Threading handled |
| Send with attachments | File sharing | Med | `POST /api/accounts/{id}/messages` | Upload then attach flow |
| Download attachments | Receive files | Med | Messages API attachment endpoints | By attachment ID |
| Mark read/unread | Inbox management | Low | `PUT /api/accounts/{id}/updatemessage` | Bulk support |
| Move messages | Organization | Low | `PUT /api/accounts/{id}/updatemessage` | Between folders |
| Flag messages | Priority marking | Low | `PUT /api/accounts/{id}/updatemessage` | Star/unstar |
| Delete messages | Cleanup | Low | `DELETE` Messages API | Soft delete to trash |
| Archive messages | Long-term storage | Low | `PUT /api/accounts/{id}/updatemessage` | Archive/unarchive |

### Mail: Account Settings

| Feature | Why Expected | Complexity | API Support | Notes |
|---------|-------------|------------|-------------|-------|
| Manage signatures | Email branding | Low | Signatures API (full CRUD) | Create/update/delete/get |
| Set vacation reply | Out-of-office | Low | Accounts API vacation endpoints | Enable/disable + message |
| Update display name | Identity management | Low | `PUT /api/accounts/{id}` | Per-account |
| Set reply-to address | Routing control | Low | `PUT /api/accounts/{id}` | Verify + set |
| Configure forwarding | Mail routing | Med | Accounts API forwarding endpoints | Add/verify/enable/disable/remove |

### Mail: Organization (Folders & Labels)

| Feature | Why Expected | Complexity | API Support | Notes |
|---------|-------------|------------|-------------|-------|
| List/create/delete folders | Mail organization | Low | Folders API (full CRUD) | Nested folder support |
| Rename/move folders | Reorganization | Low | Folders API PUT | Standard operations |
| List/create/delete labels | Tagging system | Low | Labels API (full CRUD) | Color-coded labels |
| Apply/remove labels from messages | Categorization | Low | Messages API updatemessage | Bulk support |

---

## Differentiators

Features that would set "zoh" apart from just being a thin API wrapper. These are not expected, but would make power users love the tool.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Bulk user operations** | Process CSV/JSON files for mass onboarding/offboarding | Med | `zoh admin users import users.csv` -- GAM's killer feature is bulk ops |
| **User audit report** | Single command showing user's full state: aliases, protocols, 2FA, storage, groups, login history | Med | Aggregates multiple API calls into one view |
| **Interactive email compose** | TUI-based email composition with editor integration (`$EDITOR`) | High | Like `git commit` opening your editor for body text |
| **Output format flexibility** | Table, JSON, CSV, TSV output for every command | Med | `--output json` / `--output csv` -- follows gogcli pattern |
| **Pipe-friendly design** | Every list command outputs IDs that pipe into action commands | Med | `zoh mail list --unread --ids \| xargs zoh mail read` |
| **Smart search** | Combine API search with local filtering/sorting | Med | `zoh mail search "from:boss subject:urgent" --after 2024-01-01` |
| **Policy templates** | Pre-built policy configurations (restrictive, moderate, open) | Med | `zoh admin policy apply --template restrictive --group engineering` |
| **Domain health check** | Single command validating DNS, SPF, DKIM, DMARC, MX for all domains | Med | Fetches domain details and reports configuration issues |
| **Storage reports** | Organization-wide storage usage with top consumers | Low | Aggregates per-user storage data into report |
| **Onboarding workflow** | Single command: create user + add to groups + set aliases + configure protocols | High | `zoh admin onboard --config onboard.yaml` |
| **Offboarding workflow** | Disable user + remove from groups + set forwarding + audit export | High | `zoh admin offboard user@domain.com --forward-to manager@domain.com` |
| **Shell completion** | Bash/Zsh/Fish completions for all commands and arguments | Med | Cobra generates these; add dynamic completions for user/group names |
| **Configuration profiles** | Multiple Zoho org configs, switch between them | Med | `zoh --profile production` vs `zoh --profile staging` |
| **Watching/polling** | Monitor inbox or audit logs in real-time | Med | `zoh mail watch --folder inbox` polling at intervals |
| **Message threading view** | Display email threads as conversation trees | Med | Aggregate thread messages into readable conversation |
| **Mail rule gap notification** | Warn users about features only available in web UI | Low | `zoh admin rules` shows helpful message pointing to web UI |
| **Multi-datacenter awareness** | Auto-detect or configure correct API base URL per datacenter | Low | US/EU/IN/AU/JP/CA/CN/AE/SA support built-in |

---

## Anti-Features

Features to explicitly NOT build. Each has a clear reason.

| Anti-Feature | Why Avoid | What to Do Instead |
|---|---|---|
| **Mail filter/rule management** | No API exists. Would require screen-scraping the web UI, which is fragile, unsupported, and breaks on UI changes. | Print a clear message: "Mail rules are not available via the Zoho Mail API. Use the web console: https://mail.zoho.com/..." |
| **eDiscovery / retention config** | No API exists. This is a compliance-critical feature that should not be hacked together. | Same approach: clear redirect to web UI with direct URL |
| **SSO/SAML configuration** | No API exists. Security-critical config should not be approximated. | Redirect to Zoho Directory / web console |
| **Device management** | No API exists. Security-sensitive feature. | Redirect to web console |
| **Email migration** | No API exists. Zoho provides dedicated migration tools. | Document how to use Zoho's migration tooling |
| **Calendar/contacts management** | Out of scope for v1. Zoho Mail API includes Tasks, Notes, Bookmarks APIs but these dilute the core admin+mail focus. | Defer to v2 if there's demand. Keep scope tight. |
| **Tasks/Notes/Bookmarks** | API exists but out of scope. These are collaboration features, not admin or mail operations. | Could be a separate `zoh collab` subcommand tree in v2 |
| **Full webmail replacement** | A CLI will never replace the full Zoho Mail web experience. Don't try. | Focus on power-user workflows: quick reads, bulk ops, scripting |
| **Bulk email sending** | Zoho explicitly prohibits bulk/burst sending (50-500/hr external limit). Building bulk send tools invites abuse and account suspension. | Enforce rate limits, warn users, suggest ZeptoMail for transactional email |
| **HTML email rendering in terminal** | Rendering HTML emails in a terminal is a poor experience. | Show plain text, offer `--html` flag to open in browser, or pipe to `w3m`/`lynx` |
| **Password policy configuration** | No API exists. Org-level password policies are web-UI only. | Redirect to admin console |
| **Advanced email routing** | No API for multi-server routing configuration. | Redirect to admin console |

---

## Feature Dependencies

```
Authentication (OAuth self-client) → ALL features
  |
  ├── Config (datacenter URL, org ID) → ALL API calls
  |
  ├── Organization API access → Admin features
  |     ├── Org details → Storage reports, subscription info
  |     ├── User CRUD → User management
  |     │     ├── User list → Bulk operations
  |     │     ├── User details → User audit report
  |     │     └── User CRUD → Onboarding/Offboarding workflows
  |     ├── Group CRUD → Group management
  |     │     └── Group members → Moderation queue
  |     ├── Domain management → Domain health check
  |     ├── Mail Policy API → Policy management, templates
  |     ├── Spam management → Allowlist/blocklist
  |     └── Logs API → Audit logs, login history, SMTP logs
  |
  └── Accounts/Messages API access → Mail features
        ├── Account details → Settings management
        │     ├── Signatures → Signature CRUD
        │     ├── Vacation replies → OOO management
        │     └── Forwarding → Forward config
        ├── Folder/Label CRUD → Mail organization
        ├── Message list/search → Reading email
        │     ├── Message content → Read individual emails
        │     └── Attachments → Download files
        ├── Send message → Email composition
        │     └── Upload attachment → Send with files
        └── Update message → Flag/move/label/archive
```

---

## Rate Limit Considerations

The Zoho Mail API has a **30 requests/minute** rate limit across all endpoints. This is aggressive and directly impacts feature design.

| Impact Area | Constraint | Mitigation |
|---|---|---|
| Bulk user operations | Listing 100+ users then fetching details = easily hits limit | Implement request queuing with backoff; batch where API allows |
| User audit report | Aggregates user details + groups + login history = 3+ calls per user | Cache aggressively; parallelize within rate limit |
| Storage reports | Per-user storage fetch for org of 50+ users = 50+ calls | Paginate and queue; offer `--slow` flag for large orgs |
| Email search + read | Search returns IDs, then fetch content = 2 calls per message | Lazy-load content only when needed |
| Onboarding workflow | Create user + add to groups + set aliases = 5+ calls per user | Sequential execution with progress bar |

**Design implication:** Every command that makes multiple API calls should display progress, respect rate limits automatically, and support `--dry-run` to preview operations before executing.

---

## OAuth Scope Requirements by Feature Area

| Feature Area | Required Scopes |
|---|---|
| Organization management | `ZohoMail.partner.organization`, `ZohoMail.organization.accounts` |
| Subscription/storage | `ZohoMail.organization.subscriptions` |
| Spam management | `ZohoMail.organization.spam` |
| Domain management | `ZohoMail.organization.domains` |
| Group management | `ZohoMail.organization.groups` |
| Mail policies | `ZohoMail.organization.policy` |
| Account settings | `ZohoMail.accounts` |
| Folder management | `ZohoMail.folders` |
| Label management | `ZohoMail.tags` |
| Email read/send | `ZohoMail.messages` |
| Audit/logs | `ZohoMail.organization.audit` |
| Tasks | `ZohoMail.tasks` |
| Notes | `ZohoMail.notes` |
| Bookmarks | `ZohoMail.links` |

**Design implication:** The CLI should request only the scopes needed for the operations the user wants. Consider a `zoh auth setup` wizard that asks which features they want and requests minimal scopes. At minimum, separate "admin" scopes from "mail user" scopes.

---

## MVP Recommendation

### Phase 1: Foundation + Core Admin (build first)

Prioritize these because admin operations are the primary pain point (Zoho's web UI is slow for repetitive admin tasks) and have the simplest API surface:

1. **OAuth authentication** (self-client flow with refresh token storage)
2. **Multi-datacenter configuration** (9 regions to support)
3. **User management** (list, add, remove, update, enable/disable)
4. **Group management** (list, create, delete, add/remove members)
5. **Output format support** (table + JSON + CSV from day 1)

### Phase 2: Domain + Security Admin

6. **Domain management** (list, add, verify, DKIM, catch-all)
7. **Organization details** (org info, subscription, storage)
8. **Spam management** (allowlist/blocklist)
9. **IP whitelist management**
10. **Audit/login/SMTP logs**

### Phase 3: Mail Operations

11. **Read email** (list, read, search)
12. **Send email** (compose, reply, attachments)
13. **Folder/label management**
14. **Message management** (flag, move, archive, delete)
15. **Signature management**
16. **Vacation replies**
17. **Forwarding configuration**

### Phase 4: Power User Features

18. **Bulk operations** (CSV import for user management)
19. **Policy management** (create, assign, templates)
20. **Onboarding/offboarding workflows**
21. **Domain health check**
22. **Storage reports**
23. **User audit reports**
24. **Shell completions**

### Defer to v2+

- Tasks, Notes, Bookmarks (out of scope for admin+mail focus)
- Interactive TUI (nice-to-have, not core value)
- Calendar/contacts (separate product concern)

**Rationale:** Admin operations first because (a) they have the clearest CLI value proposition over web UI (bulk, scriptable, fast), (b) the API surface is simpler (no complex email rendering), and (c) this is the primary pain point the user expressed.

---

## Competitor Feature Comparison

| Feature | GAM (Google) | gogcli (Google) | zoh (Zoho, planned) |
|---------|-------------|----------------|---------------------|
| User CRUD | Yes | No (user-focused) | Yes |
| Group management | Yes | Yes (basic) | Yes |
| Domain management | Yes | No | Yes |
| Bulk operations (CSV) | Yes (core feature) | No | Planned (Phase 4) |
| Read/send email | No | Yes | Yes |
| Email search | No | Yes | Yes |
| Mail filters/rules | Yes (Google API supports) | Yes | No (Zoho API gap) |
| Security policies | Yes | No | Partial (Mail Policy API) |
| Audit logs | Yes | No | Yes |
| Output formats | CSV primarily | JSON, table, TSV | JSON, table, CSV (planned) |
| Shell completions | No | Yes | Planned |
| Multi-account | Yes (delegation) | Yes (profiles) | Planned (profiles) |
| OAuth handling | Built-in | Built-in (keyring) | Planned (self-client) |
| Language | Python | Go | Go |

**Key takeaway:** "zoh" uniquely combines admin operations (like GAM) with mail operations (like gogcli) in a single tool. Neither competitor serves the Zoho ecosystem. The closest analog is GAM for Google Workspace, which is the gold standard for admin CLIs -- bulk operations and scriptability are what make it indispensable.

---

## Sources

### Official Zoho Documentation (HIGH confidence)
- [Zoho Mail API Index](https://www.zoho.com/mail/help/api/) - Complete endpoint inventory
- [Zoho Mail API Overview](https://www.zoho.com/mail/help/api/overview.html) - Architecture and categories
- [Organization API](https://www.zoho.com/mail/help/api/organization-api.html) - Org management endpoints
- [Users API](https://www.zoho.com/mail/help/api/users-api.html) - User management endpoints
- [Account API](https://www.zoho.com/mail/help/api/account-api.html) - Account settings endpoints
- [Email Messages API](https://www.zoho.com/mail/help/api/email-api.html) - Message operations
- [Mail Policy API](https://www.zoho.com/mail/help/api/mail-policy-api.html) - Policy management
- [OAuth Self-Client Overview](https://www.zoho.com/accounts/protocol/oauth/self-client/overview.html) - Auth flow
- [Self-Client Auth Code Flow](https://www.zoho.com/accounts/protocol/oauth/self-client/authorization-code-flow.html) - Token generation
- [Getting Started](https://www.zoho.com/mail/help/api/getting-started-with-api.html) - Base URLs, headers
- [Admin Console Overview](https://www.zoho.com/mail/help/adminconsole/overview.html) - Full admin feature list
- [Incoming Rules](https://www.zoho.com/mail/help/adminconsole/incoming-rules.html) - Confirmed no API
- [Rates and Limits](https://www.zoho.com/mail/help/adminconsole/rates-and-limits.html) - 30 req/min limit
- [Zoho Mail Features](https://zoho.com/mail/features.html) - Complete feature list

### Competitor Tools (MEDIUM confidence)
- [GAM - Google Workspace CLI](https://github.com/GAM-team/GAM) - Admin CLI reference
- [gogcli - Google Suite CLI](https://github.com/steipete/gogcli) - Mail/productivity CLI reference
- [gogcli.sh](https://gogcli.sh/) - gogcli documentation

### Gap Analysis (MEDIUM confidence)
- [Zoho Mail eDiscovery](https://www.zoho.com/mail/ediscovery.html) - Web UI only, no API confirmed
- [Zoho Directory SCIM](https://www.zoho.com/directory/features/scim-user-provisioning-software.html) - Separate from Mail API
- [Outgoing Rules](https://www.zoho.com/mail/help/adminconsole/outgoing-rules.html) - Web UI only
- [Security Reports](https://www.zoho.com/mail/help/adminconsole/security-reports.html) - Web UI features
