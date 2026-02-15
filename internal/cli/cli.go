package cli

import (
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/semmy-space/zoh/internal/config"
	"github.com/semmy-space/zoh/internal/output"
)

// FormatterProvider wraps the formatter interface for Kong binding
type FormatterProvider struct {
	Formatter output.Formatter
}

// CLI is the root command structure
type CLI struct {
	Globals

	Auth    AuthCmd    `cmd:"" help:"Authentication commands"`
	Config  ConfigCmd  `cmd:"" help:"Configuration commands"`
	Admin   AdminCmd   `cmd:"" help:"Admin operations"`
	Mail    MailCmd    `cmd:"" help:"Mail operations"`

	// Desire-path shortcuts
	Send MailSendComposeCmd `cmd:"" help:"Send an email (shortcut for mail send compose)" hidden:""`
	Ls   LsCmd              `cmd:"" help:"List resources (users, groups, folders, labels)" hidden:""`

	// Shell completion
	Completion CompletionCmd `cmd:"" help:"Shell completion commands"`

	// Introspection
	Schema  SchemaCmd  `cmd:"" help:"Show machine-readable command tree as JSON"`
	Version VersionCmd `cmd:"" help:"Show version information"`
}

// BeforeApply hook runs before any command execution
// It loads config, resolves region, creates formatter, and binds dependencies
func (c *CLI) BeforeApply(ctx *kong.Context) error {
	// Validate flag combinations
	if c.Force && c.DryRun {
		return fmt.Errorf("cannot use --force with --dry-run")
	}
	if c.ResultsOnly && c.ResolvedOutput() != "json" {
		return fmt.Errorf("--results-only requires --output=json")
	}

	// Load config from XDG path (returns defaults if missing)
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Resolve region: CLI flag > config > "us" default
	region := c.Region
	if region == "" && cfg.Region != "" {
		region = cfg.Region
	}
	if region == "" {
		region = "us"
	}
	cfg.Region = region

	// Create output formatter
	var formatter *FormatterProvider
	outputMode := c.ResolvedOutput()
	if outputMode == "json" {
		formatter = &FormatterProvider{
			Formatter: output.NewJSON(c.ResultsOnly),
		}
	} else {
		formatter = &FormatterProvider{
			Formatter: output.New(outputMode),
		}
	}

	// Bind dependencies to kong context
	ctx.Bind(cfg)
	ctx.Bind(formatter)
	ctx.Bind(&c.Globals)
	sp := NewServiceProvider(cfg)
	ctx.Bind(sp)

	return nil
}

// AuthCmd holds authentication subcommands
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Log in to Zoho account"`
	Logout AuthLogoutCmd `cmd:"" help:"Log out and remove stored credentials"`
	List   AuthListCmd   `cmd:"" help:"List stored accounts"`
}

// ConfigCmd holds configuration subcommands
type ConfigCmd struct {
	Get   ConfigGetCmd        `cmd:"" help:"Get a configuration value"`
	Set   ConfigSetCmd        `cmd:"" help:"Set a configuration value"`
	Unset ConfigUnsetCmd      `cmd:"" help:"Remove a configuration value"`
	List  ConfigListConfigCmd `cmd:"" name:"list" help:"List all configuration values"`
	Path  ConfigPathCmd       `cmd:"" help:"Show config file path"`
}

// AdminCmd holds admin subcommands
type AdminCmd struct {
	Users   AdminUsersCmd   `cmd:"" help:"Manage organization users"`
	Groups  AdminGroupsCmd  `cmd:"" help:"Manage organization groups"`
	Domains AdminDomainsCmd `cmd:"" help:"Manage organization domains"`
	Audit   AdminAuditCmd   `cmd:"" help:"View audit logs and security information"`
}

