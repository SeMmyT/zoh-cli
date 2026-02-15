package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/SeMmyT/zoh/internal/auth"
	"github.com/SeMmyT/zoh/internal/config"
	"github.com/SeMmyT/zoh/internal/output"
	"github.com/SeMmyT/zoh/internal/secrets"
	"github.com/SeMmyT/zoh/internal/zoho"
)

// newMailAdminClient creates a MailAdminClient with token cache
func newMailAdminClient(cfg *config.Config) (*zoho.MailAdminClient, error) {
	store, err := secrets.NewStore()
	if err != nil {
		return nil, &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize secrets store: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	tokenCache, err := auth.NewTokenCache(cfg, store)
	if err != nil {
		return nil, &output.CLIError{
			Message:  fmt.Sprintf("Failed to initialize token cache: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	mac, err := zoho.NewMailAdminClient(cfg, tokenCache)
	if err != nil {
		// Check if it's an authentication error
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
			return nil, &output.CLIError{
				Message:  fmt.Sprintf("Authentication failed: %v\n\nRun: zoh auth login", err),
				ExitCode: output.ExitAuth,
			}
		}
		return nil, &output.CLIError{
			Message:  fmt.Sprintf("Failed to create mail admin client: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	return mac, nil
}

// DeliveryLogRow is a display struct for delivery log output
type DeliveryLogRow struct {
	Subject      string
	From         string
	To           string
	Status       string
	SentTime     string
	DeliveryTime string
}

// SpamCategoryRow is a display struct for spam category list
type SpamCategoryRow struct {
	Name     string
	APIValue string
}

// MailAdminRetentionGetCmd retrieves retention policy settings
type MailAdminRetentionGetCmd struct{}

// Run executes the get retention policy command
func (cmd *MailAdminRetentionGetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	mac, err := newMailAdminClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	rawPolicy, err := mac.GetRetentionPolicy(ctx)
	if err != nil {
		// If API doesn't exist, provide informative message
		fmt.Fprintf(os.Stderr, "Warning: Retention policy API not available. Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Please use Zoho Admin Console to manage retention policies.\n")
		return &output.CLIError{
			Message:  "Retention policy API unavailable",
			ExitCode: output.ExitAPIError,
		}
	}

	// Pretty print the raw JSON response
	var formatted interface{}
	if err := json.Unmarshal(rawPolicy, &formatted); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to parse retention policy: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	prettyJSON, err := json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to format retention policy: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	fmt.Println(string(prettyJSON))
	return nil
}

// MailAdminSpamGetCmd retrieves spam settings for a category
type MailAdminSpamGetCmd struct {
	Category string `help:"Spam category (e.g., allowlist-email, blocklist-domain)" required:""`
}

// Run executes the get spam settings command
func (cmd *MailAdminSpamGetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	// Validate category
	category, ok := zoho.SpamCategoryMap[cmd.Category]
	if !ok {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid category: %s. Run 'zoh mail admin spam categories' to see valid options.", cmd.Category),
			ExitCode: output.ExitUsage,
		}
	}

	mac, err := newMailAdminClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	values, err := mac.GetSpamSettings(ctx, category)
	if err != nil {
		// If GET not supported, provide informative message
		fmt.Fprintf(os.Stderr, "Warning: Spam settings GET API may not be supported. Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Use 'zoh mail admin spam update' to manage spam settings.\n")
		return &output.CLIError{
			Message:  "Failed to retrieve spam settings",
			ExitCode: output.ExitAPIError,
		}
	}

	if len(values) == 0 {
		fmt.Fprintf(os.Stderr, "No entries found for category: %s\n", cmd.Category)
		return nil
	}

	// Display as simple list
	type ValueRow struct {
		Value string
	}

	rows := make([]ValueRow, len(values))
	for i, v := range values {
		rows[i] = ValueRow{Value: v}
	}

	columns := []output.Column{
		{Name: "Value", Key: "Value"},
	}

	return fp.Formatter.PrintList(rows, columns)
}

// MailAdminSpamUpdateCmd updates spam settings for a category
type MailAdminSpamUpdateCmd struct {
	Category string   `help:"Spam category (e.g., allowlist-email, blocklist-domain)" required:""`
	Values   []string `help:"Email addresses, domains, or IPs to add (repeatable)" required:""`
}

// Run executes the update spam list command
func (cmd *MailAdminSpamUpdateCmd) Run(cfg *config.Config, globals *Globals) error {
	// Validate category
	category, ok := zoho.SpamCategoryMap[cmd.Category]
	if !ok {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid category: %s. Run 'zoh mail admin spam categories' to see valid options.", cmd.Category),
			ExitCode: output.ExitUsage,
		}
	}

	if len(cmd.Values) == 0 {
		return &output.CLIError{
			Message:  "At least one value must be provided",
			ExitCode: output.ExitUsage,
		}
	}

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would update %s: add %d address(es)\n", cmd.Category, len(cmd.Values))
		return nil
	}

	mac, err := newMailAdminClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := mac.UpdateSpamList(ctx, category, cmd.Values); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to update spam list: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Updated %s with %d entries\n", cmd.Category, len(cmd.Values))
	return nil
}

// MailAdminSpamCategoriesCmd lists all available spam categories
type MailAdminSpamCategoriesCmd struct{}

// Run executes the list spam categories command
func (cmd *MailAdminSpamCategoriesCmd) Run(fp *FormatterProvider) error {
	// Convert map to sorted slice for consistent display
	categories := make([]SpamCategoryRow, 0, len(zoho.SpamCategoryMap))
	for name, apiValue := range zoho.SpamCategoryMap {
		categories = append(categories, SpamCategoryRow{
			Name:     name,
			APIValue: string(apiValue),
		})
	}

	// Sort by name for better readability
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	columns := []output.Column{
		{Name: "CLI Name", Key: "Name"},
		{Name: "API Value", Key: "APIValue"},
	}

	return fp.Formatter.PrintList(categories, columns)
}

// MailAdminLogsCmd retrieves delivery logs
type MailAdminLogsCmd struct {
	Limit int `help:"Maximum logs to show" default:"50"`
	Start int `help:"Starting offset" default:"0"`
}

// Run executes the get delivery logs command
func (cmd *MailAdminLogsCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	mac, err := newMailAdminClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	logs, err := mac.GetDeliveryLogs(ctx, cmd.Start, cmd.Limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Delivery logs API may have limitations. Error: %v\n", err)
		return &output.CLIError{
			Message:  "Failed to retrieve delivery logs",
			ExitCode: output.ExitAPIError,
		}
	}

	if len(logs) == 0 {
		fmt.Fprintf(os.Stderr, "No delivery logs found\n")
		return nil
	}

	// Convert to display struct
	rows := make([]DeliveryLogRow, len(logs))
	for i, log := range logs {
		rows[i] = DeliveryLogRow{
			Subject:      log.Subject,
			From:         log.FromAddress,
			To:           log.ToAddress,
			Status:       log.Status,
			SentTime:     log.SentTime,
			DeliveryTime: log.DeliveryTime,
		}
	}

	columns := []output.Column{
		{Name: "Subject", Key: "Subject"},
		{Name: "From", Key: "From"},
		{Name: "To", Key: "To"},
		{Name: "Status", Key: "Status"},
		{Name: "Sent", Key: "SentTime"},
		{Name: "Delivered", Key: "DeliveryTime"},
	}

	return fp.Formatter.PrintList(rows, columns)
}
