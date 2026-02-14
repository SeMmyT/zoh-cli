package zoho

// OrgResponse is the response from GET /api/organization/
type OrgResponse struct {
	Status struct {
		Code        int    `json:"code"`
		Description string `json:"description"`
	} `json:"status"`
	Data struct {
		OrganizationID int64  `json:"zoid"`
		CompanyName    string `json:"companyName"`
		UserCount      int    `json:"userCount"`
		GroupCount     int    `json:"groupCount"`
	} `json:"data"`
}

// User represents a Zoho user account
type User struct {
	ZUID              int64  `json:"zuid"`
	AccountID         int64  `json:"accountId"`
	EmailAddress      string `json:"emailAddress"`
	FirstName         string `json:"firstName"`
	LastName          string `json:"lastName"`
	DisplayName       string `json:"displayName"`
	Role              string `json:"role"`
	MailboxStatus     string `json:"mailboxStatus"`
	UsedStorage       int64  `json:"usedStorage"`
	PlanStorage       int64  `json:"planStorage"`
	TFAEnabled        bool   `json:"tfaEnabled"`
	IMAPAccessEnabled bool   `json:"imapAccessEnabled"`
	POPAccessEnabled  bool   `json:"popAccessEnabled"`
	LastLogin         int64  `json:"lastLogin"` // Unix timestamp
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
	Data []Group `json:"data"`
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
