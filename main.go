package main

import (
	"context"
	"crhuber/kelp/pkg/config"
	"crhuber/kelp/pkg/install"
	"crhuber/kelp/pkg/logging"
	"crhuber/kelp/pkg/rm"
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
			&cli.BoolFlag{
				Name:    "verbose",
				Value:   false,
				Usage:   "verbose output",
				Sources: cli.EnvVars("KELP_VERBOSE"),
				Action: func(_ context.Context, _ *cli.Command, val bool) error {
					logging.SetLogVerbose(val)
					return nil
				},
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
				Action: func(_ context.Context, cmd *cli.Command) error {

					project := cmd.Args().First()
					ownerRepo := strings.Split(project, "/")
					if len(ownerRepo) != 2 {
						return fmt.Errorf("use owner/repo format")
					}
					if err := config.ValidateRepoName(ownerRepo[0], ownerRepo[1]); err != nil {
						return err
					}

					// resolve release version
					releaseFlag := cmd.String("release")
					var actualRelease string
					if utils.IsKubectl(ownerRepo[0], ownerRepo[1], releaseFlag) {
						if releaseFlag == "latest" || releaseFlag == "" {
							latestVersion, err := utils.GetKubectlLatestRelease()
							if err != nil {
								return fmt.Errorf("failed to get latest kubectl release: %w", err)
							}
							actualRelease = utils.GetKubectlDownloadURL(latestVersion)
						} else if strings.HasPrefix(releaseFlag, "http") {
							actualRelease = releaseFlag
						} else {
							actualRelease = utils.GetKubectlDownloadURL(releaseFlag)
						}
					} else if releaseFlag == "latest" {
						// Get the actual latest release version from GitHub
						latestRelease, err := utils.GetGithubRelease(ownerRepo[0], ownerRepo[1], "latest")
						if err != nil {
							return fmt.Errorf("failed to get latest release for %s/%s: %s", ownerRepo[0], ownerRepo[1], err)
						}
						actualRelease = latestRelease.TagName
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
				Action: func(_ context.Context, cmd *cli.Command) error {
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
					return config.Browse(p.Owner, p.Repo)
				},
			},
			{
				Name:  "doctor",
				Usage: "checks if packages are installed properly",
				Action: func(_ context.Context, cmd *cli.Command) error {
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
				Action: func(_ context.Context, cmd *cli.Command) error {

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
				Action: func(_ context.Context, cmd *cli.Command) error {
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
				Action: func(_ context.Context, _ *cli.Command) error {
					return config.Inspect()
				},
			},
			{
				Name:  "install",
				Usage: "install kelp package",
				Action: func(_ context.Context, cmd *cli.Command) error {
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
				Action: func(_ context.Context, cmd *cli.Command) error {
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
				Action: func(_ context.Context, cmd *cli.Command) error {
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

					// remove binary from disk
					binaryName := kp.Binary
					if binaryName == "" {
						binaryName = kp.Repo
					}
					if err := rm.RemoveBinary(binaryName); err != nil {
						logging.LogInfo("Warning: could not remove binary %s from disk: %v", binaryName, err)
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
						Value:   "",
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
				Action: func(_ context.Context, cmd *cli.Command) error {
					project := cmd.Args().First()
					if project == "" {
						return errors.New("project argument required")
					}

					// load config
					kc, err := config.Load(cmd.String("config"))
					if err != nil {
						return fmt.Errorf("%s", err)
					}

					release := cmd.String("release")
					if release != "" {
						kp, err := kc.GetPackage(project)
						if err == nil && utils.IsKubectl(kp.Owner, kp.Repo, kp.Release) {
							if release == "latest" {
								latestVersion, err := utils.GetKubectlLatestRelease()
								if err != nil {
									return fmt.Errorf("failed to get latest kubectl release: %w", err)
								}
								release = utils.GetKubectlDownloadURL(latestVersion)
							} else if !strings.HasPrefix(release, "http") {
								release = utils.GetKubectlDownloadURL(release)
							}
						}
					}

					err = kc.SetPackage(project, release, cmd.String("description"), cmd.String("binary"))
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
				Action: func(_ context.Context, cmd *cli.Command) error {
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

					// handle kubectl packages
					if utils.IsKubectl(kp.Owner, kp.Repo, kp.Release) {
						latestVersion, err := utils.GetKubectlLatestRelease()
						if err != nil {
							return fmt.Errorf("failed to get latest kubectl release: %w", err)
						}

						currentVersion := utils.GetKubectlVersion(kp.Release)
						if latestVersion == currentVersion {
							logging.LogInfo("Latest release %s already matches release %s in kelp config", latestVersion, currentVersion)
							return nil
						}

						logging.LogInfo("Latest release %s. Kelp configured release %s. Update config [y/n] ? : ", latestVersion, currentVersion)

						var confirmation string
						if _, err := fmt.Scanln(&confirmation); err != nil {
							return nil
						}
						confirmation = strings.TrimSpace(confirmation)
						newURL := utils.GetKubectlDownloadURL(latestVersion)

						if strings.EqualFold(confirmation, "y") || strings.EqualFold(confirmation, "yes") {
							err = kc.SetPackage(kp.Repo, newURL, "", "")
							if err != nil {
								return fmt.Errorf("%s", err)
							}
							// save config
							err = kc.Save()
							if err != nil {
								return fmt.Errorf("%s", err)
							}
						} else {
							return nil
						}

						// auto install
						if cmd.Bool("install") {
							err = install.Install(kp.Owner, kp.Repo, newURL)
							if err != nil {
								return err
							}
						}

						return nil
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
						logging.LogInfo("Latest release %s already matches release %s in kelp config", ghr.TagName, kp.Release)
						return nil
					}

					logging.LogInfo("Latest release %s. Kelp configured release %s. Update config [y/n] ? : ", ghr.TagName, kp.Release)

					var confirmation string
					// Taking input from user
					if _, err := fmt.Scanln(&confirmation); err != nil {
						return nil
					}
					if confirmationUpper := strings.ToUpper(strings.TrimSpace(confirmation)); confirmationUpper == "Y" || confirmationUpper == "YES" {
						err = kc.SetPackage(kp.Repo, ghr.TagName, "", "")
						if err != nil {
							return fmt.Errorf("%s", err)
						}
						// save config
						err = kc.Save()
						if err != nil {
							return fmt.Errorf("%s", err)
						}
					} else {
						return nil
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
