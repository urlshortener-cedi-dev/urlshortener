package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCMD = &cobra.Command{
	Use:     "version",
	Short:   "Shows version information",
	Example: "cmap version",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Printf("%s - %s - commit %s - by %s\n",
			Version,
			Date,
			Commit,
			BuiltBy,
		)

	},
}

func init() {
	rootCmd.AddCommand(versionCMD)
}
