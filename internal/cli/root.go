package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/store"
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
	if err := root.ExecuteContext(context.Background()); err != nil {
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

func newSourcesCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "sources", Short: "manage sources"}
	cmd.AddCommand(&cobra.Command{Use: "list", Short: "list sources", RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		sources, err := a.Sources(true)
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "sources_list", false, sources)
		}
		for _, s := range sources {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\tenabled=%s\t%s\t%s\n", s.ID, s.Type, s.Lifecycle, plainBool(s.Enabled), s.URL, s.Name)
		}
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "show ID", Short: "show source", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		s, err := a.Source(args[0])
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "sources_show", false, s)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "id: %s\ntype: %s\nname: %s\nurl: %s\nlifecycle: %s\nenabled: %s\ntags: %s\nlast_sync_status: %s\nlast_error: %s\n", s.ID, s.Type, s.Name, s.URL, s.Lifecycle, plainBool(s.Enabled), strings.Join(s.Tags, ","), s.LastSyncStatus, s.LastError)
		return nil
	}})
	cmd.AddCommand(&cobra.Command{Use: "test ID", Short: "test source", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		meta, err := a.TestSource(cmd.Context(), args[0])
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "sources_test", false, meta)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "ok: %s items=%d title=%s\n", args[0], meta.ItemsFound, meta.Title)
		return nil
	}})
	cmd.AddCommand(sourceToggleCommand(opts, true), sourceToggleCommand(opts, false), sourceRemoveCommand(opts))
	return cmd
}

func sourceToggleCommand(opts *options, enabled bool) *cobra.Command {
	var dryRun, yes bool
	name := "disable"
	if enabled {
		name = "enable"
	}
	cmd := &cobra.Command{Use: name + " ID", Short: name + " source", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		_ = yes
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		src, err := a.SetSourceEnabled(args[0], enabled, dryRun)
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "sources_"+name, dryRun, src)
		}
		if dryRun {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would %s source %s\n", name, args[0])
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%sd source %s\n", strings.Title(name), args[0])
		}
		return nil
	}}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned change")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm non-interactively")
	return cmd
}

func sourceRemoveCommand(opts *options) *cobra.Command {
	var dryRun, yes bool
	cmd := &cobra.Command{Use: "remove ID", Short: "remove source config", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		_ = yes
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		res, err := a.RemoveSource(args[0], dryRun)
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), res.Action, res.DryRun, res)
		}
		if dryRun {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would remove %s and keep runtime data\n", res.ConfigPath)
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Removed source %s; runtime data preserved\n", args[0])
		}
		return nil
	}}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show planned change")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm non-interactively")
	return cmd
}

func newConfigCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "inspect config"}
	cmd.AddCommand(&cobra.Command{Use: "path", Short: "show paths", RunE: func(cmd *cobra.Command, args []string) error {
		loaded, err := config.Load()
		if err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "config_path", false, loaded.Paths)
		}
		return printLines(cmd.OutOrStdout(), "config_file: "+loaded.Paths.ConfigFile, "sources_dir: "+loaded.Paths.SourcesDir, "data_root: "+loaded.Paths.DataRoot, "database: "+loaded.Paths.Database, "content_dir: "+loaded.Paths.ContentDir, "versions_dir: "+loaded.Paths.VersionsDir)
	}})
	cmd.AddCommand(&cobra.Command{Use: "validate", Short: "validate config", RunE: func(cmd *cobra.Command, args []string) error {
		loaded, err := config.Load()
		if err != nil {
			return err
		}
		if err := loaded.Validate(); err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "config_validate", false, map[string]any{"valid": true})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Config valid")
		return nil
	}})
	var formatYes bool
	format := &cobra.Command{Use: "format", Short: "format config", RunE: func(cmd *cobra.Command, args []string) error {
		_ = formatYes
		loaded, err := config.Load()
		if err != nil {
			return err
		}
		if err := config.FormatExisting(loaded); err != nil {
			return err
		}
		if opts.JSON {
			return writeSuccess(cmd.OutOrStdout(), "config_format", false, map[string]any{"formatted": true})
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Config formatted")
		return nil
	}}
	format.Flags().BoolVar(&formatYes, "yes", false, "confirm non-interactively")
	cmd.AddCommand(format)
	return cmd
}

func newItemsCommand(opts *options) *cobra.Command {
	cmd := &cobra.Command{Use: "items", Short: "inspect items"}
	var unread, removed bool
	list := &cobra.Command{Use: "list", Short: "list items", RunE: func(cmd *cobra.Command, args []string) error {
		a, err := app.Open(cmd.Context())
		if err != nil {
			return err
		}
		defer a.Close()
		items, err := a.Items(store.ItemFilter{Unread: unread, RemovedSources: removed})
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
