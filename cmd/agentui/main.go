package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	glazed_cli "github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	agentcli "github.com/go-go-golems/plz-confirm/internal/cli"
	"github.com/go-go-golems/plz-confirm/internal/server"
	"github.com/go-go-golems/plz-confirm/internal/store"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rootCmd := &cobra.Command{
		Use:   "agentui",
		Short: "agentui: CLI + backend for agent-ui-system (Go port)",
	}

	// Glazed standard output layers (adds --output, --fields, etc.)
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		fatal(err)
	}
	layersList := []layers.ParameterLayer{glazedLayer}

	confirmCmd, err := agentcli.NewConfirmCommand(layersList...)
	if err != nil {
		fatal(err)
	}
	cobraConfirmCmd, err := glazed_cli.BuildCobraCommand(confirmCmd,
		glazed_cli.WithParserConfig(glazed_cli.CobraParserConfig{
			ShortHelpLayers: []string{layers.DefaultSlug},
			MiddlewaresFunc: glazed_cli.CobraCommandDefaultMiddlewares,
		}),
	)
	if err != nil {
		fatal(err)
	}
	rootCmd.AddCommand(cobraConfirmCmd)

	rootCmd.AddCommand(newServeCmd(ctx))

	// Enhanced help system
	helpSystem := help.NewHelpSystem()
	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func newServeCmd(ctx context.Context) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the agentui backend server",
		RunE: func(cmd *cobra.Command, args []string) error {
			st := store.New()
			srv := server.New(st)
			return srv.ListenAndServe(ctx, server.Options{Addr: addr})
		},
	}

	cmd.Flags().StringVar(&addr, "addr", ":3001", "Listen address (default :3001)")
	return cmd
}

func fatal(err error) {
	_, _ = os.Stderr.WriteString(errors.Wrap(err, "agentui").Error() + "\n")
	os.Exit(1)
}
