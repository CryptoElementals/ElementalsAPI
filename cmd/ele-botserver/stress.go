/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// stressCmd represents the stress command
var stressCmd = &cobra.Command{
	Use:   "stress",
	Short: "stress test elementra backend server",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("stress called")
	},
}

func init() {
	rootCmd.AddCommand(stressCmd)
	stressCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	stressCmd.MarkFlagRequired("config")
}

