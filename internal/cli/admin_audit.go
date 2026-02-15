package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/zoho"
)

// parseDate parses a date string in either YYYY-MM-DD or RFC3339 format
func parseDate(s string, endOfDay bool) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Try date-only format (YYYY-MM-DD)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		if endOfDay {
			// Set to 23:59:59 UTC for end-of-day
			return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC), nil
		}
		// Set to 00:00:00 UTC for start-of-day
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: use YYYY-MM-DD or RFC3339")
}

// AdminAuditLogsCmd shows admin action audit logs
type AdminAuditLogsCmd struct {
	From   string `help:"Start date (YYYY-MM-DD or RFC3339)" required:""`
	To     string `help:"End date (YYYY-MM-DD or RFC3339)" required:""`
	Search string `help:"Filter by category, performer, or operation"`
	Limit  int    `help:"Results per page" default:"100"`
}

// Run executes the audit logs command
func (cmd *AdminAuditLogsCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	// Parse dates
	fromTime, err := parseDate(cmd.From, false)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --from date: %v", err),
			ExitCode: output.ExitUsage,
		}
	}

	toTime, err := parseDate(cmd.To, true)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --to date: %v", err),
			ExitCode: output.ExitUsage,
		}
	}

	ctx := context.Background()

	// Fetch audit logs
	logs, err := adminClient.GetAuditLogs(ctx, fromTime, toTime, cmd.Search, cmd.Limit)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch audit logs: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Transform logs for display
	type displayLog struct {
		Time        string
		Category    string
		Operation   string
		PerformedBy string
		Target      string
		IP          string
	}

	displayLogs := make([]displayLog, len(logs))
	for i, log := range logs {
		displayLogs[i] = displayLog{
			Time:        zoho.FormatMillisTimestamp(log.RequestTime),
			Category:    log.Category,
			Operation:   log.Operation,
			PerformedBy: log.PerformedBy,
			Target:      log.PerformedOn,
			IP:          log.ClientIP,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Time", Key: "Time"},
		{Name: "Category", Key: "Category"},
		{Name: "Operation", Key: "Operation"},
		{Name: "Performed By", Key: "PerformedBy"},
		{Name: "Target", Key: "Target"},
		{Name: "IP", Key: "IP"},
	}

	return fp.Formatter.PrintList(displayLogs, columns)
}

// AdminAuditLoginHistoryCmd shows login history logs
type AdminAuditLoginHistoryCmd struct {
	From  string `help:"Start date (YYYY-MM-DD or RFC3339)" required:""`
	To    string `help:"End date (YYYY-MM-DD or RFC3339)" required:""`
	Mode  string `help:"Activity type filter" default:"loginActivity" enum:"loginActivity,failedLoginActivity,protocolLoginActivity,failedProtocolLoginActivity"`
	Limit int    `help:"Batch size per page (Note: 90-day retention limit)" default:"100"`
}

// Run executes the login history command
func (cmd *AdminAuditLoginHistoryCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	// Parse dates
	fromTime, err := parseDate(cmd.From, false)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --from date: %v", err),
			ExitCode: output.ExitUsage,
		}
	}

	toTime, err := parseDate(cmd.To, true)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --to date: %v", err),
			ExitCode: output.ExitUsage,
		}
	}

	ctx := context.Background()

	// Fetch login history
	entries, err := adminClient.GetLoginHistory(ctx, cmd.Mode, fromTime, toTime, cmd.Limit)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch login history: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Transform entries for display
	type displayEntry struct {
		Time       string
		Email      string
		IP         string
		Status     string
		AccessType string
		Client     string
	}

	displayEntries := make([]displayEntry, len(entries))
	for i, entry := range entries {
		displayEntries[i] = displayEntry{
			Time:       zoho.FormatMillisTimestamp(entry.LoginTime),
			Email:      entry.EmailAddress,
			IP:         entry.IPAddress,
			Status:     entry.Status,
			AccessType: entry.AccessType,
			Client:     entry.ClientInfo,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Time", Key: "Time"},
		{Name: "Email", Key: "Email"},
		{Name: "IP", Key: "IP"},
		{Name: "Status", Key: "Status"},
		{Name: "Access Type", Key: "AccessType"},
		{Name: "Client", Key: "Client"},
	}

	return fp.Formatter.PrintList(displayEntries, columns)
}

// AdminAuditSMTPLogsCmd shows SMTP transaction logs
type AdminAuditSMTPLogsCmd struct {
	From     string `help:"Start date (YYYY-MM-DD or RFC3339)" required:""`
	To       string `help:"End date (YYYY-MM-DD or RFC3339)" required:""`
	SearchBy string `help:"Search criteria field" enum:"messageId,fromAddr,toAddr," default:""`
	Search   string `help:"Search value (requires --search-by)"`
	Limit    int    `help:"Results per page" default:"100"`
}

// Run executes the SMTP logs command
func (cmd *AdminAuditSMTPLogsCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	// Validate search parameters
	if cmd.Search != "" && cmd.SearchBy == "" {
		return &output.CLIError{
			Message:  "--search requires --search-by to be set",
			ExitCode: output.ExitUsage,
		}
	}

	// Parse dates
	fromTime, err := parseDate(cmd.From, false)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --from date: %v", err),
			ExitCode: output.ExitUsage,
		}
	}

	toTime, err := parseDate(cmd.To, true)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --to date: %v", err),
			ExitCode: output.ExitUsage,
		}
	}

	ctx := context.Background()

	// Fetch SMTP logs
	entries, err := adminClient.GetSMTPLogs(ctx, fromTime, toTime, cmd.SearchBy, cmd.Search, cmd.Limit)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch SMTP logs: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Transform entries for display
	type displayEntry struct {
		Time      string
		From      string
		To        string
		Subject   string
		Status    string
		MessageID string
	}

	displayEntries := make([]displayEntry, len(entries))
	for i, entry := range entries {
		displayEntries[i] = displayEntry{
			Time:      zoho.FormatMillisTimestamp(entry.Timestamp),
			From:      entry.FromAddress,
			To:        strings.Join(entry.ToAddresses, ", "),
			Subject:   entry.Subject,
			Status:    entry.Status,
			MessageID: entry.MessageID,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Time", Key: "Time"},
		{Name: "From", Key: "From"},
		{Name: "To", Key: "To"},
		{Name: "Subject", Key: "Subject"},
		{Name: "Status", Key: "Status"},
		{Name: "Message ID", Key: "MessageID"},
	}

	return fp.Formatter.PrintList(displayEntries, columns)
}

// AdminAuditSessionsCmd provides information about active sessions
type AdminAuditSessionsCmd struct{}

// Run executes the sessions command
func (cmd *AdminAuditSessionsCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	fmt.Fprintln(os.Stderr, "Active session listing is not available via the Zoho Mail API.")
	fmt.Fprintln(os.Stderr, "View active sessions in the Zoho Admin Console:")
	fmt.Fprintln(os.Stderr, "  https://mailadmin.zoho.com → Dashboard → Active Sessions")
	return nil
}

// AdminAuditSecurityCmd provides information about security policy settings
type AdminAuditSecurityCmd struct{}

// Run executes the security command
func (cmd *AdminAuditSecurityCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	fmt.Fprintln(os.Stderr, "Security policy settings (2FA, password policies) are not available via the Zoho Mail API.")
	fmt.Fprintln(os.Stderr, "Manage security settings in the Zoho Admin Console:")
	fmt.Fprintln(os.Stderr, "  https://mailadmin.zoho.com → Security & Compliance")
	return nil
}
