package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/zoho"
)

// resolveGroupID converts a group identifier (email or ZGID) to a ZGID
func resolveGroupID(ac zoho.AdminService, identifier string) (int64, error) {
	ctx := context.Background()

	// If identifier contains "@", treat as email
	if strings.Contains(identifier, "@") {
		group, err := ac.GetGroupByEmail(ctx, identifier)
		if err != nil {
			return 0, err
		}
		return group.ZGID, nil
	}

	// Otherwise, parse as int64
	zgid, err := strconv.ParseInt(identifier, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid group identifier: %s (must be ZGID or email)", identifier)
	}
	return zgid, nil
}

// AdminGroupsListCmd lists groups in the organization
type AdminGroupsListCmd struct {
	Limit int  `help:"Maximum groups to show per page" short:"l" default:"50"`
	All   bool `help:"Fetch all groups" short:"a"`
}

// Run executes the list groups command
func (cmd *AdminGroupsListCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()
	var groups []zoho.Group

	if cmd.All {
		// Use PageIterator to fetch all groups
		iterator := zoho.NewPageIterator(func(start, limit int) ([]zoho.Group, error) {
			return adminClient.ListGroups(ctx, start, limit)
		}, 50)

		groups, err = iterator.FetchAll()
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch groups: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	} else {
		// Fetch single page
		groups, err = adminClient.ListGroups(ctx, 0, cmd.Limit)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch groups: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Name", Key: "GroupName"},
		{Name: "Email", Key: "GroupEmailAddress"},
		{Name: "Members", Key: "MembersCount"},
		{Name: "ZGID", Key: "ZGID"},
	}

	return fp.Formatter.PrintList(groups, columns)
}

// AdminGroupsGetCmd gets details for a specific group
type AdminGroupsGetCmd struct {
	Identifier  string `arg:"" help:"Group ID (zgid) or group email address"`
	ShowMembers bool   `help:"Include member list" short:"m" default:"true"`
}

// Run executes the get group command
func (cmd *AdminGroupsGetCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Resolve identifier to ZGID
	zgid, err := resolveGroupID(adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to resolve group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Fetch group details
	group, err := adminClient.GetGroup(ctx, zgid)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print group details
	if err := fp.Formatter.Print(group); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to print group: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// Fetch and print members if requested
	if cmd.ShowMembers {
		members, err := adminClient.GetGroupMembers(ctx, zgid)
		if err != nil {
			return &output.CLIError{
				Message:  fmt.Sprintf("Failed to fetch group members: %v", err),
				ExitCode: output.ExitAPIError,
			}
		}

		if len(members) > 0 {
			fmt.Fprintln(os.Stderr, "\nMembers:")
			columns := []output.Column{
				{Name: "Email", Key: "MemberEmailID"},
				{Name: "Role", Key: "Role"},
				{Name: "ZUID", Key: "ZUID"},
			}
			return fp.Formatter.PrintList(members, columns)
		}
	}

	return nil
}

// AdminGroupsCreateCmd creates a new group
type AdminGroupsCreateCmd struct {
	Name        string `arg:"" help:"Group display name"`
	Email       string `help:"Group email address" required:"" short:"e"`
	Description string `help:"Group description" short:"d"`
}

// Run executes the create group command
func (cmd *AdminGroupsCreateCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would create group: %s (email=%s)\n", cmd.Name, cmd.Email)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	req := zoho.CreateGroupRequest{
		GroupName:         cmd.Name,
		GroupEmailAddress: cmd.Email,
		Description:       cmd.Description,
	}

	group, err := adminClient.CreateGroup(ctx, req)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to create group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print created group
	if err := fp.Formatter.Print(group); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to print group: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	fmt.Fprintf(os.Stderr, "\nGroup created successfully: %s\n", group.GroupEmailAddress)
	return nil
}

// AdminGroupsUpdateCmd updates a group's settings
type AdminGroupsUpdateCmd struct {
	Identifier  string `arg:"" help:"Group ID (zgid) or group email address"`
	Name        string `help:"New group name" short:"n"`
	Description string `help:"New description" short:"d"`
}

