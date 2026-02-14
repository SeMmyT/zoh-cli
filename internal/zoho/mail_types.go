package zoho

// MailAccount represents a Zoho Mail account
type MailAccount struct {
	AccountID          string `json:"accountId"`
	EmailAddress       string `json:"emailAddress"`
	AccountDisplayName string `json:"accountDisplayName"`
	Type               string `json:"type"`
	Status             string `json:"status"`
}

// MailAccountListResponse is the response for list accounts
type MailAccountListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []MailAccount `json:"data"`
}

// Folder represents a mail folder
type Folder struct {
	FolderID     string `json:"folderId"`
	FolderName   string `json:"folderName"`
	FolderType   string `json:"folderType"`
	Path         string `json:"path"`
	UnreadCount  int    `json:"unreadCount"`
	MessageCount int    `json:"messageCount"`
}

// FolderListResponse is the response for list folders
type FolderListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []Folder `json:"data"`
}

// Label represents a mail label/tag
type Label struct {
	LabelID    string `json:"labelId"`
	LabelName  string `json:"labelName"`
	LabelColor string `json:"labelColor"`
}

// LabelListResponse is the response for list labels
type LabelListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []Label `json:"data"`
}

// MessageSummary represents a message in list view
type MessageSummary struct {
	MessageID     string `json:"messageId"`
	ThreadID      string `json:"threadId"`
	Subject       string `json:"subject"`
	FromAddress   string `json:"fromAddress"`
	Sender        string `json:"sender"`
	ReceivedTime  int64  `json:"receivedTime"`  // Unix milliseconds
	Status        string `json:"status"`        // READ/UNREAD
	HasAttachment bool   `json:"hasAttachment"`
	FlagID        int    `json:"flagid"`
	Priority      int    `json:"priority"`
	Summary       string `json:"summary"`
}

// MessageListResponse is the response for list messages
type MessageListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []MessageSummary `json:"data"`
}

// MessageMetadata represents full message metadata
type MessageMetadata struct {
	MessageID     string `json:"messageId"`
	ThreadID      string `json:"threadId"`
	FolderID      string `json:"folderId"`
	Subject       string `json:"subject"`
	FromAddress   string `json:"fromAddress"`
	Sender        string `json:"sender"`
	ToAddress     string `json:"toAddress"`
	CcAddress     string `json:"ccAddress"`
	SentDateInGMT int64  `json:"sentDateInGMT"` // Unix milliseconds
	ReceivedTime  int64  `json:"receivedTime"`
	MessageSize   int64  `json:"messageSize"`
	HasAttachment bool   `json:"hasAttachment"`
	HasInline     bool   `json:"hasInline"`
	Status        string `json:"status"`
	Priority      int    `json:"priority"`
	FlagID        int    `json:"flagid"`
}

// MessageMetadataResponse is the response for message details
type MessageMetadataResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data MessageMetadata `json:"data"`
}

// MessageContent represents message body content
type MessageContent struct {
	MessageID string `json:"messageId"`
	Content   string `json:"content"` // HTML body
}

// MessageContentResponse is the response for message content
type MessageContentResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data MessageContent `json:"data"`
}

// Attachment represents a message attachment
type Attachment struct {
	AttachmentID   string `json:"attachmentId"`
	AttachmentName string `json:"attachmentName"`
	AttachmentSize int64  `json:"attachmentSize"`
	AttachmentType string `json:"attachmentType"` // MIME type
}

// AttachmentListResponse is the response for list attachments
type AttachmentListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []Attachment `json:"data"`
}

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	FromAddress string                 `json:"fromAddress"`
	ToAddress   string                 `json:"toAddress"`
	CcAddress   string                 `json:"ccAddress,omitempty"`
	BccAddress  string                 `json:"bccAddress,omitempty"`
	Subject     string                 `json:"subject"`
	Content     string                 `json:"content"`
	MailFormat  string                 `json:"mailFormat,omitempty"` // "html" or "plaintext"
	Action      string                 `json:"action,omitempty"`     // "reply", "replyall", "forward"
	Attachments []AttachmentReference `json:"attachments,omitempty"`
}

// AttachmentReference represents an uploaded attachment reference
type AttachmentReference struct {
	StoreName      string `json:"storeName"`
	AttachmentName string `json:"attachmentName"`
	AttachmentPath string `json:"attachmentPath"`
}

// AttachmentUploadResponse is the response for attachment upload
type AttachmentUploadResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data AttachmentReference `json:"data"`
}

// SendEmailResponse is the response for send email
type SendEmailResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
}
