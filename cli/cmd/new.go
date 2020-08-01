package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var newCommand = &cobra.Command{
	Use:   "new",
	Short: "Scaffold a new core application",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("this command doesn't do anything atm...")
	},
}
