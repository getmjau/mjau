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
		if cmd.Help() != nil {
			os.Exit(0)
		}
	},
}

func Execute() {
	rootCmd.PersistentFlags().
		StringP("config", "c", "mjau.yaml", "config file")
	rootCmd.PersistentFlags().BoolP("full-request", "F", false, "print full request")

	// request
	rootCmd.PersistentFlags().BoolP("request-body", "B", false, "print request body")
	rootCmd.PersistentFlags().BoolP("request-headers", "H", false, "print request headers")

	// response
	rootCmd.PersistentFlags().BoolP("body", "b", false, "print response body")
	rootCmd.PersistentFlags().BoolP("headers", "r", false, "print response headers")

	rootCmd.PersistentFlags().BoolP("show-variables", "V", false, "print variables")

	rootCmd.PersistentFlags().BoolP("show-commands", "C", false, "print commands")

	rootCmd.PersistentFlags().BoolP("show-asserts", "A", false, "print asserts")

	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output, prints everything")

	rootCmd.PersistentFlags().StringP("env", "e", "default", "environment to use")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
