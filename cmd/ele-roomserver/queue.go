/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/spf13/cobra"
)

// queueCmd represents the queue command
var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "tool to manage queue",
}

var queueJoinCmd = &cobra.Command{
	Use:   "join",
	Short: "tool to manage queue",
}

var queueExitCmd = &cobra.Command{
	Use:   "exit",
	Short: "tool to manage queue",
}

func init() {
	rootCmd.AddCommand(queueCmd)
	queueCmd.PersistentFlags().StringVarP(&roomServerEndpoint, "room-server-endpoint", "r", "", "room server endpoint")
	queueCmd.PersistentFlags().StringVarP(&playerAddress, "address", "a", "", "player wallet address")
	queueCmd.PersistentFlags().StringVarP(&tempAddress, "temp-addr", "t", "", "temporary address for locking")
	queueCmd.MarkFlagRequired("room-server-endpoint")
	queueCmd.AddCommand(queueJoinCmd)
	queueCmd.AddCommand(queueExitCmd)

}
