package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	TAG     = "unknown"
	COMMIT  = "unknown"
	BLDTIME = "unknown"
	GOVER   = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ele-ledgerserver %s (%s) built %s %s\n", TAG, COMMIT, BLDTIME, GOVER)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
