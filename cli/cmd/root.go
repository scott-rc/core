package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/scott-rc/core"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:              "core",
	Short:            "",
	Long:             "",
	TraverseChildren: true,
}

var (
	configPath string
	config     *core.Config
)

var _ = func() error {
	// this is where the config path is actually bound
	flag.StringVar(&configPath, "config", "./core.toml", "path to core config file")
	flag.Parse()

	viper.SetConfigFile(configPath)
	err := viper.ReadInConfig()
	fatalIf(errors.WithMessage(err, "failed to read core.toml"))
	err = viper.Unmarshal(&config)
	fatalIf(errors.WithMessage(err, "failed to unmarshal config"))
	return nil
}()

func init() {
	// this flag is just for help text
	rootCmd.Flags().String("config", "./core.toml", "path to core config file")
	rootCmd.AddCommand(buildCmd, pushCommand, deployCommand, migrateCommand, modelsCommand)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
