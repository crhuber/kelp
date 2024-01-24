package main

import (
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/install"
	"crhuber/kelp/pkg/utils"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

var version = "1.12.1"

func main() {

	// default config
	var home, _ = os.UserHomeDir()
	var KelpConf = filepath.Join(home, "/.kelp/kelp.json")

	app := &cli.App{
		Name:    "kelp",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   KelpConf,
				Usage:   "path to kelp config file",
				EnvVars: []string{"KELP_CONFIG"},
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
				Action: func(cCtx *cli.Context) error {

					project := cCtx.Args().First()
					ownerRepo := strings.Split(project, "/")
					if len(ownerRepo) < 2 {
						return fmt.Errorf("use owner/repo format")

					}

					// load config
					kc, err := config.Load(cCtx.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					err = kc.AddPackage(ownerRepo[0], ownerRepo[1], cCtx.String("release"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}
					// save config
					err = kc.Save()
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					// auto install
					if cCtx.Bool("install") {
						err = install.Install(ownerRepo[0], ownerRepo[1], cCtx.String("release"))
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
				Action: func(cCtx *cli.Context) error {
					project := cCtx.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cCtx.String("config"))
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
				Action: func(cCtx *cli.Context) error {
					// load config
					kc, err := config.Load(cCtx.String("config"))
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
				Action: func(cCtx *cli.Context) error {

					project := cCtx.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cCtx.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					p, err := kc.GetPackage(project)
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
			},
			{
				Name:  "init",
				Usage: "initialize kelp",
				Action: func(cCtx *cli.Context) error {
					// load config
					kc, err := config.Load(cCtx.String("config"))
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
				Name:  "inspect",
				Usage: "inspect kelp bin directory",
				Action: func(cCtx *cli.Context) error {
					config.Inspect()
					return nil
				},
			},
			{
				Name:  "install",
				Usage: "install kelp package",
				Action: func(cCtx *cli.Context) error {
					project := cCtx.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cCtx.String("config"))
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
				Action: func(cCtx *cli.Context) error {
					// load config
					kc, err := config.Load(cCtx.String("config"))
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
				Action: func(cCtx *cli.Context) error {
					project := cCtx.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cCtx.String("config"))
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
				Action: func(cCtx *cli.Context) error {
					project := cCtx.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cCtx.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					err = kc.SetPackage(project, cCtx.String("release"), cCtx.String("description"), cCtx.String("binary"))
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
				Action: func(cCtx *cli.Context) error {
					project := cCtx.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cCtx.String("config"))
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
					if cCtx.Bool("install") {
						err = install.Install(kp.Owner, kp.Repo, cCtx.String("release"))
						if err != nil {
							return err
						}
					}

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
