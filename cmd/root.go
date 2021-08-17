package cmd

import "github.com/spf13/cobra"

var (
	release string
)

func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kelp",
		Short:   "kelp",
		Long:    `kelp`,
		Version: version,
	}
	add := AddCmd()
	add.Flags().StringVarP(&release, "release", "r", "latest", "release to install")
	cmd.AddCommand(BrowseCmd())
	cmd.AddCommand(add)
	cmd.AddCommand(InitCmd())
	cmd.AddCommand(InspectCmd())
	cmd.AddCommand(ListCmd())
	cmd.AddCommand(InstallCmd())
	cmd.AddCommand(RmCmd())
	cmd.AddCommand(DoctorCmd())

	return cmd
}
