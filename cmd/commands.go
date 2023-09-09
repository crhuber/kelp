package cmd

import (
	"errors"
	"fmt"
	"strings"

	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/install"
	"crhuber/kelp/pkg/rm"
	"crhuber/kelp/pkg/utils"

	"github.com/spf13/cobra"
)

func BrowseCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "browse",
		Aliases: []string{"open"},
		Short:   "Browse to project github page",
		Long:    `Browse to project github page`,
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
			p, err := kc.GetPackage(repo)
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

			err = kc.AddPackage(ownerRepo[0], ownerRepo[1], AddRelease)
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

func SetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set",
		Short: "Set package configuration in kelp config",
		Long:  `Set package configuration in kelp config`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("owner/repo argument required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := args[0]
			// load config
			kc, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			err = kc.SetPackage(repo, SetRelease, Description, Binary)
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
			p, err := kc.GetPackage(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			err = install.Install(p.Owner, p.Repo, p.Release)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func UpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "update",
		Aliases: []string{"upgrade"},
		Short:   "update kelp packge",
		Long:    `update kelp packge`,
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
			p, err := kc.GetPackage(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			// handle http packages
			if strings.HasPrefix(p.Release, "http") {
				return errors.New("update functionality not supported for http packages")
			}

			ghr, err := utils.GetGithubRelease(p.Owner, p.Repo, "latest")
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			if ghr.TagName == p.Release {
				fmt.Printf("Latest release %s already matches release %s in kelp config", ghr.TagName, p.Release)
				return nil
			}

			fmt.Printf("Latest release %s. Kelp configured release %s. Update config [y/n] ? : ", ghr.TagName, p.Release)

			var confirmation string
			confirmation = strings.TrimSpace(confirmation)
			confirmation = strings.ToLower(confirmation)

			// Taking input from user
			fmt.Scanln(&confirmation)
			if confirmation == "y" || confirmation == "yes" {
				err = kc.SetPackage(p.Repo, ghr.TagName, "", "")
				if err != nil {
					return fmt.Errorf("%s", err)
				}
				// save config
				err = kc.Save()
				if err != nil {
					return fmt.Errorf("%s", err)
				}
			}

			return nil
		},
	}
}

func GetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get package details",
		Long:  `Get package details`,
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
			p, err := kc.GetPackage(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			fmt.Printf("\n[%s/%s]", p.Owner, p.Repo)
			fmt.Printf("\nRelease: %s", p.Release)
			fmt.Printf("\nDescription: %s", p.Description)
			fmt.Printf("\nUrl: https://github.com/%s/%s", p.Owner, p.Repo)
			fmt.Printf("\nBinary: %s", p.Binary)
			fmt.Printf("\nUpdated At: %s", p.UpdatedAt)
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
			kp, err := kc.GetPackage(repo)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			// remove the binary
			// if binary uses an alias remove that instead
			var binary string
			if kp.Binary != "" {
				binary = kp.Binary
			} else {
				binary = kp.Repo
			}
			err = rm.RemoveBinary(binary)
			if err != nil {
				return fmt.Errorf("%s", err)
			}

			// remove from config
			err = kc.RemovePackage(kp.Repo)
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