// Run executes the update group command
func (cmd *AdminGroupsUpdateCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Require at least one field to update
	if cmd.Name == "" && cmd.Description == "" {
		return &output.CLIError{
			Message:  "At least one of --name or --description is required",
			ExitCode: output.ExitUsage,
		}
	}

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would update group %s: name=%s\n", cmd.Identifier, cmd.Name)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	// Resolve identifier to ZGID
	zgid, err := resolveGroupID(adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to resolve group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	ctx := context.Background()

	// Update the group
	err = adminClient.UpdateGroup(ctx, zgid, cmd.Name, cmd.Description)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to update group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Group updated successfully (ZGID: %d)\n", zgid)
	return nil
}

// AdminGroupsDeleteCmd permanently deletes a group
type AdminGroupsDeleteCmd struct {
	Identifier string `arg:"" help:"Group ID (zgid) or group email address"`
	Confirm    bool   `help:"Confirm permanent deletion"`
}

// Run executes the delete group command
func (cmd *AdminGroupsDeleteCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Check confirmation requirement (unless --force or --dry-run)
	if !cmd.Confirm && !globals.Force && !globals.DryRun {
		return &output.CLIError{
			Message:  "Deletion requires --confirm or --force flag",
			ExitCode: output.ExitUsage,
		}
	}

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would permanently delete group: %s\n", cmd.Identifier)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	// Resolve identifier to ZGID
	zgid, err := resolveGroupID(adminClient, cmd.Identifier)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to resolve group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	ctx := context.Background()

	// Delete the group
	err = adminClient.DeleteGroup(ctx, zgid)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to delete group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Group deleted successfully (ZGID: %d)\n", zgid)
	return nil
}

// AdminGroupsMembersAddCmd adds members to a group
type AdminGroupsMembersAddCmd struct {
	Group   string   `arg:"" help:"Group ID (zgid) or group email address"`
	Members []string `arg:"" help:"Email addresses of members to add"`
	Role    string   `help:"Member role: member, moderator" default:"member" enum:"member,moderator" short:"r"`
}

// Run executes the add group members command
func (cmd *AdminGroupsMembersAddCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would add %d member(s) to group %s\n", len(cmd.Members), cmd.Group)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	// Resolve group identifier to ZGID
	zgid, err := resolveGroupID(adminClient, cmd.Group)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to resolve group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	ctx := context.Background()

	// Build member list
	members := make([]zoho.GroupMemberToAdd, len(cmd.Members))
	for i, email := range cmd.Members {
		members[i] = zoho.GroupMemberToAdd{
			MemberEmailID: email,
			Role:          cmd.Role,
		}
	}

	// Add members to group
	err = adminClient.AddGroupMembers(ctx, zgid, members)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to add members: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Added %d member(s) to group (ZGID: %d)\n", len(cmd.Members), zgid)
	return nil
}

// AdminGroupsMembersRemoveCmd removes members from a group
type AdminGroupsMembersRemoveCmd struct {
	Group   string   `arg:"" help:"Group ID (zgid) or group email address"`
	Members []string `arg:"" help:"Email addresses of members to remove"`
}

// Run executes the remove group members command
func (cmd *AdminGroupsMembersRemoveCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would remove %d member(s) from group %s\n", len(cmd.Members), cmd.Group)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	// Resolve group identifier to ZGID
	zgid, err := resolveGroupID(adminClient, cmd.Group)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to resolve group: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	ctx := context.Background()

	// Build member list
	members := make([]zoho.GroupMemberToRemove, len(cmd.Members))
	for i, email := range cmd.Members {
		members[i] = zoho.GroupMemberToRemove{
			MemberEmailID: email,
		}
	}

	// Remove members from group
	err = adminClient.RemoveGroupMembers(ctx, zgid, members)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to remove members: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Removed %d member(s) from group (ZGID: %d)\n", len(cmd.Members), zgid)
	return nil
}
