package zoho

import (
	"context"
	"time"
)

// AdminService defines the interface for Zoho admin operations.
type AdminService interface {
	ListUsers(ctx context.Context, start, limit int) ([]User, error)
	GetUser(ctx context.Context, accountID int64) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByIdentifier(ctx context.Context, identifier string) (*User, error)
	CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
	UpdateUserRole(ctx context.Context, zuid int64, newRole string) error
	EnableUser(ctx context.Context, zuid int64) error
	DisableUser(ctx context.Context, zuid int64, opts DisableUserOpts) error
	DeleteUser(ctx context.Context, zuid int64) error

	ListGroups(ctx context.Context, start, limit int) ([]Group, error)
	GetGroup(ctx context.Context, zgid int64) (*Group, error)
	GetGroupByEmail(ctx context.Context, email string) (*Group, error)
	GetGroupMembers(ctx context.Context, zgid int64) ([]GroupMember, error)
	CreateGroup(ctx context.Context, req CreateGroupRequest) (*Group, error)
	UpdateGroup(ctx context.Context, zgid int64, name, description string) error
	DeleteGroup(ctx context.Context, zgid int64) error
	AddGroupMembers(ctx context.Context, zgid int64, members []GroupMemberToAdd) error
	RemoveGroupMembers(ctx context.Context, zgid int64, members []GroupMemberToRemove) error

	ListDomains(ctx context.Context) ([]Domain, error)
	GetDomain(ctx context.Context, domainName string) (*Domain, error)
	AddDomain(ctx context.Context, domainName string) (*Domain, error)
	VerifyDomain(ctx context.Context, domainName, method string) error
	UpdateDomainSettings(ctx context.Context, domainName, mode string) error

	GetAuditLogs(ctx context.Context, startTime, endTime time.Time, searchKey string, limit int) ([]AuditLog, error)
	GetLoginHistory(ctx context.Context, mode string, startTime, endTime time.Time, batchSize int) ([]LoginHistoryEntry, error)
	GetSMTPLogs(ctx context.Context, startTime, endTime time.Time, searchCriteria, searchKey string, limit int) ([]SMTPLogEntry, error)
}

// Compile-time interface compliance check
var _ AdminService = (*AdminClient)(nil)
