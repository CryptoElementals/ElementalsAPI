package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	lobbyRPCConfigPath   string
	lobbyRPCAddrOverride string
	setTokenPlayerID     int64
	setTokenAmount       int32
)

var lobbyRPCCmd = &cobra.Command{
	Use:   "lobby_rpc",
	Short: "Call lobby server gRPC methods",
}

var lobbyRPCSetUserTokenAmountCmd = &cobra.Command{
	Use:   "set-user-token-amount",
	Short: "Call LobbyService.SetUserTokenAmount",
	Run: func(cmd *cobra.Command, args []string) {
		if err := config.InitToolsConfig(lobbyRPCConfigPath); err != nil {
			fmt.Printf("Failed to load config: %v\n", err)
			os.Exit(1)
		}
		addr := lobbyRPCAddrOverride
		if addr == "" {
			addr = config.ToolsGConf.Game.LobbyServerEndpoint
		}
		if addr == "" {
			addr = "127.0.0.1:50052"
		}

		dialOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(4*1024*1024),
				grpc.MaxCallSendMsgSize(4*1024*1024),
			),
		}
		conn, err := grpc.NewClient(addr, dialOpts...)
		if err != nil {
			fmt.Printf("Failed to dial lobby %s: %v\n", addr, err)
			os.Exit(1)
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := proto.NewLobbyServiceClient(conn)
		resp, err := client.SetUserTokenAmount(ctx, &proto.SetUserTokenAmountRequest{
			PlayerID:    setTokenPlayerID,
			TokenAmount: setTokenAmount,
		})
		if err != nil {
			fmt.Printf("SetUserTokenAmount failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("ok: id=%d tokens=%d points=%d locked_tokens=%d\n",
			resp.GetId(), resp.GetTokens(), resp.GetPoints(), resp.GetLockedTokens())
	},
}

func init() {
	lobbyRPCSetUserTokenAmountCmd.Flags().StringVarP(&lobbyRPCConfigPath, "config", "c", "", "tools config file path")
	lobbyRPCSetUserTokenAmountCmd.Flags().StringVar(&lobbyRPCAddrOverride, "lobby-endpoint", "", "lobby gRPC address (default: game.lobby-server-endpoint from config, else 127.0.0.1:50052)")
	lobbyRPCSetUserTokenAmountCmd.Flags().Int64Var(&setTokenPlayerID, "player-id", 0, "player ID")
	lobbyRPCSetUserTokenAmountCmd.Flags().Int32Var(&setTokenAmount, "token-amount", 0, "token amount to set")
	_ = lobbyRPCSetUserTokenAmountCmd.MarkFlagRequired("config")
	_ = lobbyRPCSetUserTokenAmountCmd.MarkFlagRequired("player-id")
	_ = lobbyRPCSetUserTokenAmountCmd.MarkFlagRequired("token-amount")

	lobbyRPCCmd.AddCommand(lobbyRPCSetUserTokenAmountCmd)
	rootCmd.AddCommand(lobbyRPCCmd)
}
