package cli

import (
	"fmt"

	"feedctl/internal/app"

	"github.com/spf13/cobra"
)

func newAddCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "add", Short: "add a source"}
	var id, name, tags string
	var dryRun, yes bool
	rss := &cobra.Command{
		Use:   "rss URL",
		Short: "add an RSS source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = yes
			res, err := app.AddRSS(cmd.Context(), args[0], app.AddRSSParams{ID: id, Name: name, Tags: splitTags(tags), DryRun: dryRun})
			if err != nil {
				return err
			}
			if opts.JSON {
				return writeSuccess(cmd.OutOrStdout(), res.Action, res.DryRun, res)
			}
			if dryRun {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would create RSS source %s at %s (%d items found)\n", res.SourceID, res.ConfigPath, res.ItemsFound)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created RSS source %s at %s (%d items found)\n", res.SourceID, res.ConfigPath, res.ItemsFound)
			}
			return nil
		},
	}
	rss.Flags().StringVar(&id, "id", "", "source id")
	rss.Flags().StringVar(&name, "name", "", "source name")
	rss.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	rss.Flags().BoolVar(&dryRun, "dry-run", false, "show planned change without writing")
	rss.Flags().BoolVar(&yes, "yes", false, "confirm non-interactively")

	var tgID, tgName, tgTags string
	var tgMaxItems int
	var tgDryRun, tgYes bool
	telegram := &cobra.Command{
		Use:   "telegram CHANNEL",
		Short: "add a public Telegram channel source",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = tgYes
			res, err := app.AddTelegram(cmd.Context(), args[0], app.AddTelegramParams{ID: tgID, Name: tgName, Tags: splitTags(tgTags), MaxItems: tgMaxItems, DryRun: tgDryRun})
			if err != nil {
				return err
			}
			if opts.JSON {
				return writeSuccess(cmd.OutOrStdout(), res.Action, res.DryRun, res)
			}
			if tgDryRun {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would create Telegram source %s at %s (%d posts found)\n", res.SourceID, res.ConfigPath, res.ItemsFound)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created Telegram source %s at %s (%d posts found)\n", res.SourceID, res.ConfigPath, res.ItemsFound)
			}
			return nil
		},
	}
	telegram.Flags().StringVar(&tgID, "id", "", "source id")
	telegram.Flags().StringVar(&tgName, "name", "", "source name")
	telegram.Flags().StringVar(&tgTags, "tags", "", "comma-separated tags")
	telegram.Flags().IntVar(&tgMaxItems, "max-items", 0, "maximum Telegram posts to fetch per sync")
	telegram.Flags().BoolVar(&tgDryRun, "dry-run", false, "show planned change without writing")
	telegram.Flags().BoolVar(&tgYes, "yes", false, "confirm non-interactively")

	unsupported := func(kind string) *cobra.Command {
		return &cobra.Command{Use: kind + " URL", Short: kind + " sources are not in the MVP", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
			return app.AppError("unsupported-source-type", kind+" sources are not supported in the MVP", nil)
		}}
	}
	cmd.AddCommand(rss, telegram, unsupported("html"))
	return cmd
}
