/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	gameclient "github.com/CryptoElementals/common/game_client"
	"github.com/CryptoElementals/common/log"
	rpc "github.com/CryptoElementals/common/rpc/client"
	"github.com/CryptoElementals/common/wallet"
	"github.com/spf13/cobra"
)

// gameCmd represents the game command
var gameCmd = &cobra.Command{
	Use:   "game",
	Short: "game tools for room server testing",
}

var gameRunCmd = &cobra.Command{
	Use:   "run",
	Short: "game tools for room server testing",
	Run: func(cmd *cobra.Command, args []string) {
		err := startGame()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(gameCmd)
	gameCmd.AddCommand(gameRunCmd)

	gameRunCmd.Flags().StringVarP(&roomServerEndpoint, "room-server-endpoint", "r", "", "room server endpoint")
	gameRunCmd.Flags().StringVarP(&lobbyServerEndpoint, "lobby-server-endpoint", "l", "", "lobby server endpoint")
	gameRunCmd.Flags().Int64VarP(&playerId, "player-id", "p", 0, "player ID")
	gameRunCmd.Flags().StringVarP(&tempWalletPath, "temp-wallet-path", "t", "", "temp wallet path")
	gameRunCmd.MarkFlagRequired("room-server-endpoint")
	gameRunCmd.MarkFlagRequired("lobby-server-endpoint")
	gameRunCmd.MarkFlagRequired("player-id")
	gameRunCmd.MarkFlagRequired("temp-wallet-path")
}

// InteractiveCardProvider reads card from user input
type InteractiveCardProvider struct {
	scanner *bufio.Scanner
}

func NewInteractiveCardProvider(scanner *bufio.Scanner) *InteractiveCardProvider {
	return &InteractiveCardProvider{scanner: scanner}
}

func (p *InteractiveCardProvider) GetCard(round uint32, turn uint32) (uint32, error) {
	fmt.Printf("Round %d, Turn %d: Please enter card number (1-5): ", round, turn)
	if !p.scanner.Scan() {
		return 0, fmt.Errorf("failed to read input")
	}
	line := strings.TrimSpace(p.scanner.Text())
	cardNum, err := strconv.ParseUint(line, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid card number: %v", err)
	}

	if cardNum < 1 || cardNum > 5 {
		return 0, fmt.Errorf("card number must be between 1 and 5, got %d", cardNum)
	}
	fmt.Println("card number: ", cardNum)
	return uint32(cardNum), nil
}

func startGame() error {
	client, err := rpc.NewClient(roomServerEndpoint, lobbyServerEndpoint)
	if err != nil {
		return err
	}
	log.InitGlobalLogger(&log.Config{
		Development: true,
	})
	var wTemp *wallet.Wallet
	wTemp, err = wallet.LoadWallet(tempWalletPath)
	if err != nil {
		return err
	}
	fmt.Println("using temp account, address: ", wTemp.GetAddrHex())
	fmt.Println("using player ID: ", playerId)
	// Set up interactive card provider
	scanner := bufio.NewScanner(os.Stdin)
	cardProvider := NewInteractiveCardProvider(scanner)
	gameContext, err := gameclient.NewGameContext(context.Background(), playerId, wTemp, client, cardProvider)
	if err != nil {
		return err
	}
	err = gameContext.Subscribe()
	if err != nil {
		return err
	}
	err = gameContext.JoinQueue()
	if err != nil {
		return err
	}
	return gameContext.Run()
}