// AdminUsersCmd holds user subcommands
type AdminUsersCmd struct {
	List       AdminUsersListCmd       `cmd:"" help:"List organization users"`
	Get        AdminUsersGetCmd        `cmd:"" help:"Get user details"`
	Create     AdminUsersCreateCmd     `cmd:"" help:"Create a new user"`
	Update     AdminUsersUpdateCmd     `cmd:"" help:"Update user role"`
	Activate   AdminUsersActivateCmd   `cmd:"" help:"Activate a user account"`
	Deactivate AdminUsersDeactivateCmd `cmd:"" help:"Deactivate a user account"`
	Delete     AdminUsersDeleteCmd     `cmd:"" help:"Delete a user permanently"`
}

// AdminGroupsCmd holds group subcommands
type AdminGroupsCmd struct {
	List    AdminGroupsListCmd    `cmd:"" help:"List organization groups"`
	Get     AdminGroupsGetCmd     `cmd:"" help:"Get group details"`
	Create  AdminGroupsCreateCmd  `cmd:"" help:"Create a new group"`
	Update  AdminGroupsUpdateCmd  `cmd:"" help:"Update group settings"`
	Delete  AdminGroupsDeleteCmd  `cmd:"" help:"Delete a group permanently"`
	Members AdminGroupsMembersCmd `cmd:"" help:"Manage group members"`
}

// AdminGroupsMembersCmd holds group member management subcommands
type AdminGroupsMembersCmd struct {
	Add    AdminGroupsMembersAddCmd    `cmd:"" help:"Add members to a group"`
	Remove AdminGroupsMembersRemoveCmd `cmd:"" help:"Remove members from a group"`
}

// AdminDomainsCmd holds domain subcommands
type AdminDomainsCmd struct {
	List   AdminDomainsListCmd   `cmd:"" help:"List all domains"`
	Get    AdminDomainsGetCmd    `cmd:"" help:"Get domain details"`
	Add    AdminDomainsAddCmd    `cmd:"" help:"Add a new domain"`
	Verify AdminDomainsVerifyCmd `cmd:"" help:"Verify domain ownership"`
	Update AdminDomainsUpdateCmd `cmd:"" help:"Update domain settings"`
}

// AdminAuditCmd holds audit and security subcommands
type AdminAuditCmd struct {
	Logs         AdminAuditLogsCmd         `cmd:"" help:"View admin action audit logs"`
	LoginHistory AdminAuditLoginHistoryCmd `cmd:"login-history" help:"View login history (90-day retention)"`
	SMTPLogs     AdminAuditSMTPLogsCmd     `cmd:"smtp-logs" help:"View SMTP transaction logs"`
	Sessions     AdminAuditSessionsCmd     `cmd:"" help:"View active sessions"`
	Security     AdminAuditSecurityCmd     `cmd:"" help:"View security policy settings"`
}

// MailCmd holds mail subcommands
type MailCmd struct {
	Folders     MailFoldersCmd     `cmd:"" help:"Manage mail folders"`
	Labels      MailLabelsCmd      `cmd:"" help:"Manage mail labels"`
	Messages    MailMessagesCmd    `cmd:"" help:"Manage messages"`
	Attachments MailAttachmentsCmd `cmd:"" help:"Manage attachments"`
	Send        MailSendCmd        `cmd:"" help:"Send email messages"`
	Settings    MailSettingsCmd    `cmd:"" help:"Manage mail settings"`
	Admin       MailAdminCmd       `cmd:"" help:"Mail administration operations"`
}

// MailFoldersCmd holds folder subcommands
type MailFoldersCmd struct {
	List MailFoldersListCmd `cmd:"" help:"List all folders"`
}

// MailLabelsCmd holds label subcommands
type MailLabelsCmd struct {
	List MailLabelsListCmd `cmd:"" help:"List all labels"`
}

