package cmd

import "github.com/spf13/cobra"

func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kelp",
		Short:   "kelp",
		Long:    `kelp`,
		Version: version,
	}
	cmd.AddCommand(NewBrowseCmd())
	cmd.AddCommand(NewAddCmd())
	cmd.AddCommand(NewInitCmd())
	return cmd
}
