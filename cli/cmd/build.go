package cmd

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	name       string
	tag        string
	dockerfile string
	force      bool
	push       bool
	deploy     bool
)

func init() {
	buildCmd.Flags().BoolVar(&force, "force", false, "create the docker image with uncommited changes")
	buildCmd.Flags().StringVarP(&name, "name", "n", config.Deploy.Docker.Name, "name of the image being built")
	buildCmd.Flags().StringVarP(&tag, "tag", "t", "", "additional tag for the image")
	buildCmd.Flags().StringVarP(&dockerfile, "dockerfile", "f", "./Dockerfile", "path to the Dockerfile")
	buildCmd.Flags().BoolVarP(&push, "push", "p", false, "push built images")
	buildCmd.Flags().BoolVarP(&deploy, "deploy", "d", false, "deploy latest image")
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Create docker image",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !force {
			clean, err := isGitClean()
			fatalIf(err)
			if !clean {
				fatal(errors.New("you have un-commit changes (commit your changes or use --force)"))
			}
		}

		hash, err := getGitHash()
		fatalIf(err)

		tags := []string{name + ":latest", name + ":" + hash}
		if tag != "" {
			tags = append(tags, name+":"+tag)
		}

		path := "."
		if len(args) > 0 {
			path = args[0]
		}

		wait("building docker images", func() (string, error) {
			var b strings.Builder
			b.WriteString("docker build -f ")
			b.WriteString(dockerfile)
			for _, t := range tags {
				b.WriteString(" -t ")
				b.WriteString(t)
			}
			b.WriteString(" ")
			b.WriteString(path)
			_, err := run(b.String(), "failed to build docker images")
			if err != nil {
				return "", err
			}

			b.Reset()
			for _, t := range tags {
				b.WriteString(" -t ")
				b.WriteString(t)
				b.WriteString("\n")
			}

			return b.String(), nil
		})

		if push || deploy {
			pushCommand.Run(cmd, tags)
		}

		if deploy {
			deployCommand.Run(cmd, nil)
		}
	},
}
