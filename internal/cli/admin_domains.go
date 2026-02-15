package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/semmy-space/zoh/internal/output"
)

// AdminDomainsListCmd lists all domains in the organization
type AdminDomainsListCmd struct{}

// Run executes the list domains command
func (cmd *AdminDomainsListCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Fetch all domains (API returns all in one response, no pagination needed)
	domains, err := adminClient.ListDomains(ctx)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch domains: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Define columns for list output
	columns := []output.Column{
		{Name: "Domain", Key: "DomainName"},
		{Name: "Verified", Key: "VerificationStatus"},
		{Name: "MX", Key: "MXStatus"},
		{Name: "DKIM", Key: "DKIMStatus"},
		{Name: "SPF", Key: "SPFStatus"},
		{Name: "Primary", Key: "Primary"},
	}

	return fp.Formatter.PrintList(domains, columns)
}

// AdminDomainsGetCmd gets details for a specific domain
type AdminDomainsGetCmd struct {
	Name string `arg:"" help:"Domain name (e.g., example.com)"`
}

// Run executes the get domain command
func (cmd *AdminDomainsGetCmd) Run(sp *ServiceProvider, fp *FormatterProvider) error {
	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Fetch domain details
	domain, err := adminClient.GetDomain(ctx, cmd.Name)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to fetch domain: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	return fp.Formatter.Print(domain)
}

// AdminDomainsAddCmd adds a new domain to the organization
type AdminDomainsAddCmd struct {
	Name string `arg:"" help:"Domain name to add (e.g., example.com)"`
}

// Run executes the add domain command
func (cmd *AdminDomainsAddCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would add domain: %s\n", cmd.Name)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Add the domain
	domain, err := adminClient.AddDomain(ctx, cmd.Name)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to add domain: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	// Print domain details
	if err := fp.Formatter.Print(domain); err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to print domain: %v", err),
			ExitCode: output.ExitGeneral,
		}
	}

	// Print verification instructions to stderr
	fmt.Fprintln(os.Stderr, "\nDomain added. To verify ownership, add one of the following DNS records:")
	if domain.TXTVerificationCode != "" {
		fmt.Fprintf(os.Stderr, "\nTXT Record:\n  %s\n", domain.TXTVerificationCode)
	}
	if domain.CNAMEVerificationCode != "" {
		fmt.Fprintf(os.Stderr, "\nCNAME Record:\n  %s\n", domain.CNAMEVerificationCode)
	}
	if domain.HTMLVerificationCode != "" {
		fmt.Fprintf(os.Stderr, "\nHTML Verification:\n  %s\n", domain.HTMLVerificationCode)
	}

	return nil
}

// AdminDomainsVerifyCmd verifies domain ownership
type AdminDomainsVerifyCmd struct {
	Name   string `arg:"" help:"Domain name to verify"`
	Method string `help:"Verification method: txt, cname, html" required:"" enum:"txt,cname,html" short:"m"`
}

// Run executes the verify domain command
func (cmd *AdminDomainsVerifyCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Map user-friendly method names to API values
	methodMap := map[string]string{
		"txt":   "verifyDomainByTXT",
		"cname": "verifyDomainByCName",
		"html":  "verifyDomainByHTML",
	}

	apiMethod := methodMap[cmd.Method]

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would verify domain %s using method=%s\n", cmd.Name, cmd.Method)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Verify the domain
	err = adminClient.VerifyDomain(ctx, cmd.Name, apiMethod)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to verify domain: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Domain verification initiated for %s using %s method.\n", cmd.Name, cmd.Method)
	return nil
}

// AdminDomainsUpdateCmd updates domain settings
type AdminDomainsUpdateCmd struct {
	Name    string `arg:"" help:"Domain name to update"`
	Setting string `help:"Setting to update: enable-hosting, disable-hosting, set-primary, enable-dkim, disable-dkim" required:"" enum:"enable-hosting,disable-hosting,set-primary,enable-dkim,disable-dkim" short:"s"`
}

// Run executes the update domain command
func (cmd *AdminDomainsUpdateCmd) Run(sp *ServiceProvider, fp *FormatterProvider, globals *Globals) error {
	// Map user-friendly setting names to API mode values
	settingMap := map[string]string{
		"enable-hosting":  "enableHosting",
		"disable-hosting": "disableHosting",
		"set-primary":     "setPrimary",
		"enable-dkim":     "enableDkim",
		"disable-dkim":    "disableDkim",
	}

	apiMode := settingMap[cmd.Setting]

	// Dry-run preview
	if globals.DryRun {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would update domain %s: mode=%s\n", cmd.Name, cmd.Setting)
		return nil
	}

	adminClient, err := sp.Admin()
	if err != nil {
		return err
	}

	ctx := context.Background()

	// Update domain settings
	err = adminClient.UpdateDomainSettings(ctx, cmd.Name, apiMode)
	if err != nil {
		return &output.CLIError{
			Message:  fmt.Sprintf("Failed to update domain settings: %v", err),
			ExitCode: output.ExitAPIError,
		}
	}

	fmt.Fprintf(os.Stderr, "Domain %s updated: %s\n", cmd.Name, cmd.Setting)
	return nil
}
