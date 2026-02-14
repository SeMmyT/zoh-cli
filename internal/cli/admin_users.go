package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/semmy-space/zoh/internal/auth"
	"github.com/semmy-space/zoh/internal/config"
	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/secrets"
	"github.com/semmy-space/zoh/internal/zoho"
)

// newAdminClient creates an AdminClient from config and stored credentials
func newAdminClient(cfg *config.Config) (*zoho.AdminClient, error) {
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

	adminClient, err := zoho.NewAdminClient(cfg, tokenCache)
	if err != nil {
		// Check if it's an authentication error
		if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
			return nil, &output.CLIError{
				Message:  fmt.Sprintf("Authentication failed: %v\n\nRun: zoh auth login", err),
				ExitCode: output.ExitAuth,
			}
		}
		return nil, &output.CLIError{
			Message:  fmt.Sprintf("Failed to create admin client: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	return adminClient, nil
}

// AdminUsersListCmd lists users in the organization
type AdminUsersListCmd struct {
	Limit int  `help:"Maximum users to show per page" short:"l" default:"50"`
	All   bool `help:"Fetch all users (no pagination limit)" short:"a"`
}

// Run executes the list users command
func (cmd *AdminUsersListCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	adminClient, err := newAdminClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	var users []zoho.User

	if cmd.All {
		// Use PageIterator to fetch all users
		iterator := zoho.NewPageIterator(func(start, limit int) ([]zoho.User, error) {
			return adminClient.ListUsers(ctx, start, limit)
		}, 50)

		users, err = iterator.FetchAll()
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch users: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	} else {
		// Fetch single page
		users, err = adminClient.ListUsers(ctx, 0, cmd.Limit)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch users: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Email", Key: "EmailAddress"},
		{Name: "Name", Key: "DisplayName"},
		{Name: "Role", Key: "Role"},
		{Name: "Status", Key: "MailboxStatus"},
		{Name: "ZUID", Key: "ZUID"},
	}

	return fp.Formatter.PrintList(users, columns)
}

// AdminUsersGetCmd gets details for a specific user
type AdminUsersGetCmd struct {
	Identifier string `arg:"" help:"User ID (zuid) or email address"`
}

// Run executes the get user command
func (cmd *AdminUsersGetCmd) Run(cfg *config.Config, fp *FormatterProvider) error {
	adminClient, err := newAdminClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	var user *zoho.User

	// Try to parse as int64 (ZUID)
	if zuid, err := strconv.ParseInt(cmd.Identifier, 10, 64); err == nil {
		user, err = adminClient.GetUser(ctx, zuid)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch user: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	} else {
		// Otherwise, treat as email
		user, err = adminClient.GetUserByEmail(ctx, cmd.Identifier)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch user: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	}

	return fp.Formatter.Print(user)
}
