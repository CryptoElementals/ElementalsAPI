package main

import (
	"fmt"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/spf13/cobra"
)

var gameGetCmd = &cobra.Command{
	Use:   "get",
	Short: "check game info from db",
	Run: func(cmd *cobra.Command, args []string) {
		if configPath != "" {
			if err := config.InitToolsConfig(configPath); err != nil {
				fmt.Printf("load tools config failed, err: %v\n", err)
				return
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
			if gameID == 0 {
				gameID = config.ToolsGConf.Game.Get.GameID
			}
		}
		if endpoint == "" || user == "" || dbName == "" {
			fmt.Printf("database endpoint/user/db-name are required (flags or tools config)\n")
			return
		}
		if gameID == 0 {
			fmt.Printf("game-id is required (flag or tools config game.get.game-id)\n")
			return
		}
		err := db.Init(&db.Config{
			Endpoint: endpoint,
			User:     user,
			Password: password,
			DbName:   dbName,
		})
		if err != nil {
			fmt.Printf("init db failed, err: %v\n", err)
			return
		}
		game, err := db.LoadGameByGameID(gameID)
		if err != nil {
			fmt.Printf("load game failed, err: %v\n", err)
			return
		}
		fmt.Printf("game info: %s\n", types.ToJsonLoggableIndent(game))
	},
}

func init() {
	gameCmd.AddCommand(gameGetCmd)
	gameGetCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
	gameGetCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "endpoint of mysql")
	gameGetCmd.Flags().StringVarP(&user, "user", "u", "", "user of mysql")
	gameGetCmd.Flags().StringVarP(&password, "password", "p", "", "password of mysql")
	gameGetCmd.Flags().StringVarP(&dbName, "db-name", "d", "", "db name of mysql")
	gameGetCmd.Flags().Int64VarP(&gameID, "game-id", "i", 0, "game id")
}
