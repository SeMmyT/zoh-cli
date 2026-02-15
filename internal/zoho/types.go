package zoho

// AccountsResponse is the response from GET /api/accounts
type AccountsResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []AccountData `json:"data"`
}

// AccountData represents a single account from /api/accounts
type AccountData struct {
	AccountID string `json:"accountId"`
	ZUID      int64  `json:"zuid"`
	PolicyID  struct {
		Zoid int64 `json:"zoid"`
	} `json:"policyId"`
	PrimaryEmailAddress string `json:"primaryEmailAddress"`
	DisplayName         string `json:"displayName"`
	Role                string `json:"role"`
}

// EmailAddress represents a Zoho user email address entry
type EmailAddress struct {
	MailID    string `json:"mailId"`
	IsAlias   bool   `json:"isAlias"`
	IsPrimary bool   `json:"isPrimary"`
}

// User represents a Zoho user account
// Field types verified against official Zoho docs:
// https://www.zoho.com/mail/help/api/get-org-users-details.html
type User struct {
	ZUID              int64          `json:"zuid"`
	AccountID         string         `json:"accountId"` // Zoho returns this as a quoted string
	EmailAddress      []EmailAddress `json:"emailAddress"`
	PrimaryEmailID    string         `json:"primaryEmailAddress"`
	FirstName         string         `json:"firstName"`
	LastName          string         `json:"lastName"`
	DisplayName       string         `json:"displayName"`
	Role              string         `json:"role"`
	MailboxStatus     string         `json:"mailboxStatus"`
	UsedStorage       int64          `json:"usedStorage"`
	PlanStorage       int64          `json:"planStorage"`
	TFAEnabled        bool           `json:"tfaEnabled"`
	IMAPAccessEnabled bool           `json:"imapAccessEnabled"`
	POPAccessEnabled  bool           `json:"popAccessEnabled"`
	LastLogin         int64          `json:"lastLogin"` // Unix timestamp in milliseconds
}

// PrimaryEmail returns the primary email address string for the user
func (u *User) PrimaryEmail() string {
	if u.PrimaryEmailID != "" {
		return u.PrimaryEmailID
	}
	for _, ea := range u.EmailAddress {
		if ea.IsPrimary {
			return ea.MailID
		}
	}
	if len(u.EmailAddress) > 0 {
		return u.EmailAddress[0].MailID
	}
	return ""
}

// UserListResponse is the response from GET /api/organization/{zoid}/accounts
type UserListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []User `json:"data"`
}

// UserDetailResponse is the response from GET /api/organization/{zoid}/accounts/{accountId}
type UserDetailResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data User `json:"data"`
}

// CreateUserRequest is the request body for POST /api/organization/{zoid}/accounts
type CreateUserRequest struct {
	PrimaryEmailAddress string   `json:"primaryEmailAddress"`
	Password            string   `json:"password"`
	FirstName           string   `json:"firstName"`
	LastName            string   `json:"lastName"`
	DisplayName         string   `json:"displayName"`
	Role                string   `json:"role"`
	Country             string   `json:"country"`
	Language            string   `json:"language"`
	TimeZone            string   `json:"timeZone"`
	OneTimePassword     bool     `json:"oneTimePassword"`
	GroupMailList       []string `json:"groupMailList,omitempty"`
}

// UpdateUserRequest is the request body for PUT /api/organization/{zoid}/accounts/{accountId}
type UpdateUserRequest struct {
	Mode    string `json:"mode"`
	ZUID    int64  `json:"zuid"`
	NewRole string `json:"newRole,omitempty"`
}

// Group represents a Zoho group
type Group struct {
	ZGID             int64  `json:"zgid"`
	GroupName        string `json:"groupName"`
	GroupEmailAddress string `json:"groupEmailAddress"`
	Description      string `json:"description"`
	MembersCount     int    `json:"membersCount"`
}

// GroupListResponse is the response from GET /api/organization/{zoid}/groups
type GroupListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		Groups []Group `json:"groups"`
		Count  int     `json:"count"`
	} `json:"data"`
}

// GroupDetailResponse is the response from GET /api/organization/{zoid}/groups/{zgid}
type GroupDetailResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data Group `json:"data"`
}

// GroupMember represents a member of a group
type GroupMember struct {
	MemberEmailID string `json:"memberEmailID"`
	Role          string `json:"role"`
	ZUID          int64  `json:"zuid"`
}

// GroupMembersResponse is the response from GET /api/organization/{zoid}/groups/{zgid}/members
type GroupMembersResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data []GroupMember `json:"data"`
}

// CreateGroupRequest is the request body for POST /api/organization/{zoid}/groups
type CreateGroupRequest struct {
	GroupName         string   `json:"groupName"`
	GroupEmailAddress string   `json:"groupEmailAddress"`
	Description       string   `json:"description"`
	AdminsEmailList   []string `json:"adminsEmailList,omitempty"`
	MembersEmailList  []string `json:"membersEmailList,omitempty"`
}

// AddGroupMembersRequest is the request body for adding group members
type AddGroupMembersRequest struct {
	Mode                string              `json:"mode"` // "addMailGroupMember"
	MailGroupMemberList []GroupMemberToAdd  `json:"mailGroupMemberList"`
}

// GroupMemberToAdd represents a member to add to a group
type GroupMemberToAdd struct {
	MemberEmailID string `json:"memberEmailID"`
	Role          string `json:"role"`
}

