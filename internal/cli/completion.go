package cli

import "github.com/willabides/kongplete"

// CompletionCmd wraps kongplete's InstallCompletions
type CompletionCmd struct {
	Install kongplete.InstallCompletions `cmd:"" help:"Install shell completions for bash, zsh, or fish"`
}
