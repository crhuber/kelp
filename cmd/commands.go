package cmd

import (
	"errors"
	"fmt"
	"os"
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
		Short: "Run the browse command",
		Long:  `Run the browse command`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("usage: kelp browse <project>")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := args[0]
			kc, err := config.FindKelpConfig(repo)
			if err != nil {
				fmt.Printf("\n %s", err)
				os.Exit(1)
			}
			config.Browse(kc.Owner, kc.Repo)
			return nil
		},
	}
}

func AddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add",
		Short: "Run the add command",
		Long:  `Run the add command`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("usage: kelp add owner/repo")
			}
			if len(args) < 2 {
				args = append(args, "latest")
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
		Short: "Run the init command",
		Long:  `Run the init command`,
		RunE: func(cmd *cobra.Command, args []string) error {
			initialize.Initialize()
			return nil
		},
	}
}

func InspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Run the inspect command",
		Long:  `Run the inspect command`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Inspect()
			return nil
		},
	}
}

func ListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "Run the list command",
		Long:  `Run the list command`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config.List()
			return nil
		},
	}
}

func InstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Run the install command",
		Long:  `Run the install command`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("usage: kelp install repo")
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
		Short: "Run the rm command",
		Long:  `Run the rm command`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("usage: kelp rm repo")
			}
			return nil
		},
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
