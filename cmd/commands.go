package cmd

import (
	"errors"
	"fmt"
	"strings"

	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/install"
	"crhuber/kelp/pkg/rm"

	"github.com/spf13/cobra"
)

func BrowseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "browse",
		Short: "Browse to project github page",
		Long:  `Browse to project github page`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("project argument required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			kc, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			repo := args[0]
			p, err := kc.FindPackage(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			config.Browse(p.Owner, p.Repo)
			return nil
		},
	}
}

func AddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Add a new package to kelp config",
		Long:  `Add a new package to kelp config`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("owner/repo argument required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			project := args[0]
			ownerRepo := strings.Split(project, "/")
			if len(ownerRepo) < 2 {
				return fmt.Errorf("use owner/repo format")

			}
			// load config
			kc, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			err = kc.AddPackage(ownerRepo[0], ownerRepo[1], Release)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			// save config
			err = kc.Save()
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			return nil
		},
	}
}

func InitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize kelp",
		Long:  `Initialize kelp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			kc, err := config.Initialize(ConfigPath)
			kc.Path = ConfigPath
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			// save config
			err = kc.Save()
			if err != nil {
				return fmt.Errorf("error saving: %s", err)
			}
			return nil
		},
	}
}

func InspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Inspect kelp bin",
		Long:  `Inspect kelp bin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Inspect()
			return nil
		},
	}
}

func ListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List kelp packages",
		Long:    `List kelp packages`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			kc, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			kc.List()
			return nil
		},
	}
}

func InstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install kelp packge",
		Long:  `Install kelp packge`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("repo argument required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			kc, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			repo := args[0]
			p, err := kc.FindPackage(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			install.Install(p.Owner, p.Repo, p.Release)
			return nil
		},
	}
}

func RmCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove",
		Aliases: []string{"rm"},
		Short:   "Remove a packages from config and disk",
		Long:    `Remove a packages from config and disk`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			kc, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			repo := args[0]
			err = kc.RemovePackage(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			// save config
			err = kc.Save()
			if err != nil {
				return fmt.Errorf("error saving: %s", err)
			}

			err = rm.RemoveBinary(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			return nil
		},
	}
}

func DoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Checks if kelp packages are installed properly",
		Long:  `Checks if kelp packages are installed properly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			kc, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			kc.Doctor()
			return nil
		},
	}
}
