package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"feedctl/internal/tui"

	"github.com/spf13/cobra"
)

func Execute() int {
	opts := &options{}
	root := newRootCommand(opts)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SilenceUsage = true
	root.SilenceErrors = true
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := root.ExecuteContext(ctx); err != nil {
		if opts.JSON {
			_ = writeError(os.Stdout, err)
		} else {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		return exitCode(err)
	}
	return 0
}

func newRootCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "feedctl",
		Short: "feedctl — local-first terminal content inbox",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(cmd.Context())
		},
	}
	cmd.PersistentFlags().BoolVar(&opts.JSON, "json", false, "emit JSON output")
	cmd.AddCommand(newTUICommand(), newSyncCommand(opts), newAddCommand(opts), newSourcesCommand(opts), newConfigCommand(opts), newItemsCommand(opts), newStorageCommand(opts), newStatusCommand(opts))
	return cmd
}

func newTUICommand() *cobra.Command {
	return &cobra.Command{Use: "tui", Short: "open the TUI", RunE: func(cmd *cobra.Command, args []string) error { return tui.Run(cmd.Context()) }}
}
