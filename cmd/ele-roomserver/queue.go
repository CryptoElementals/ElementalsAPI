/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/CryptoElementals/common/room_server/worker/types"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
)

// queueCmd represents the queue command
var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "tool to manage queue",
}

var queueJoinCmd = &cobra.Command{
	Use:   "join",
	Short: "join queue",
	Run: func(cmd *cobra.Command, args []string) {
		addr := types.NewPlayerAddress(playerAddress, tempAddress)
		client, err := rpc.NewClient(roomServerEndpoint)
		if err != nil {
			fmt.Printf("connect to room server failed: %s", err.Error())
			os.Exit(1)
		}
		err = client.PubSubClient.Subscribe(addr.String(), "", make(chan *proto.Event), make(chan error))
		if err != nil {
			fmt.Printf("subscribe to room server failed: %s", err.Error())
			os.Exit(1)
		}
		err = client.RpcClient.JoinQueue(context.Background(), addr)
		if err != nil {
			fmt.Printf("join queue failed: %s", err.Error())
			os.Exit(1)
		}
	},
}

var queueExitCmd = &cobra.Command{
	Use:   "exit",
	Short: "exit queue",
	Run: func(cmd *cobra.Command, args []string) {
		addr := types.NewPlayerAddress(playerAddress, tempAddress)
		client, err := rpc.NewClient(roomServerEndpoint)
		if err != nil {
			fmt.Printf("connect to room server failed: %s", err.Error())
			os.Exit(1)
		}
		err = client.PubSubClient.Subscribe(addr.String(), "", make(chan *proto.Event), make(chan error))
		if err != nil {
			fmt.Printf("subscribe to room server failed: %s", err.Error())
			os.Exit(1)
		}
		err = client.RpcClient.ExitQueue(context.Background(), addr)
		if err != nil {
			fmt.Printf("exit queue failed: %s", err.Error())
			os.Exit(1)
		}
	},
}

var queueCheckCmd = &cobra.Command{
	Use:   "check-addr",
	Short: "check queue for a dedicated address",
	Run: func(cmd *cobra.Command, args []string) {
		addr := types.NewPlayerAddress(playerAddress, tempAddress)
		client, err := rpc.NewClient(roomServerEndpoint)
		if err != nil {
			fmt.Printf("connect to room server failed: %s", err.Error())
			os.Exit(1)
		}
		err = client.PubSubClient.Subscribe(addr.String(), "", make(chan *proto.Event), make(chan error))
		if err != nil {
			fmt.Printf("subscribe to room server failed: %s", err.Error())
			os.Exit(1)
		}
		inQueue, err := client.RpcClient.IsPlayerInQueue(context.Background(), *addr)
		if err != nil {
			fmt.Printf("exit queue failed: %s", err.Error())
			os.Exit(1)
		}
		fmt.Printf("player wallet address: %s, player temp address: %s , is in queue: %v\n", playerAddress, tempAddress, inQueue)
	},
}

func init() {
	rootCmd.AddCommand(queueCmd)
	queueCmd.PersistentFlags().StringVarP(&roomServerEndpoint, "room-server-endpoint", "r", "", "room server endpoint")
	queueCmd.PersistentFlags().StringVarP(&playerAddress, "address", "a", "", "player wallet address")
	queueCmd.PersistentFlags().StringVarP(&tempAddress, "temp-addr", "t", "", "temporary address for locking")
	queueCmd.MarkPersistentFlagRequired("temp-addr")
	queueCmd.MarkPersistentFlagRequired("address")
	queueCmd.MarkPersistentFlagRequired("room-server-endpoint")
	queueCmd.AddCommand(queueJoinCmd)
	queueCmd.AddCommand(queueExitCmd)
	queueCmd.AddCommand(queueCheckCmd)
}