// RemoveGroupMembersRequest is the request body for removing group members
type RemoveGroupMembersRequest struct {
	Mode                string                 `json:"mode"` // "removeMailGroupMember"
	MailGroupMemberList []GroupMemberToRemove `json:"mailGroupMemberList"`
}

// GroupMemberToRemove represents a member to remove from a group
type GroupMemberToRemove struct {
	MemberEmailID string `json:"memberEmailID"`
}

// DeleteConfirmation is the request body for delete operations
type DeleteConfirmation struct {
	Mode string `json:"mode"` // "deleteUser" or similar
}

// DisableUserOpts contains options for disabling a user
type DisableUserOpts struct {
	BlockIncoming         bool `json:"blockIncoming,omitempty"`
	RemoveMailForward     bool `json:"removeMailForward,omitempty"`
	RemoveGroupMembership bool `json:"removeGroupMembership,omitempty"`
	RemoveAlias           bool `json:"removeAlias,omitempty"`
}

// Domain represents a domain in the organization
type Domain struct {
	DomainName            string `json:"domainName"`
	DomainID              string `json:"domainId"`
	VerificationStatus    bool   `json:"verificationStatus"`
	DKIMStatus            bool   `json:"dkimstatus"`
	SPFStatus             bool   `json:"spfstatus"`
	MXStatus              string `json:"mxstatus"`
	VerifiedDate          int64  `json:"verifiedDate"` // Unix milliseconds
	MailHostingEnabled    bool   `json:"mailHostingEnabled"`
	IsDomainAlias         bool   `json:"isDomainAlias"`
	IsExpired             bool   `json:"isExpired"`
	Primary               bool   `json:"primary"`
	CNAMEVerificationCode string `json:"CNAMEVerificationCode"`
	HTMLVerificationCode  string `json:"HTMLVerificationCode"`
	TXTVerificationCode   string `json:"txtRecord"` // TXT record value for DNS verification
}

// DKIM represents DKIM settings for a domain
type DKIM struct {
	Selector   string `json:"selector"`
	DomainName string `json:"domainName"`
	TXTRecord  string `json:"txtRecord"`
	Status     bool   `json:"status"`
}

// DomainListResponse is the response from GET /api/organization/{zoid}/domains
type DomainListResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		DomainVO []Domain `json:"domainVO"`
	} `json:"data"`
}

// DomainDetailResponse is the response from GET /api/organization/{zoid}/domains/{domainName}
type DomainDetailResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data Domain `json:"data"`
}

// AddDomainRequest is the request body for POST /api/organization/{zoid}/domains
type AddDomainRequest struct {
	DomainName string `json:"domainName"`
}

// DomainModeRequest is the request body for domain mode operations
type DomainModeRequest struct {
	Mode string `json:"mode"`
}

// AuditLog represents an admin action audit log entry
type AuditLog struct {
	SubCategory   string                 `json:"subCategory"`
	Data          map[string]interface{} `json:"data"`
	Type          string                 `json:"type"`
	RequestTime   int64                  `json:"requestTime"` // Unix milliseconds
	PerformedBy   string                 `json:"performedBy"`
	AuditLogType  string                 `json:"auditLogType"`
	ClientIP      string                 `json:"clientIp"`
	MainCategory  string                 `json:"mainCategory"`
	OperationType string                 `json:"operationType"`
	PerformedOn   string                 `json:"performedOn"`
	Category      string                 `json:"category"`
	Operation     string                 `json:"operation"`
}

// AuditLogResponse is the response from GET /api/organization/{zoid}/activity
type AuditLogResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		Audit         []AuditLog `json:"audit"`
		LastIndexTime string     `json:"lastIndexTime"`
		LastEntityID  string     `json:"lastEntityId"`
	} `json:"data"`
}

// LoginHistoryEntry represents a login history log entry
type LoginHistoryEntry struct {
	UserID       int64  `json:"userId"`
	EmailAddress string `json:"emailAddress"`
	IPAddress    string `json:"ipAddress"`
	LoginTime    int64  `json:"loginTime"` // Unix milliseconds
	Status       string `json:"status"`
	AccessType   string `json:"accessType"`
	ClientInfo   string `json:"clientInfo"`
}

// LoginHistoryResponse is the response from GET /api/organization/{zoid}/accounts/reports/loginHistory
type LoginHistoryResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		LoginHistory []LoginHistoryEntry `json:"loginHistory"`
		ScrollID     string              `json:"scrollId"`
	} `json:"data"`
}

// SMTPLogEntry represents an SMTP transaction log entry
type SMTPLogEntry struct {
	MessageID     string   `json:"messageId"`
	FromAddress   string   `json:"fromAddr"`
	ToAddresses   []string `json:"toAddr"`
	Subject       string   `json:"subject"`
	TransactionID string   `json:"transactionId"`
	Timestamp     int64    `json:"timestamp"` // Unix milliseconds
	Status        string   `json:"status"`
}

// SMTPLogResponse is the response from POST /api/organization/{zoid}/smtplogs
type SMTPLogResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		HasNext      bool           `json:"hnxt"`
		HasPrevious  bool           `json:"hasPrevious"`
		PageKey      string         `json:"pagekey"`
		PagePrevKey  string         `json:"pagePrevKey"`
		Response     []SMTPLogEntry `json:"response"`
	} `json:"data"`
}

// APIError represents an error response from the Zoho API
type APIError struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		MoreInfo string `json:"moreInfo"`
	} `json:"data"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	if e.Data.MoreInfo != "" {
		return e.Data.MoreInfo
	}
	return e.Status.Description
}
