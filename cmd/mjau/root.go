package mjau

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version string

var rootCmd = &cobra.Command{
	Use:   "mjau",
	Short: "Mjau is a api testing tool.",
	Long:  `Mjau is a api testing tool that uses yaml files to define environments and requests.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
		cmd.Help()
	},
}

func Execute() {
	rootCmd.PersistentFlags().
		StringP("config", "c", "mjau.yaml", "config file")
	rootCmd.PersistentFlags().BoolP("full-request", "f", false, "print full request")

	rootCmd.PersistentFlags().StringP("env", "e", "default", "environment to use")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
