package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	service  string
	platform string
	region   string
)

func init() {
	deployCommand.Flags().StringVarP(&service, "service", "s", config.Deploy.GCloud.Service, "google cloud run service name")
	deployCommand.Flags().StringVarP(&platform, "platform", "p", config.Deploy.GCloud.Platform, "google cloud run platform")
	deployCommand.Flags().StringVarP(&region, "region", "r", config.Deploy.GCloud.Region, "google cloud region")
}

var deployCommand = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy latest docker image",
	Long:  "",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		wait("deploying latest docker image", func() (string, error) {
			// FIXME: remove config.Docker.Name
			_, err := run(fmt.Sprintf("gcloud run deploy %s --image %s:latest --platform %s --region %s", service, config.Deploy.Docker.Name, platform, region), "failed to deploy latest docker image")
			return "", err
		})
	},
}
