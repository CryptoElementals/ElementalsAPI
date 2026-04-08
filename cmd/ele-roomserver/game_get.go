package main

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/spf13/cobra"
)

var gameGetCmd = &cobra.Command{
	Use:   "get",
	Short: "check game info from db",
	Run: func(cmd *cobra.Command, args []string) {
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
	gameGetCmd.Flags().StringVarP(&endpoint, "endpoint", "e", "", "endpoint of mysql")
	gameGetCmd.Flags().StringVarP(&user, "user", "u", "", "user of mysql")
	gameGetCmd.Flags().StringVarP(&password, "password", "p", "", "password of mysql")
	gameGetCmd.Flags().StringVarP(&dbName, "db-name", "d", "", "db name of mysql")
	gameGetCmd.Flags().Int64VarP(&gameID, "game-id", "i", 0, "game id")
}
