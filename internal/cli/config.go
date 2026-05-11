package cli

import (
	"fmt"

	"feedctl/internal/config"

	"github.com/spf13/cobra"
)

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
