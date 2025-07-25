package main

import (
	"context"
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/install"
	"crhuber/kelp/pkg/types"
	"crhuber/kelp/pkg/utils"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {

	// default config
	var home, _ = os.UserHomeDir()
	var KelpConf = filepath.Join(home, "/.kelp/kelp.json")

	if types.GetCapabilities() == nil {
		fmt.Println("Sorry, your OS is not yet supported.")
		os.Exit(1)
	}

	app := &cli.Command{
		Name:    "kelp",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   KelpConf,
				Usage:   "path to kelp config file",
				Sources: cli.EnvVars("KELP_CONFIG"),
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "add",
				Usage: "add a new package to config",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "release",
						Aliases: []string{"r"},
						Value:   "latest",
						Usage:   "release for package",
					},
					&cli.BoolFlag{
						Name:    "install",
						Aliases: []string{"i"},
						Value:   false,
						Usage:   "also install package",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {

					project := cmd.Args().First()
					ownerRepo := strings.Split(project, "/")
					if len(ownerRepo) < 2 {
						return fmt.Errorf("use owner/repo format")

					}

					// resolve release version
					releaseFlag := cmd.String("release")
					var actualRelease string
					if releaseFlag == "latest" {
						// Get the actual latest release version from GitHub
						ghr, err := utils.GetGithubRelease(ownerRepo[0], ownerRepo[1], "latest")
						if err != nil {
							return fmt.Errorf("failed to get latest release for %s/%s: %s", ownerRepo[0], ownerRepo[1], err)
						}
						actualRelease = ghr.TagName
					} else {
						actualRelease = releaseFlag
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					err = kc.AddPackage(ownerRepo[0], ownerRepo[1], actualRelease)
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					// save config
					err = kc.Save()
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					// auto install
					if cmd.Bool("install") {
						err = install.Install(ownerRepo[0], ownerRepo[1], actualRelease)
						if err != nil {
							return err
						}
					}

					return nil
				},
			},
			{
				Name:  "browse",
				Usage: "browse to project github page",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					project := cmd.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					p, err := kc.GetPackage(project)
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					config.Browse(p.Owner, p.Repo)
					return nil
				},
			},
			{
				Name:  "doctor",
				Usage: "checks if packages are installed properly",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					kc.Doctor()
					return nil

				},
			},
			{
				Name:  "get",
				Usage: "get package details",
				Action: func(ctx context.Context, cmd *cli.Command) error {

					project := cmd.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					p, err := kc.GetPackage(project)
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					fmt.Printf("[%s/%s]\n", p.Owner, p.Repo)
					fmt.Printf("Release: %s\n", p.Release)
					fmt.Printf("Description: %s\n", p.Description)
					fmt.Printf("Url: https://github.com/%s/%s\n", p.Owner, p.Repo)
					fmt.Printf("Binary: %s\n", p.Binary)
					fmt.Printf("Updated At: %s\n", p.UpdatedAt)
					return nil
				},
			},
			{
				Name:  "init",
				Usage: "initialize kelp",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					err := config.Initialize(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					return nil
				},
			},
			{
				Name:  "inspect",
				Usage: "inspect kelp bin directory",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					config.Inspect()
					return nil
				},
			},
			{
				Name:  "install",
				Usage: "install kelp package",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					project := cmd.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					kp, err := kc.GetPackage(project)
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					err = install.Install(kp.Owner, kp.Repo, kp.Release)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "list kelp packages",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					kc.List()
					return nil
				},
			},
			{
				Name:    "remove",
				Aliases: []string{"rm"},
				Usage:   "remove a package from config and disk",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					project := cmd.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					kp, err := kc.GetPackage(project)
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
			},
			{
				Name:  "set",
				Usage: "set package configuration in config",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "release",
						Aliases: []string{"r"},
						Value:   "latest",
						Usage:   "release for package",
					},
					&cli.StringFlag{
						Name:    "description",
						Aliases: []string{"d"},
						Value:   "",
						Usage:   "description of package",
					},
					&cli.StringFlag{
						Name:    "binary",
						Aliases: []string{"b"},
						Value:   "",
						Usage:   "alias of binary",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					project := cmd.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					err = kc.SetPackage(project, cmd.String("release"), cmd.String("description"), cmd.String("binary"))
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
			},
			{
				Name:  "update",
				Usage: "update kelp package in config",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "install",
						Aliases: []string{"i"},
						Value:   false,
						Usage:   "also install package",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					project := cmd.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					kp, err := kc.GetPackage(project)
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					// handle http packages
					if strings.HasPrefix(kp.Release, "http") {
						return errors.New("update functionality not supported for http packages")
					}

					ghr, err := utils.GetGithubRelease(kp.Owner, kp.Repo, "latest")
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					if ghr.TagName == kp.Release {
						fmt.Printf("Latest release %s already matches release %s in kelp config", ghr.TagName, kp.Release)
						return nil
					}

					fmt.Printf("Latest release %s. Kelp configured release %s. Update config [y/n] ? : ", ghr.TagName, kp.Release)

					var confirmation string
					confirmation = strings.TrimSpace(confirmation)
					confirmation = strings.ToLower(confirmation)

					// Taking input from user
					fmt.Scanln(&confirmation)
					if confirmation == "y" || confirmation == "yes" {
						err = kc.SetPackage(kp.Repo, ghr.TagName, "", "")
						if err != nil {
							return fmt.Errorf("%s", err)
						}
						// save config
						err = kc.Save()
						if err != nil {
							return fmt.Errorf("%s", err)
						}
					}

					// auto install
					if cmd.Bool("install") {
						err = install.Install(kp.Owner, kp.Repo, ghr.TagName)
						if err != nil {
							return err
						}
					}

					return nil
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
