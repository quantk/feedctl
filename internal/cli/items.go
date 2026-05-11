package cli

import (
	"fmt"

	"feedctl/internal/app"

	"github.com/spf13/cobra"
)

func newItemsCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "items", Short: "inspect items"}
	var unread, removed bool
	list := &cobra.Command{Use: "list", Short: "list items", RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		items, err := a.Items(app.ItemFilter{Unread: unread, RemovedSources: removed})
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "items_list", false, items)
		}
		for _, item := range items {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\tread=%s\tstarred=%s\n", item.ID, item.SourceID, item.Title, plainBool(item.ReadAt != ""), plainBool(item.Starred))
		}
		return nil
	}}
	list.Flags().BoolVar(&unread, "unread", false, "only unread")
	list.Flags().BoolVar(&removed, "removed-sources", false, "show removed-source items")
	cmd.AddCommand(list)
	cmd.AddCommand(&cobra.Command{Use: "open ITEM_ID", Short: "open original URL", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		if err := a.OpenItem(cmd.Context(), args[0]); err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "items_open", false, map[string]string{"item_id": args[0]})
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Opened %s\n", args[0])
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "markdown ITEM_ID", Short: "print Markdown path", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		path, err := a.MarkdownPath(args[0])
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "items_markdown", false, map[string]string{"item_id": args[0], "path": path})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	}})
	return cmd
}
