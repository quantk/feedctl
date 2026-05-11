package cli

import (
	"fmt"
	"strings"

	"feedctl/internal/app"

	"github.com/spf13/cobra"
)

func newSyncCommand(opts *options) *cobra.Command {
	var sourceID string
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "sync RSS sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := app.Open(cmd.Context())
			if err != nil {
				return err
			}
			defer a.Close()
			res := a.Sync(cmd.Context(), sourceID)
			if opts.JSON {
				return writeSuccess(cmd.OutOrStdout(), "sync", false, res)
			}
			for _, src := range res.Sources {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %s new=%d updated=%d unchanged=%d errors=%d\n", src.SourceID, src.Status, src.NewItems, src.UpdatedItems, src.UnchangedItems, len(src.Errors))
			}
			if !res.OK {
				return app.AppError("sync-failed", strings.Join(res.Errors, "; "), nil)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&sourceID, "source", "", "sync only one source")
	return cmd
}
