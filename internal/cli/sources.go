package cli

import (
	"fmt"
	"strings"

	"feedctl/internal/app"

	"github.com/spf13/cobra"
)

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
