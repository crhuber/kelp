package cmd

import (
	"errors"
	"fmt"
	"strings"

	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/initialize"
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
			repo := args[0]
			kc, err := config.FindKelpConfig(repo)
			if err != nil {
				return fmt.Errorf("%s \n", err)
			}
			config.Browse(kc.Owner, kc.Repo)
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
			config.ConfigAdd(ownerRepo[0], ownerRepo[1], release)

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
			initialize.Initialize()
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
		Use:   "ls",
		Short: "List kelp packages",
		Long:  `List kelp packages`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.List()
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
			repo := args[0]

			kc, err := config.FindKelpConfig(repo)
			if err != nil {
				return fmt.Errorf("%s \n", err)
			}
			install.Install(kc.Owner, kc.Repo, kc.Release)
			return nil
		},
	}
}

func RmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm",
		Short: "Remove a packages from config and disk",
		Long:  `Remove a packages from config and disk`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := args[0]

			kc := config.LoadKelpConfig()
			err := kc.RemovePackage(repo)
			if err != nil {
				return errors.New(err.Error())
			}
			err = rm.RemoveBinary(repo)
			if err != nil {
				return errors.New(err.Error())
			}
			return nil
		},
	}
}
