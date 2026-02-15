package zoho

import "context"

// MailService defines the interface for Zoho mail operations.
type MailService interface {
	// Folder and label operations
	ListFolders(ctx context.Context) ([]Folder, error)
	GetFolderByName(ctx context.Context, name string) (*Folder, error)
	ListLabels(ctx context.Context) ([]Label, error)

	// Message operations
	ListMessages(ctx context.Context, folderID string, start, limit int) ([]MessageSummary, error)
	GetMessageMetadata(ctx context.Context, folderID, messageID string) (*MessageMetadata, error)
	GetMessageContent(ctx context.Context, folderID, messageID string) (*MessageContent, error)
	SearchMessages(ctx context.Context, searchKey string, start, limit int) ([]MessageSummary, error)
	GetThread(ctx context.Context, folderID, threadID string, limit int) ([]MessageSummary, error)

	// Attachment operations
	ListAttachments(ctx context.Context, folderID, messageID string) ([]Attachment, error)
	DownloadAttachment(ctx context.Context, folderID, messageID, attachmentID, destPath string) error
	UploadAttachment(ctx context.Context, filePath string) (*AttachmentReference, error)

	// Send operations
	SendEmail(ctx context.Context, req *SendEmailRequest) error
	ReplyToEmail(ctx context.Context, messageID string, req *SendEmailRequest) error
	ReplyAllToEmail(ctx context.Context, messageID string, req *SendEmailRequest) error
	ForwardEmail(ctx context.Context, messageID string, req *SendEmailRequest) error

	// Settings operations
	ListSignatures(ctx context.Context) ([]Signature, error)
	AddSignature(ctx context.Context, sig *Signature) (string, error)
	GetAccountDetails(ctx context.Context) (*AccountDetails, error)
	AddVacationReply(ctx context.Context, vacation *VacationReply) error
	DisableVacationReply(ctx context.Context) error
	UpdateDisplayName(ctx context.Context, displayName string) error
}

// Compile-time interface compliance check
var _ MailService = (*MailClient)(nil)
