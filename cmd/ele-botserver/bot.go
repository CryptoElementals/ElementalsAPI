package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/wallet"
	"github.com/spf13/cobra"
)

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Bot management commands",
}

var (
	botCreateName          string
	botCreatePlayerID      int64
	botCreateKeyDir        string
	botCreateDbEndpoint    string
	botCreateDbUser        string
	botCreateDbPassword    string
	botCreateDbName        string
	botCreateAvatarURL     string
	botCreateBackgroundURL string
	botCreateTokens        int32
	botCreatePoints        int32
)

var botCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a bot (user_profile + user_token + temp wallet)",
	Run: func(cmd *cobra.Command, args []string) {
		// Generate temp private key file for this bot
		if err := os.MkdirAll(botCreateKeyDir, 0o700); err != nil {
			fmt.Printf("create key directory failed: %v\n", err)
			os.Exit(1)
		}
		keyPath := filepath.Join(botCreateKeyDir, fmt.Sprintf("temp_%d.key", botCreatePlayerID))
		w, err := wallet.NewWallet(keyPath)
		if err != nil {
			fmt.Printf("generate temp wallet failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("generated temp wallet: address=%s, key_path=%s\n", w.GetAddrHex(), keyPath)

		_ = log.InitGlobalLogger(&log.Config{Level: "info", Development: false})
		if err := db.Init(&db.Config{Endpoint: botCreateDbEndpoint, User: botCreateDbUser, Password: botCreateDbPassword, DbName: botCreateDbName}); err != nil {
			fmt.Printf("init database failed: %v\n", err)
			os.Exit(1)
		}

		profile, token, err := db.CreateBot(botCreatePlayerID, botCreateName, botCreateAvatarURL, botCreateBackgroundURL, botCreateTokens, botCreatePoints)
		if err != nil {
			fmt.Printf("create bot failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("created bot: player_id=%d, name=%s, tokens=%d, points=%d\n",
			profile.PlayerID, profile.Name, token.TokenAmount, token.Points)
	},
}

func init() {
	rootCmd.AddCommand(botCmd)
	botCmd.AddCommand(botCreateCmd)

	botCreateCmd.Flags().StringVarP(&botCreateName, "name", "n", "", "bot name (unique), e.g. bot_1")
	botCreateCmd.Flags().Int64VarP(&botCreatePlayerID, "player-id", "i", 0, "bot player ID (int64)")
	botCreateCmd.Flags().StringVarP(&botCreateKeyDir, "key-dir", "k", "", "directory to save generated temp private key")
	botCreateCmd.Flags().StringVarP(&botCreateDbEndpoint, "db-endpoint", "e", "", "database endpoint (e.g. localhost:3306)")
	botCreateCmd.Flags().StringVarP(&botCreateDbUser, "db-user", "u", "", "database user")
	botCreateCmd.Flags().StringVarP(&botCreateDbPassword, "db-password", "w", "", "database password")
	botCreateCmd.Flags().StringVarP(&botCreateDbName, "db-name", "d", "", "database name")
	botCreateCmd.Flags().StringVarP(&botCreateAvatarURL, "avatar-url", "a", "avatar_1.png", "avatar URL for the bot profile")
	botCreateCmd.Flags().StringVarP(&botCreateBackgroundURL, "background-url", "b", "bg_1.png", "background URL for the bot profile")
	botCreateCmd.Flags().Int32VarP(&botCreateTokens, "tokens", "t", 20000, "default token amount for the bot")
	botCreateCmd.Flags().Int32VarP(&botCreatePoints, "points", "p", 1000, "default points for the bot")
	botCreateCmd.MarkFlagRequired("name")
	botCreateCmd.MarkFlagRequired("player-id")
	botCreateCmd.MarkFlagRequired("key-dir")
	botCreateCmd.MarkFlagRequired("db-endpoint")
	botCreateCmd.MarkFlagRequired("db-user")
	botCreateCmd.MarkFlagRequired("db-name")
}