// MailMessagesCmd holds message subcommands
type MailMessagesCmd struct {
	List   MailMessagesListCmd   `cmd:"" help:"List messages in a folder"`
	Get    MailMessagesGetCmd    `cmd:"" help:"Get full message details"`
	Search MailMessagesSearchCmd `cmd:"" help:"Search messages with query filters"`
	Thread MailMessagesThreadCmd `cmd:"" help:"View all messages in a thread"`
}

// MailAttachmentsCmd holds attachment subcommands
type MailAttachmentsCmd struct {
	List     MailAttachmentsListCmd     `cmd:"" help:"List attachments for a message"`
	Download MailAttachmentsDownloadCmd `cmd:"" help:"Download an attachment"`
}

// MailSendCmd holds send subcommands
type MailSendCmd struct {
	Compose MailSendComposeCmd `cmd:"" help:"Compose and send a new email"`
	Reply   MailSendReplyCmd   `cmd:"" help:"Reply to a message"`
	Forward MailSendForwardCmd `cmd:"" help:"Forward a message"`
}

// MailSettingsCmd holds settings subcommands
type MailSettingsCmd struct {
	Signatures  MailSettingsSignaturesCmd  `cmd:"" help:"Manage email signatures"`
	Vacation    MailSettingsVacationCmd    `cmd:"" help:"Manage vacation auto-reply"`
	DisplayName MailSettingsDisplayNameCmd `cmd:"display-name" help:"Manage account display name"`
	Forwarding  MailSettingsForwardingCmd  `cmd:"" help:"View forwarding settings"`
}

// MailSettingsSignaturesCmd holds signature subcommands
type MailSettingsSignaturesCmd struct {
	List   MailSettingsSignaturesListCmd   `cmd:"" help:"List all email signatures"`
	Create MailSettingsSignaturesCreateCmd `cmd:"" help:"Create a new email signature"`
}

// MailSettingsVacationCmd holds vacation auto-reply subcommands
type MailSettingsVacationCmd struct {
	Get     MailSettingsVacationGetCmd     `cmd:"" help:"View vacation auto-reply settings"`
	Set     MailSettingsVacationSetCmd     `cmd:"" help:"Enable vacation auto-reply"`
	Disable MailSettingsVacationDisableCmd `cmd:"" help:"Disable vacation auto-reply"`
}

// MailSettingsDisplayNameCmd holds display name subcommands
type MailSettingsDisplayNameCmd struct {
	Get MailSettingsDisplayNameGetCmd `cmd:"" help:"View account display name"`
	Set MailSettingsDisplayNameSetCmd `cmd:"" help:"Update account display name"`
}

// MailSettingsForwardingCmd holds forwarding subcommands
type MailSettingsForwardingCmd struct {
	Get MailSettingsForwardingGetCmd `cmd:"" help:"View forwarding settings"`
}

// MailAdminCmd holds mail admin subcommands
type MailAdminCmd struct {
	Retention MailAdminRetentionCmd `cmd:"" help:"Manage retention policies"`
	Spam      MailAdminSpamCmd      `cmd:"" help:"Manage spam filter settings"`
	Logs      MailAdminLogsCmd      `cmd:"" help:"View delivery logs"`
}

// MailAdminRetentionCmd holds retention policy subcommands
type MailAdminRetentionCmd struct {
	Get MailAdminRetentionGetCmd `cmd:"" help:"View retention policy settings"`
}

// MailAdminSpamCmd holds spam filter subcommands
type MailAdminSpamCmd struct {
	Get        MailAdminSpamGetCmd        `cmd:"" help:"View spam settings for a category"`
	Update     MailAdminSpamUpdateCmd     `cmd:"" help:"Update spam allowlist/blocklist"`
	Categories MailAdminSpamCategoriesCmd `cmd:"" help:"List available spam categories"`
}

// VersionCmd shows version information
type VersionCmd struct{}

func (cmd *VersionCmd) Run(ctx *kong.Context) error {
	version := ctx.Model.Vars()["version"]
	println("zoh version " + version)
	return nil
}
