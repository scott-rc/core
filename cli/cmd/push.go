package cmd

import (
	"strings"

	"github.com/spf13/cobra"
)

var pushCommand = &cobra.Command{
	Use:   "push",
	Short: "Push docker image",
	Long:  "",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		wait("pushing docker images", func() (string, error) {
			var b strings.Builder
			for _, tag := range args {
				_, err := run("docker push "+tag, "failed to push docker image")
				if err != nil {
					return "", err
				}
				b.WriteString("- " + tag + "\n")
			}

			return b.String(), nil
		})
	},
}
