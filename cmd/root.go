package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	Release    string
	ConfigPath string
)

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "kelp",
		Short:   "kelp",
		Long:    `kelp`,
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
			return InitializeConfig(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	// set flags
	var home, _ = os.UserHomeDir()
	var KelpConf = filepath.Join(home, "/.kelp/kelp.json")
	rootCmd.PersistentFlags().StringVarP(&ConfigPath, "config", "c", KelpConf, "path to kelp config file")
	// bind flags
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	add := AddCmd()
	add.Flags().StringVarP(&Release, "release", "r", "latest", "release to install")
	rootCmd.AddCommand(add)
	rootCmd.AddCommand(BrowseCmd())
	rootCmd.AddCommand(InitCmd())
	rootCmd.AddCommand(InspectCmd())
	rootCmd.AddCommand(ListCmd())
	rootCmd.AddCommand(InstallCmd())
	rootCmd.AddCommand(RmCmd())
	rootCmd.AddCommand(DoctorCmd())

	return rootCmd
}

func InitializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Set the base name of the config file, without the file extension because viper supports many different config file languages.
	v.SetConfigName("KELP")

	// Set as many paths as you like where viper should look for the
	// config file. We are only looking in the current working directory.
	v.AddConfigPath(".")

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix("KELP")

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Replaces underscores with periods when mapping environment variables.
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	bindFlags(cmd, v)
	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --favorite-color to STING_FAVORITE_COLOR
		// if strings.Contains(f.Name, "-") {
		// 	envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		// 	v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		// }

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
