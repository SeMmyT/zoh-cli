package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/zoho"
)


// AdminUsersListCmd lists users in the organization
type AdminUsersListCmd struct {
	Limit int  `help:"Maximum users to show per page" short:"l" default:"50"`
	All   bool `help:"Fetch all users (no pagination limit)" short:"a"`
}

// Run executes the list users command
func (cmd *AdminUsersListCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
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

	// Map to display structs (EmailAddress is an array in API response)
	type UserDisplay struct {
		Email         string `json:"email"`
		DisplayName   string `json:"displayName"`
		Role          string `json:"role"`
		MailboxStatus string `json:"mailboxStatus"`
		ZUID          string `json:"zuid"`
	}
	displayUsers := make([]UserDisplay, len(users))
	for i, u := range users {
		displayUsers[i] = UserDisplay{
			Email:         u.PrimaryEmail(),
			DisplayName:   u.DisplayName,
			Role:          u.Role,
			MailboxStatus: u.MailboxStatus,
			ZUID:          strconv.FormatInt(u.ZUID, 10),
		}
	}

	columns := []output.Column{
		{Name: "Email", Key: "Email"},
		{Name: "Name", Key: "DisplayName"},
		{Name: "Role", Key: "Role"},
		{Name: "Status", Key: "MailboxStatus"},
		{Name: "ZUID", Key: "ZUID"},
	}

	return fp.Formatter.PrintList(displayUsers, columns)
}

// AdminUsersGetCmd gets details for a specific user
type AdminUsersGetCmd struct {
	Identifier string `arg:"" help:"User ID (zuid) or email address"`
}

// Run executes the get user command
func (cmd *AdminUsersGetCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
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

// resolveUserID is a helper that resolves an identifier (email or ZUID) to a ZUID
func resolveUserID(ctx context.Context, ac zoho.AdminService, identifier string) (int64, *zoho.User, error) {
	// Try to parse as int64 (ZUID)
	if zuid, err := strconv.ParseInt(identifier, 10, 64); err == nil {
		user, err := ac.GetUser(ctx, zuid)
		return zuid, user, err
	}

	// Otherwise, treat as email and look up user
	user, err := ac.GetUserByEmail(ctx, identifier)
	if err != nil {
		return 0, nil, err
	}
	return user.ZUID, user, nil
}

// AdminUsersCreateCmd creates a new user
type AdminUsersCreateCmd struct {
	Email       string `arg:"" help:"Primary email address for the new user"`
	Password    string `help:"Initial password (if not set, Zoho sends setup email)" short:"p"`
	FirstName   string `help:"First name" short:"f"`
	LastName    string `help:"Last name" short:"l"`
	DisplayName string `help:"Display name" short:"d"`
	Role        string `help:"Role: member, admin" default:"member" enum:"member,admin" short:"r"`
}

// Run executes the create user command
func (cmd *AdminUsersCreateCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would create user: %s (firstName=%s, lastName=%s)\n", cmd.Email, cmd.FirstName, cmd.LastName)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Build create request
	req := zoho.CreateUserRequest{
		PrimaryEmailAddress: cmd.Email,
		Password:            cmd.Password,
		FirstName:           cmd.FirstName,
		LastName:            cmd.LastName,
		DisplayName:         cmd.DisplayName,
		Role:                cmd.Role,
	}

	user, err := adminClient.CreateUser(ctx, req)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to create user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	fmt.Fprintf(os.Stderr, "User created: %s\n", user.PrimaryEmail())

	return fp.Formatter.Print(user)
}

// AdminUsersUpdateCmd updates a user's role
type AdminUsersUpdateCmd struct {
	Identifier string `arg:"" help:"User ID (zuid) or email address"`
	Role       string `help:"New role: member, admin" required:"" enum:"member,admin" short:"r"`
}

// Run executes the update user command
func (cmd *AdminUsersUpdateCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would update user %s: role=%s\n", cmd.Identifier, cmd.Role)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve identifier to zuid
	zuid, user, err := resolveUserID(ctx, adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to find user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	oldRole := user.Role

	// Update user role
	if err := adminClient.UpdateUserRole(ctx, zuid, cmd.Role); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to update user role: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	fmt.Fprintf(os.Stderr, "User role updated: %s -> %s\n", oldRole, cmd.Role)

	// Fetch updated user and print
	updatedUser, err := adminClient.GetUser(ctx, zuid)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch updated user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	return fp.Formatter.Print(updatedUser)
}

// AdminUsersActivateCmd activates a user account
type AdminUsersActivateCmd struct {
	Identifier string `arg:"" help:"User ID (zuid) or email address"`
}

// Run executes the activate user command
func (cmd *AdminUsersActivateCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would activate user: %s\n", cmd.Identifier)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve identifier to zuid
	zuid, user, err := resolveUserID(ctx, adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to find user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Activate user
	if err := adminClient.EnableUser(ctx, zuid); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to activate user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	fmt.Fprintf(os.Stderr, "User activated: %s\n", user.PrimaryEmail())

	// Fetch updated user and print
	updatedUser, err := adminClient.GetUser(ctx, zuid)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch updated user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	return fp.Formatter.Print(updatedUser)
}

// AdminUsersDeactivateCmd deactivates a user account
type AdminUsersDeactivateCmd struct {
	Identifier    string `arg:"" help:"User ID (zuid) or email address"`
	BlockIncoming bool   `help:"Block incoming mail for deactivated user"`
	RemoveForward bool   `help:"Remove mail forwarding rules"`
	RemoveGroups  bool   `help:"Remove from all groups"`
	RemoveAliases bool   `help:"Remove email aliases"`
}

// Run executes the deactivate user command
func (cmd *AdminUsersDeactivateCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would deactivate user: %s\n", cmd.Identifier)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve identifier to zuid
	zuid, user, err := resolveUserID(ctx, adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to find user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Build options
	opts := zoho.DisableUserOpts{
		BlockIncoming:         cmd.BlockIncoming,
		RemoveMailForward:     cmd.RemoveForward,
		RemoveGroupMembership: cmd.RemoveGroups,
		RemoveAlias:           cmd.RemoveAliases,
	}

	// Deactivate user
	if err := adminClient.DisableUser(ctx, zuid, opts); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to deactivate user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	fmt.Fprintf(os.Stderr, "User deactivated: %s\n", user.PrimaryEmail())

	// Fetch updated user and print
	updatedUser, err := adminClient.GetUser(ctx, zuid)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch updated user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	return fp.Formatter.Print(updatedUser)
}

// AdminUsersDeleteCmd permanently deletes a user
type AdminUsersDeleteCmd struct {
	Identifier string `arg:"" help:"User ID (zuid) or email address"`
	Confirm    bool   `help:"Confirm permanent deletion"`
}

// Run executes the delete user command
func (cmd *AdminUsersDeleteCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Check confirmation requirement (unless --force or --dry-run)
	if !cmd.Confirm && !globals.Force && !globals.DryRun {
		return &output.CLIError{
			Message:  "Deletion requires --confirm or --force flag",
			ExitCode: output.ExitUsage,
		}
	}

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would permanently delete user: %s\n", cmd.Identifier)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve identifier to zuid
	zuid, user, err := resolveUserID(ctx, adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to find user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Delete user
	if err := adminClient.DeleteUser(ctx, zuid); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to delete user: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print confirmation to stderr
	fmt.Fprintf(os.Stderr, "User deleted permanently: %s\n", user.PrimaryEmail())

	return nil
}
