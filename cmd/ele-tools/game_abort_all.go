package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	gameAbortAllDryRun bool
)

var gameAbortAllCmd = &cobra.Command{
	Use:   "abort-all",
	Short: "Abort all active games (status != GAME_END) via room server",
	Long: `Calls RoomService.AbortAllActiveGames on the room server so each game uses the
same abort path as internal timeouts (persist, notify clients, unlock tokens).

With --dry-run, only lists active game IDs from the database without calling the room server.`,
	Run: func(cmd *cobra.Command, args []string) {
		if configPath != "" {
			if err := config.InitToolsConfig(configPath); err != nil {
				fmt.Printf("load tools config failed: %v\n", err)
				os.Exit(1)
			}
			if endpoint == "" {
				endpoint = config.ToolsGConf.DbCfg.Endpoint
			}
			if user == "" {
				user = config.ToolsGConf.DbCfg.User
			}
			if password == "" {
				password = config.ToolsGConf.DbCfg.Password
			}
			if dbName == "" {
				dbName = config.ToolsGConf.DbCfg.DbName
			}
			if roomServerEndpoint == "" {
				roomServerEndpoint = config.ToolsGConf.Game.RoomServerEndpoint
			}
		}

		if gameAbortAllDryRun {
			if endpoint == "" || user == "" || dbName == "" {
				fmt.Printf("database endpoint/user/db-name are required for --dry-run (flags or tools config)\n")
				os.Exit(1)
			}
			if err := db.Init(&db.Config{
				Endpoint: endpoint,
				User:     user,
				Password: password,
				DbName:   dbName,
			}); err != nil {
				fmt.Printf("init db failed: %v\n", err)
				os.Exit(1)
			}
			games, err := db.GetAllActiveGames()
			if err != nil {
				fmt.Printf("list active games failed: %v\n", err)
				os.Exit(1)
			}
			if len(games) == 0 {
				fmt.Println("no active games")
				return
			}
			for _, g := range games {
				if g == nil {
					continue
				}
				fmt.Printf("game id=%d status=%s\n", g.ID, g.Status.String())
			}
			fmt.Printf("dry-run: %d active game(s), not aborted\n", len(games))
			return
		}

		if roomServerEndpoint == "" {
			fmt.Printf("room-server-endpoint is required (flag or tools config game.room-server-endpoint)\n")
			os.Exit(1)
		}

		dialOpts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(4*1024*1024),
				grpc.MaxCallSendMsgSize(4*1024*1024),
			),
		}
		conn, err := grpc.NewClient(roomServerEndpoint, dialOpts...)
		if err != nil {
			fmt.Printf("failed to dial room server %s: %v\n", roomServerEndpoint, err)
			os.Exit(1)
		}
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		client := proto.NewRoomServiceClient(conn)
		resp, err := client.AbortAllActiveGames(ctx, &emptypb.Empty{})
		if err != nil {
			fmt.Printf("AbortAllActiveGames failed: %v\n", err)
			os.Exit(1)
		}

		for _, id := range resp.GetAbortedGameIds() {
			fmt.Printf("aborted game id=%d\n", id)
		}
		for _, f := range resp.GetFailures() {
			fmt.Printf("failed game id=%d: %s\n", f.GetGameId(), f.GetMessage())
		}

		nOK := len(resp.GetAbortedGameIds())
		nFail := len(resp.GetFailures())
		fmt.Printf("done: aborted=%d failed=%d\n", nOK, nFail)
		if nFail > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	gameCmd.AddCommand(gameAbortAllCmd)
	gameAbortAllCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	gameAbortAllCmd.Flags().StringVarP(&roomServerEndpoint, "room-server-endpoint", "r", "", "room server gRPC address")
	gameAbortAllCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "mysql endpoint (for --dry-run)")
	gameAbortAllCmd.Flags().StringVarP(&user, "user", "u", "", "mysql user (for --dry-run)")
	gameAbortAllCmd.Flags().StringVarP(&password, "password", "p", "", "mysql password (for --dry-run)")
	gameAbortAllCmd.Flags().StringVarP(&dbName, "db-name", "d", "", "mysql db name (for --dry-run)")
	gameAbortAllCmd.Flags().BoolVar(&gameAbortAllDryRun, "dry-run", false, "list active games from DB only, do not abort")
}
