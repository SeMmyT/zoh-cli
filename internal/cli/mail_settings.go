package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/zoho"
)

// SignatureRow is a display struct for signature list output with formatted fields
type SignatureRow struct {
	Name        string
	Position    string
	AssignUsers string
	ID          string
}

// MailSettingsSignaturesListCmd lists all email signatures
type MailSettingsSignaturesListCmd struct{}

// Run executes the list signatures command
func (cmd *MailSettingsSignaturesListCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	signatures, err := mailClient.ListSignatures(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to list signatures: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Convert to display struct with formatted position
	rows := make([]SignatureRow, len(signatures))
	for i, sig := range signatures {
		position := "Below Quoted"
		if sig.Position == 1 {
			position = "Above Quoted"
		}
		rows[i] = SignatureRow{
			Name:        sig.Name,
			Position:    position,
			AssignUsers: sig.AssignUsers,
			ID:          sig.ID,
		}
	}

	columns := []output.Column{
		{Name: "Name", Key: "Name"},
		{Name: "Position", Key: "Position"},
		{Name: "Assigned Users", Key: "AssignUsers"},
		{Name: "ID", Key: "ID"},
	}

	return fp.Formatter.PrintList(rows, columns)
}

// MailSettingsSignaturesCreateCmd creates a new email signature
type MailSettingsSignaturesCreateCmd struct {
	Name        string `help:"Signature name" required:""`
	Content     string `help:"Signature HTML content" required:""`
	Position    int    `help:"Position: 0=below quoted, 1=above quoted" default:"0"`
	AssignUsers string `help:"Comma-separated email addresses to assign signature to" name:"assign-users"`
}

// Run executes the create signature command
func (cmd *MailSettingsSignaturesCreateCmd) Run(sp *ServiceProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would create signature: name=%s\n", cmd.Name)
		return nil
	}

	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()

	sig := &zoho.Signature{
		Name:        cmd.Name,
		Content:     cmd.Content,
		Position:    cmd.Position,
		AssignUsers: cmd.AssignUsers,
	}

	sigID, err := mailClient.AddSignature(ctx, sig)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to create signature: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Created signature %s (ID: %s)\n", cmd.Name, sigID)
	return nil
}

// MailSettingsVacationGetCmd displays vacation auto-reply settings
type MailSettingsVacationGetCmd struct{}

// Run executes the get vacation command
func (cmd *MailSettingsVacationGetCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	accountDetails, err := mailClient.GetAccountDetails(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to get account details: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Check if vacation response exists
	if len(accountDetails.VacationResponse) == 0 || string(accountDetails.VacationResponse) == "null" {
		fmt.Fprintln(os.Stderr, "No vacation reply configured")
		return nil
	}

	// Parse vacation response
	var vacationReply zoho.VacationReply
	if err := json.Unmarshal(accountDetails.VacationResponse, &vacationReply); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to parse vacation response: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Display vacation details
	return fp.Formatter.Print(vacationReply)
}

// MailSettingsVacationSetCmd enables vacation auto-reply
type MailSettingsVacationSetCmd struct {
	From     string `help:"Start date (MM/DD/YYYY HH:MM:SS)" required:""`
	To       string `help:"End date (MM/DD/YYYY HH:MM:SS)" required:""`
	Subject  string `help:"Auto-reply subject" required:""`
	Content  string `help:"Auto-reply message content" required:""`
	Interval int    `help:"Reply interval in minutes" default:"1440"`
	SendTo   string `help:"Send to: all/contacts/noncontacts/org/nonOrgAll" default:"all" enum:"all,contacts,noncontacts,org,nonOrgAll"`
}

// Run executes the set vacation command
func (cmd *MailSettingsVacationSetCmd) Run(sp *ServiceProvider, globals *Globals) error {
	// Validate date format
	dateLayout := "01/02/2006 15:04:05"
	if _, err := time.Parse(dateLayout, cmd.From); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --from date format (expected MM/DD/YYYY HH:MM:SS): %v", err),
			ExitCode: output.ExitUsage,
		}
	}
	if _, err := time.Parse(dateLayout, cmd.To); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Invalid --to date format (expected MM/DD/YYYY HH:MM:SS): %v", err),
			ExitCode: output.ExitUsage,
		}
	}

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would enable vacation reply: subject=%s\n", cmd.Subject)
		return nil
	}

	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()

	vacation := &zoho.VacationReply{
		FromDate:   cmd.From,
		ToDate:     cmd.To,
		SendingInt: cmd.Interval,
		Subject:    cmd.Subject,
		Content:    cmd.Content,
		SendTo:     cmd.SendTo,
	}

	if err := mailClient.AddVacationReply(ctx, vacation); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to enable vacation reply: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Vacation reply enabled (%s to %s)\n", cmd.From, cmd.To)
	return nil
}

// MailSettingsVacationDisableCmd disables vacation auto-reply
type MailSettingsVacationDisableCmd struct{}

// Run executes the disable vacation command
func (cmd *MailSettingsVacationDisableCmd) Run(sp *ServiceProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would disable vacation auto-reply\n")
		return nil
	}

	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := mailClient.DisableVacationReply(ctx); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to disable vacation reply: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintln(os.Stderr, "Vacation reply disabled")
	return nil
}

// MailSettingsDisplayNameGetCmd displays the account display name
type MailSettingsDisplayNameGetCmd struct{}

// Run executes the get display name command
func (cmd *MailSettingsDisplayNameGetCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	accountDetails, err := mailClient.GetAccountDetails(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to get account details: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Create display struct with just display name and email
	displayInfo := struct {
		DisplayName  string `json:"displayName"`
		EmailAddress string `json:"emailAddress"`
	}{
		DisplayName:  accountDetails.AccountDisplayName,
		EmailAddress: accountDetails.EmailAddress,
	}

	return fp.Formatter.Print(displayInfo)
}

// MailSettingsDisplayNameSetCmd updates the account display name
type MailSettingsDisplayNameSetCmd struct {
	Name string `arg:"" help:"New display name" required:""`
}

// Run executes the set display name command
func (cmd *MailSettingsDisplayNameSetCmd) Run(sp *ServiceProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would update display name to: %s\n", cmd.Name)
		return nil
	}

	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	if err := mailClient.UpdateDisplayName(ctx, cmd.Name); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to update display name: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Display name updated to %s\n", cmd.Name)
	return nil
}

// MailSettingsForwardingGetCmd displays forwarding settings
type MailSettingsForwardingGetCmd struct{}

// Run executes the get forwarding command
func (cmd *MailSettingsForwardingGetCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	accountDetails, err := mailClient.GetAccountDetails(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to get account details: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Check if forward details exist
	if len(accountDetails.ForwardDetails) == 0 || string(accountDetails.ForwardDetails) == "null" {
		fmt.Fprintln(os.Stderr, "No forwarding configured")
		return nil
	}

	// Parse forward settings
	var forwardSettings zoho.ForwardSettings
	if err := json.Unmarshal(accountDetails.ForwardDetails, &forwardSettings); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to parse forwarding settings: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Display forwarding details
	return fp.Formatter.Print(forwardSettings)
}
