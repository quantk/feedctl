package cli

import (
	"fmt"

	"feedctl/internal/app"
	"feedctl/internal/store"

	"github.com/spf13/cobra"
)

func newStorageCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "storage", Short: "show storage", RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		stats, err := a.Storage()
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "storage", false, stats)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Items: %d\nCurrent markdown: %s\nVersions: %s\nDatabase: %s\nTotal: %s\n", stats.ItemsCount, store.HumanBytes(stats.CurrentMarkdownBytes), store.HumanBytes(stats.VersionsBytes), store.HumanBytes(stats.DatabaseBytes), store.HumanBytes(stats.Total()))
		return nil
	}}
	cmd.AddCommand(&cobra.Command{Use: "reconcile", Short: "reconcile storage", RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		res, err := a.ReconcileStorage()
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "storage_reconcile", false, res)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Reconciled storage: total %s\n", store.HumanBytes(res.Storage.Total()))
		return nil
	}})
	return cmd
}
