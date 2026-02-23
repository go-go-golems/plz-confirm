package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	glazed_cli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	agentcli "github.com/go-go-golems/plz-confirm/internal/cli"
	"github.com/go-go-golems/plz-confirm/internal/server"
	"github.com/go-go-golems/plz-confirm/internal/store"
	"github.com/go-go-golems/plz-confirm/pkg/doc"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rootCmd := &cobra.Command{
		Use:   "plz-confirm",
		Short: "plz-confirm: CLI + backend for agent-ui-system (Go port)",
	}

	parserConfig := glazed_cli.CobraParserConfig{
		ShortHelpSections: []string{schema.DefaultSlug},
		MiddlewaresFunc:   glazed_cli.CobraCommandDefaultMiddlewares,
	}

	confirmCmd, err := agentcli.NewConfirmCommand()
	if err != nil {
		fatal(err)
	}
	cobraConfirmCmd, err := glazed_cli.BuildCobraCommand(confirmCmd,
		glazed_cli.WithParserConfig(parserConfig),
	)
	if err != nil {
		fatal(err)
	}
	rootCmd.AddCommand(cobraConfirmCmd)

	selectCmd, err := agentcli.NewSelectCommand()
	if err != nil {
		fatal(err)
	}
	cobraSelectCmd, err := glazed_cli.BuildCobraCommand(selectCmd,
		glazed_cli.WithParserConfig(parserConfig),
	)
	if err != nil {
		fatal(err)
	}
	rootCmd.AddCommand(cobraSelectCmd)

	formCmd, err := agentcli.NewFormCommand()
	if err != nil {
		fatal(err)
	}
	cobraFormCmd, err := glazed_cli.BuildCobraCommand(formCmd,
		glazed_cli.WithParserConfig(parserConfig),
	)
	if err != nil {
		fatal(err)
	}
	rootCmd.AddCommand(cobraFormCmd)

	tableCmd, err := agentcli.NewTableCommand()
	if err != nil {
		fatal(err)
	}
	cobraTableCmd, err := glazed_cli.BuildCobraCommand(tableCmd,
		glazed_cli.WithParserConfig(parserConfig),
	)
	if err != nil {
		fatal(err)
	}
	rootCmd.AddCommand(cobraTableCmd)

	uploadCmd, err := agentcli.NewUploadCommand()
	if err != nil {
		fatal(err)
	}
	cobraUploadCmd, err := glazed_cli.BuildCobraCommand(uploadCmd,
		glazed_cli.WithParserConfig(parserConfig),
	)
	if err != nil {
		fatal(err)
	}
	rootCmd.AddCommand(cobraUploadCmd)

	imageCmd, err := agentcli.NewImageCommand()
	if err != nil {
		fatal(err)
	}
	cobraImageCmd, err := glazed_cli.BuildCobraCommand(imageCmd,
		glazed_cli.WithParserConfig(parserConfig),
	)
	if err != nil {
		fatal(err)
	}
	rootCmd.AddCommand(cobraImageCmd)

	rootCmd.AddCommand(newServeCmd(ctx))
	rootCmd.AddCommand(newWSCmd(ctx))

	// Enhanced help system
	helpSystem := help.NewHelpSystem()
	if err := doc.AddDocToHelpSystem(helpSystem); err != nil {
		fatal(err)
	}
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func newServeCmd(ctx context.Context) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the plz-confirm backend server",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := store.New()
			srv := server.New(st)
			return srv.ListenAndServe(ctx, server.Options{Addr: addr})
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":3000", "Listen address (default :3000)")
	return cmd
}

func fatal(err error) {
	_, _ = os.Stderr.WriteString(errors.Wrap(err, "plz-confirm").Error() + "\n")
	os.Exit(1)
}
