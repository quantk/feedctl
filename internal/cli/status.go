package cli

import (
	"fmt"

	"feedctl/internal/app"
	"feedctl/internal/store"

	"github.com/spf13/cobra"
)

func newStatusCommand(opts *options) *cobra.Command {
	return &cobra.Command{Use: "status", Short: "show status", RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		status, err := a.Status()
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "status", false, status)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%d unread | src:%d | removed:%d | disk:%s | sync %s | %s\n", status.UnreadCount, status.SourceCount, status.RemovedSourceCount, store.HumanBytes(status.Storage.Total()), status.LatestSyncStatus, status.LatestSyncAt)
		return nil
	}}
}
