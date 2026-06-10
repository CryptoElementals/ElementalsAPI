package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	TAG     string
	GOVER   string
	COMMIT  string
	BLDTIME string
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "show the version of ele-chainserver",
	Run: func(cmd *cobra.Command, args []string) {
		ver := "ele-chainserver\n" +
			"  Version: " + TAG + "\n" +
			"  Commit ID: " + COMMIT + "\n" +
			"  Build: " + BLDTIME + "\n" +
			"  Go Version: " + GOVER + "\n"
		fmt.Print(ver)
	},
}
