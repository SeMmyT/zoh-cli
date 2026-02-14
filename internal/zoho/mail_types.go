package zoho

import "encoding/json"

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

// Signature represents an email signature
type Signature struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Content     string `json:"content"`
	Position    int    `json:"position"`    // 0=below quoted, 1=above quoted
	AssignUsers string `json:"assignUsers,omitempty"` // comma-separated emails
}

// SignatureListResponse is the response for list signatures
type SignatureListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []Signature `json:"data"`
}

// SignatureCreateResponse is the response for create signature
type SignatureCreateResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

// VacationReply represents vacation auto-reply settings
type VacationReply struct {
	FromDate   string `json:"fromDate"`   // MM/DD/YYYY HH:MM:SS
	ToDate     string `json:"toDate"`     // MM/DD/YYYY HH:MM:SS
	SendingInt int    `json:"sendingInt"` // reply interval in minutes
	Subject    string `json:"subject"`
	Content    string `json:"content"`
	SendTo     string `json:"sendTo"` // "all"/"contacts"/"noncontacts"/"org"/"nonOrgAll"
}

// AccountDetails represents account details including vacation and forwarding settings
type AccountDetails struct {
	AccountDisplayName string          `json:"accountDisplayName"`
	EmailAddress       string          `json:"emailAddress"`
	VacationResponse   json.RawMessage `json:"vacationResponse,omitempty"` // for GET
	ForwardDetails     json.RawMessage `json:"forwardDetails,omitempty"`   // for GET
}

// AccountDetailsResponse is the response for get account details
type AccountDetailsResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data AccountDetails `json:"data"`
}

// ForwardSettings represents email forwarding settings
type ForwardSettings struct {
	Enabled   bool   `json:"enabled"`
	ForwardTo string `json:"forwardTo"`
	KeepCopy  bool   `json:"keepCopy"`
}

// SpamCategory represents spam filter category types
type SpamCategory string

// Spam category constants for allowlist/blocklist operations
const (
	// Email allowlist/blocklist
	WhiteListEmail    SpamCategory = "WhiteListEmail"
	SpamEmail         SpamCategory = "SpamEmail"
	RejectEmail       SpamCategory = "RejectEmail"
	QuarantineEmail   SpamCategory = "QuarantineEmail"
	TrustedEmail      SpamCategory = "TrustedEmail"

	// Domain allowlist/blocklist
	WhiteListDomain   SpamCategory = "WhiteListDomain"
	SpamDomain        SpamCategory = "SpamDomain"
	RejectDomain      SpamCategory = "RejectDomain"
	QuarantineDomain  SpamCategory = "QuarantineDomain"
	TrustedDomain     SpamCategory = "TrustedDomain"
	SpamTLD           SpamCategory = "SpamTLD"
	RejectTLD         SpamCategory = "RejectTLD"
	QuarantineTLD     SpamCategory = "QuarantineTLD"

	// IP allowlist/blocklist
	WhiteListIP       SpamCategory = "WhiteListIP"
	SpamIP            SpamCategory = "SpamIP"
	RejectIP          SpamCategory = "RejectIP"
	QuarantineIP      SpamCategory = "QuarantineIP"
)

// SpamCategoryMap provides CLI-friendly name mapping to Zoho API enum values
var SpamCategoryMap = map[string]SpamCategory{
	"allowlist-email":     WhiteListEmail,
	"blocklist-email":     SpamEmail,
	"reject-email":        RejectEmail,
	"quarantine-email":    QuarantineEmail,
	"trusted-email":       TrustedEmail,
	"allowlist-domain":    WhiteListDomain,
	"blocklist-domain":    SpamDomain,
	"reject-domain":       RejectDomain,
	"quarantine-domain":   QuarantineDomain,
	"trusted-domain":      TrustedDomain,
	"blocklist-tld":       SpamTLD,
	"reject-tld":          RejectTLD,
	"quarantine-tld":      QuarantineTLD,
	"allowlist-ip":        WhiteListIP,
	"blocklist-ip":        SpamIP,
	"reject-ip":           RejectIP,
	"quarantine-ip":       QuarantineIP,
}

// SpamListEntry represents a spam list entry for display
type SpamListEntry struct {
	Category SpamCategory
	Value    string
}

// SpamUpdateRequest represents a request to update spam settings
type SpamUpdateRequest struct {
	SpamCategory string   `json:"spamCategory"`
	Value        []string `json:"Value"` // Capital V matches Zoho API
}

// SpamSettingsResponse is the response for get spam settings
type SpamSettingsResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []string `json:"data"`
}

// DeliveryLog represents a mail delivery log entry
type DeliveryLog struct {
	MessageID    string `json:"messageId"`
	Subject      string `json:"subject"`
	FromAddress  string `json:"fromAddress"`
	ToAddress    string `json:"toAddress"`
	Status       string `json:"status"`
	SentTime     string `json:"sentTime"`
	DeliveryTime string `json:"deliveryTime"`
}

// DeliveryLogListResponse is the response for delivery logs
type DeliveryLogListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []DeliveryLog `json:"data"`
}
