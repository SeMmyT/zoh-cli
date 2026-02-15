package cli

import (
	"fmt"
	"strings"
	"sync"

	"github.com/semmy-space/zoh/internal/auth"
	"github.com/semmy-space/zoh/internal/config"
	"github.com/semmy-space/zoh/internal/output"
	"github.com/semmy-space/zoh/internal/secrets"
	"github.com/semmy-space/zoh/internal/zoho"
)

// ServiceProvider lazily creates and caches Zoho service clients.
type ServiceProvider struct {
	cfg *config.Config

	adminOnce sync.Once
	admin     zoho.AdminService
	adminErr  error

	mailOnce sync.Once
	mail     zoho.MailService
	mailErr  error
}

// NewServiceProvider creates a ServiceProvider with the given config.
func NewServiceProvider(cfg *config.Config) *ServiceProvider {
	return &ServiceProvider{cfg: cfg}
}

// Admin returns the AdminService, creating it on first call.
func (sp *ServiceProvider) Admin() (zoho.AdminService, error) {
	sp.adminOnce.Do(func() {
		store, err := secrets.NewStore()
		if err != nil {
			sp.adminErr = &output.CLIError{
				ExitCode: output.ExitGeneral,
				Message:  fmt.Sprintf("Failed to initialize secrets store: %v", err),
			}
			return
		}

		tokenCache, err := auth.NewTokenCache(sp.cfg, store)
		if err != nil {
			sp.adminErr = &output.CLIError{
				ExitCode: output.ExitGeneral,
				Message:  fmt.Sprintf("Failed to initialize token cache: %v", err),
			}
			return
		}

		adminClient, err := zoho.NewAdminClient(sp.cfg, tokenCache)
		if err != nil {
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
				sp.adminErr = &output.CLIError{
					ExitCode: output.ExitAuth,
					Message:  fmt.Sprintf("Authentication failed: %v\n\nRun: zoh auth login", err),
				}
				return
			}
			sp.adminErr = &output.CLIError{
				ExitCode: output.ExitAPIError,
				Message:  fmt.Sprintf("Failed to create admin client: %v", err),
			}
			return
		}

		sp.admin = adminClient
	})
	return sp.admin, sp.adminErr
}

// Mail returns the MailService, creating it on first call.
func (sp *ServiceProvider) Mail() (zoho.MailService, error) {
	sp.mailOnce.Do(func() {
		store, err := secrets.NewStore()
		if err != nil {
			sp.mailErr = &output.CLIError{
				ExitCode: output.ExitGeneral,
				Message:  fmt.Sprintf("Failed to initialize secrets store: %v", err),
			}
			return
		}

		tokenCache, err := auth.NewTokenCache(sp.cfg, store)
		if err != nil {
			sp.mailErr = &output.CLIError{
				ExitCode: output.ExitGeneral,
				Message:  fmt.Sprintf("Failed to initialize token cache: %v", err),
			}
			return
		}

		mailClient, err := zoho.NewMailClient(sp.cfg, tokenCache)
		if err != nil {
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "unauthorized") {
				sp.mailErr = &output.CLIError{
					ExitCode: output.ExitAuth,
					Message:  fmt.Sprintf("Authentication failed: %v\n\nRun: zoh auth login", err),
				}
				return
			}
			sp.mailErr = &output.CLIError{
				ExitCode: output.ExitAPIError,
				Message:  fmt.Sprintf("Failed to create mail client: %v", err),
			}
			return
		}

		sp.mail = mailClient
	})
	return sp.mail, sp.mailErr
}
