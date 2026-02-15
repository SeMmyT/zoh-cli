package cli

import (
	"context"
	"fmt"

	"github.com/semmy-space/zoh/internal/output"
)


// MailFoldersListCmd lists all mail folders
type MailFoldersListCmd struct{}

// Run executes the list folders command
func (cmd *MailFoldersListCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	folders, err := mailClient.ListFolders(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch folders: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Name", Key: "FolderName"},
		{Name: "Type", Key: "FolderType"},
		{Name: "Path", Key: "Path"},
		{Name: "Messages", Key: "MessageCount"},
		{Name: "Unread", Key: "UnreadCount"},
		{Name: "ID", Key: "FolderID"},
	}

	return fp.Formatter.PrintList(folders, columns)
}

// MailLabelsListCmd lists all mail labels
type MailLabelsListCmd struct{}

// Run executes the list labels command
func (cmd *MailLabelsListCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()
	labels, err := mailClient.ListLabels(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch labels: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Name", Key: "LabelName"},
		{Name: "Color", Key: "LabelColor"},
		{Name: "ID", Key: "LabelID"},
	}

	return fp.Formatter.PrintList(labels, columns)
}
