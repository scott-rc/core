package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/spf13/cobra"
)

var modelsCommand = &cobra.Command{
	Use:   "models",
	Short: "Generate database models",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		file, err := ioutil.TempFile("", "sqlboiler.*.toml")
		done := func() {
			_ = os.Remove(file.Name())
		}
		fatalIf(err, done)

		enc := toml.NewEncoder(file)
		err = enc.Encode(config.Database.Models)
		fatalIf(err, done)

		err = toml.NewEncoder(file).Encode(map[string]interface{}{
			"psql": config.Database.Main,
		})
		fatalIf(err, done)

		_, err = run(fmt.Sprintf("sqlboiler -c %s psql", file.Name()), "failed to generate models")
		fatalIf(err, done)
	},
}
