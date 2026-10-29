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
	"strings"

	"github.com/urfave/cli/v3"
)

var (
	version = "dev"
)

func main() {

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
				Value:   config.DefaultConf(),
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
					switch {
					case utils.IsKubectl(ownerRepo[0], ownerRepo[1], releaseFlag):
						var err error
						if actualRelease, err = kubectlRelease(releaseFlag); err != nil {
							return err
						}
					case releaseFlag == "latest":
						// Get the actual latest release version from GitHub
						latestRelease, err := utils.GetGithubRelease(ownerRepo[0], ownerRepo[1], "latest")
						if err != nil {
							return fmt.Errorf("failed to get latest release for %s/%s: %s", ownerRepo[0], ownerRepo[1], err)
						}
						actualRelease = latestRelease.TagName
					default:
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
							if release, err = kubectlRelease(release); err != nil {
								return err
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

					// latest/current are what the user sees; newRelease is what gets stored
					var latest, current, newRelease string
					switch {
					case utils.IsKubectl(kp.Owner, kp.Repo, kp.Release):
						latestVersion, err := utils.GetKubectlLatestRelease()
						if err != nil {
							return fmt.Errorf("failed to get latest kubectl release: %w", err)
						}
						latest, current, newRelease = latestVersion, utils.GetKubectlVersion(kp.Release), utils.GetKubectlDownloadURL(latestVersion)
					case strings.HasPrefix(kp.Release, "http"):
						return errors.New("update functionality not supported for http packages")
					default:
						ghr, err := utils.GetGithubRelease(kp.Owner, kp.Repo, "latest")
						if err != nil {
							return err
						}
						latest, current, newRelease = ghr.TagName, kp.Release, ghr.TagName
					}

					if latest == current {
						logging.LogInfo("Latest release %s already matches release %s in kelp config", latest, current)
						return nil
					}

					logging.LogInfo("Latest release %s. Kelp configured release %s. Update config [y/n] ? : ", latest, current)
					var confirmation string
					if _, err := fmt.Scanln(&confirmation); err != nil {
						return nil
					}
					if c := strings.ToLower(strings.TrimSpace(confirmation)); c != "y" && c != "yes" {
						return nil
					}

					if err := kc.SetPackage(kp.Repo, newRelease, "", ""); err != nil {
						return err
					}
					if err := kc.Save(); err != nil {
						return err
					}

					// auto install
					if cmd.Bool("install") {
						return install.Install(kp.Owner, kp.Repo, newRelease)
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

// kubectlRelease turns a kubectl version ("latest", "1.35.3", "v1.35.3") into its
// dl.k8s.io download URL; URLs pass through unchanged.
func kubectlRelease(release string) (string, error) {
	switch {
	case release == "latest" || release == "":
		latestVersion, err := utils.GetKubectlLatestRelease()
		if err != nil {
			return "", fmt.Errorf("failed to get latest kubectl release: %w", err)
		}
		return utils.GetKubectlDownloadURL(latestVersion), nil
	case strings.HasPrefix(release, "http"):
		return release, nil
	default:
		return utils.GetKubectlDownloadURL(release), nil
	}
}
