/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/spf13/cobra"
)

// tokenCmd represents the addToken command
var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "command to manage token and points",
}

var tokenSetCmd = &cobra.Command{
	Use:   "set",
	Short: "set token and points for player",
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
		ut := dao.UserToken{
			WalletAddress: playerAddress,
			Points:        int32(points),
			TokenAmount:   int32(tokens),
		}
		err = db.Get().Save(&ut).Error
		if err != nil {
			fmt.Printf("set token and points failed, err: %v\n", err)
			return
		}
	},
}

var tokenGetCmd = &cobra.Command{
	Use:   "show",
	Short: "command to manage token and points",
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
		var ut dao.UserToken
		err = db.Get().Where("wallet_address = ?", playerAddress).First(&ut).Error
		if err != nil {
			fmt.Printf("get token and points failed, err: %v\n", err)
			return
		}
		fmt.Printf("player address: %s, points: %d, tokens: %d\n", ut.WalletAddress, ut.Points, ut.TokenAmount)
	},
}

var (
	playerAddress string
	points        int64
	tokens        int64
	endpoint      string
	user          string
	password      string
	dbName        string
)

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "", "endpoint of mysql")
	tokenCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "user of mysql")
	tokenCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "password of mysql")
	tokenCmd.PersistentFlags().StringVarP(&dbName, "db-name", "d", "", "db name of mysql")
	tokenCmd.PersistentFlags().StringVarP(&playerAddress, "address", "a", "", "player address")
	tokenCmd.AddCommand(tokenSetCmd)
	tokenCmd.AddCommand(tokenGetCmd)

	tokenSetCmd.Flags().Int64VarP(&points, "points", "p", 0, "points to set")
	tokenSetCmd.Flags().Int64VarP(&tokens, "tokens", "t", 0, "tokens to set")
}
