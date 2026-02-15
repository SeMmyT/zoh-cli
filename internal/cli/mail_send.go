package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/zoho"
)

// MailSendComposeCmd composes and sends a new email
type MailSendComposeCmd struct {
	To      string   `help:"Recipient email address" required:""`
	Cc      string   `help:"CC recipient(s)" short:"c"`
	Bcc     string   `help:"BCC recipient(s)" short:"b"`
	Subject string   `help:"Email subject" required:""`
	Body    string   `help:"Email body content" required:""`
	HTML    bool     `help:"Send as HTML (default: plain text)" name:"html"`
	Attach  []string `help:"File path(s) to attach (repeatable)" name:"attach" predictor:"file"`
}

// Run executes the compose command
func (cmd *MailSendComposeCmd) Run(sp *ServiceProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would send email:\n")
		fmt.Fprintf(os.Stderr, "  To: %s\n", cmd.To)
		if cmd.Cc != "" {
			fmt.Fprintf(os.Stderr, "  Cc: %s\n", cmd.Cc)
		}
		if cmd.Bcc != "" {
			fmt.Fprintf(os.Stderr, "  Bcc: %s\n", cmd.Bcc)
		}
		fmt.Fprintf(os.Stderr, "  Subject: %s\n", cmd.Subject)
		if len(cmd.Attach) > 0 {
			fmt.Fprintf(os.Stderr, "  Attachments: %d file(s)\n", len(cmd.Attach))
		}
		return nil
	}

	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Upload attachments if provided
	var attachments []zoho.AttachmentReference
	for _, filePath := range cmd.Attach {
		ref, err := mailClient.UploadAttachment(ctx, filePath)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to upload attachment %s: %v", filePath, err),
				ExitCode: output.ExitAPIError,
			}
		}
		attachments = append(attachments, *ref)
	}

	// Build send request
	req := &zoho.SendEmailRequest{
		ToAddress:   cmd.To,
		CcAddress:   cmd.Cc,
		BccAddress:  cmd.Bcc,
		Subject:     cmd.Subject,
		Content:     cmd.Body,
		Attachments: attachments,
	}

	// Set mail format
	if cmd.HTML {
		req.MailFormat = "html"
	} else {
		req.MailFormat = "plaintext"
	}

	// Send email
	err = mailClient.SendEmail(ctx, req)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to send email: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	fmt.Fprintf(os.Stderr, "Email sent to %s\n", cmd.To)
	return nil
}

// MailSendReplyCmd replies to a message
type MailSendReplyCmd struct {
	MessageID string   `arg:"" help:"Message ID to reply to"`
	Folder    string   `help:"Folder name or ID" required:"" short:"f"`
	Body      string   `help:"Reply body content" required:""`
	HTML      bool     `help:"Send as HTML (default: plain text)" name:"html"`
	Attach    []string `help:"File path(s) to attach (repeatable)" name:"attach" predictor:"file"`
	All       bool     `help:"Reply to all recipients" name:"all"`
}

// Run executes the reply command
func (cmd *MailSendReplyCmd) Run(sp *ServiceProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would reply to message %s (reply-all=%v)\n", cmd.MessageID, cmd.All)
		return nil
	}

	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve folder name to folder ID
	folderID, err := resolveFolderID(ctx, mailClient, cmd.Folder)
	if err != nil {
		return err
	}

	// Fetch original message metadata
	metadata, err := mailClient.GetMessageMetadata(ctx, folderID, cmd.MessageID)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch message metadata: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Upload attachments if provided
	var attachments []zoho.AttachmentReference
	for _, filePath := range cmd.Attach {
		ref, err := mailClient.UploadAttachment(ctx, filePath)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to upload attachment %s: %v", filePath, err),
				ExitCode: output.ExitAPIError,
			}
		}
		attachments = append(attachments, *ref)
	}

	// Build send request
	req := &zoho.SendEmailRequest{
		ToAddress:   metadata.FromAddress,
		Subject:     "Re: " + metadata.Subject,
		Content:     cmd.Body,
		Attachments: attachments,
	}

	// Set mail format
	if cmd.HTML {
		req.MailFormat = "html"
	} else {
		req.MailFormat = "plaintext"
	}

	// If reply-all, add CC recipients
	if cmd.All {
		ccList := []string{}
		if metadata.ToAddress != "" {
			ccList = append(ccList, metadata.ToAddress)
		}
		if metadata.CcAddress != "" {
			ccList = append(ccList, metadata.CcAddress)
		}
		req.CcAddress = strings.Join(ccList, ",")
	}

	// Send reply
	if cmd.All {
		err = mailClient.ReplyAllToEmail(ctx, cmd.MessageID, req)
	} else {
		err = mailClient.ReplyToEmail(ctx, cmd.MessageID, req)
	}

	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to send reply: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	if cmd.All {
		fmt.Fprintf(os.Stderr, "Reply sent to all recipients\n")
	} else {
		fmt.Fprintf(os.Stderr, "Reply sent to %s\n", metadata.FromAddress)
	}
	return nil
}

// MailSendForwardCmd forwards a message
type MailSendForwardCmd struct {
	MessageID string   `arg:"" help:"Message ID to forward"`
	Folder    string   `help:"Folder name or ID" required:"" short:"f"`
	To        string   `help:"Recipient email address" required:""`
	Body      string   `help:"Additional message body" default:""`
	HTML      bool     `help:"Send as HTML (default: plain text)" name:"html"`
	Attach    []string `help:"File path(s) to attach (repeatable)" name:"attach" predictor:"file"`
}

// Run executes the forward command
func (cmd *MailSendForwardCmd) Run(sp *ServiceProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would forward message %s to %s\n", cmd.MessageID, cmd.To)
		return nil
	}

	mailClient, err := sp.Mail()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve folder name to folder ID
	folderID, err := resolveFolderID(ctx, mailClient, cmd.Folder)
	if err != nil {
		return err
	}

	// Fetch original message metadata for subject
	metadata, err := mailClient.GetMessageMetadata(ctx, folderID, cmd.MessageID)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch message metadata: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Upload attachments if provided
	var attachments []zoho.AttachmentReference
	for _, filePath := range cmd.Attach {
		ref, err := mailClient.UploadAttachment(ctx, filePath)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to upload attachment %s: %v", filePath, err),
				ExitCode: output.ExitAPIError,
			}
		}
		attachments = append(attachments, *ref)
	}

	// Build send request
	req := &zoho.SendEmailRequest{
		ToAddress:   cmd.To,
		Subject:     "Fwd: " + metadata.Subject,
		Content:     cmd.Body,
		Attachments: attachments,
	}

	// Set mail format
	if cmd.HTML {
		req.MailFormat = "html"
	} else {
		req.MailFormat = "plaintext"
	}

	// Send forward
	err = mailClient.ForwardEmail(ctx, cmd.MessageID, req)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to forward message: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	fmt.Fprintf(os.Stderr, "Message forwarded to %s\n", cmd.To)
	return nil
}

// resolveFolderID resolves a folder name to folder ID, fallback to treating input as ID
func resolveFolderID(ctx context.Context, mc zoho.MailService, folderNameOrID string) (string, error) {
	folder, err := mc.GetFolderByName(ctx, folderNameOrID)
	if err != nil {
		// If GetFolderByName fails, treat input as folder ID
		return folderNameOrID, nil
	}
	return folder.FolderID, nil
}
