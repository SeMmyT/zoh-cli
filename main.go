package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/posener/complete"
	"github.com/SeMmyT/zoh/internal/cli"
	"github.com/SeMmyT/zoh/internal/output"
	"github.com/willabides/kongplete"
)

var (
	version = "dev"
)

func main() {
	cliInstance := &cli.CLI{}

	parser := kong.Must(cliInstance,
		kong.Name("zoh"),
		kong.Description("Zoho CLI for Admin and Mail operations"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": version,
		},
	)

	// Wire shell completion — intercepts tab-completion requests before normal parsing
	kongplete.Complete(parser,
		kongplete.WithPredictor("file", complete.PredictFiles("*")),
	)

	// Parse and run
	ctx, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	err = ctx.Run()
	if err != nil {
		// Handle error with proper exit code
		if cliErr, ok := err.(*output.CLIError); ok {
			// We need a formatter instance, create a basic one for error output
			formatter := output.New("plain")
			formatter.PrintError(err)
			if cliErr.Hint != "" {
				formatter.PrintHint(cliErr.Hint)
			}
			os.Exit(cliErr.ExitCode)
		}
		// Unknown error
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(output.ExitGeneral)
	}
}
