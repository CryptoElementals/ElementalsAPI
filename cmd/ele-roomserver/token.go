/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"fmt"
	"strings"

	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/spf13/cobra"
)

// playerCmd represents the addToken command
var playerCmd = &cobra.Command{
	Use:   "player",
	Short: "command to manage tokens, points and player infos",
}

var playerSetCmd = &cobra.Command{
	Use:   "player-set",
	Short: "set player info",
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
		playerAddress = strings.ToLower(playerAddress)
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
		if cmd.Flags().Changed("name") {
			profile := dao.UserProfile{
				Address:       playerAddress,
				Name:          playerName,
				AvatarURL:     playerAvatarUrl,
				BackgroundURL: backgroundUrl,
			}
			err = db.Get().Save(&profile).Error
			if err != nil {
				fmt.Printf("set token and points failed, err: %v\n", err)
				return
			}
		}
	},
}

var playerGetCmd = &cobra.Command{
	Use:   "player-get",
	Short: "display player info",
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

var tokenLockCmd = &cobra.Command{
	Use:   "token-lock",
	Short: "lock token for wallet address",
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
		ut.LockedTokens = append(ut.LockedTokens, &dao.LockedUserToken{
			TemporaryAddress: tempAddress,
			GameID:           gameID,
			TokenAmount:      int32(tokens),
		})
		err = db.SaveUserToken(ut)
		if err != nil {
			fmt.Printf("save locked tokens failed, err: %v\n", err)
			return
		}
		fmt.Printf("player address: %s, locked tokens: %d\n", ut.WalletAddress, ut.TokenAmount)
	},
}

var tokenUnlockCmd = &cobra.Command{
	Use:   "token-unlock",
	Short: "unlock token for wallet address",
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
		for _, lt := range ut.LockedTokens {
			if cmd.Flags().Changed("game-id") && lt.GameID == gameID {
				err = db.Get().Delete(lt).Error
				if err != nil {
					fmt.Printf("unlock token failed, game id: %d", gameID)
				}
				return
			}
			if cmd.Flags().Changed("temp-addr") && lt.TemporaryAddress == strings.ToLower(tempAddress) {
				err = db.Get().Delete(lt).Error
				if err != nil {
					fmt.Printf("unlock token failed, temp addr: %s", tempAddress)
				}
				return
			}
		}
		fmt.Println("no locked token found with the given condition")
	},
}

func init() {
	rootCmd.AddCommand(playerCmd)
	playerCmd.AddCommand(playerSetCmd)
	playerCmd.AddCommand(playerGetCmd)
	playerCmd.PersistentFlags().StringVarP(&endpoint, "endpoint", "e", "", "endpoint of mysql")
	playerCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "user of mysql")
	playerCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "password of mysql")
	playerCmd.PersistentFlags().StringVarP(&dbName, "db-name", "d", "", "db name of mysql")
	playerCmd.PersistentFlags().StringVarP(&playerAddress, "address", "a", "", "player wallet address")

	playerSetCmd.Flags().Int64VarP(&points, "points", "", 0, "points to set")
	playerSetCmd.Flags().Int64VarP(&tokens, "tokens", "", 0, "tokens to set")
	playerSetCmd.Flags().StringVarP(&playerName, "name", "", "", "player name")
	playerSetCmd.Flags().StringVarP(&playerAvatarUrl, "avatar", "", "", "player avatar url")
	playerSetCmd.Flags().StringVarP(&backgroundUrl, "background", "", "", "player avatar background")
	playerSetCmd.MarkFlagsRequiredTogether("avatar", "name", "background")

	tokenLockCmd.Flags().UintVarP(&gameID, "game-id", "i", 0, "game id")
	tokenLockCmd.Flags().StringVarP(&tempAddress, "temp-addr", "t", "", "temporary address for locking")
	tokenLockCmd.Flags().Int64VarP(&tokens, "tokens", "", 0, "tokens to lock")

	tokenUnlockCmd.PersistentFlags().UintVarP(&gameID, "game-id", "i", 0, "game id")
	tokenUnlockCmd.PersistentFlags().StringVarP(&tempAddress, "temp-addr", "t", "", "temporary address for locking")
	tokenUnlockCmd.MarkFlagsOneRequired("game-id", "temp-addr")
	tokenUnlockCmd.MarkFlagsMutuallyExclusive("game-id", "temp-addr")
}
