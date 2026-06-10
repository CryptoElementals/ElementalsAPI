package main

import (
	"fmt"
	"math/big"
	"os"
	"time"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/CryptoElementals/common/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

var (
	contractInspectConfigPath string
	contractInspectRPC        string
	contractInspectAddress    string
	contractInspectGameID     int64
)

var contractCmd = &cobra.Command{
	Use:   "contract",
	Short: "Contract inspection tools",
}

var contractInspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect RoomV3 roomData by game id",
	Run: func(cmd *cobra.Command, args []string) {
		if err := runContractInspect(); err != nil {
			fmt.Printf("inspect contract failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(contractCmd)
	contractCmd.AddCommand(contractInspectCmd)

	contractInspectCmd.Flags().StringVarP(&contractInspectConfigPath, "config", "c", "", "chain-server config file path")
	contractInspectCmd.Flags().StringVar(&contractInspectRPC, "rpc", "", "chain HTTP RPC endpoint")
	contractInspectCmd.Flags().StringVar(&contractInspectAddress, "contract-address", "", "RoomV3 contract address")
	contractInspectCmd.Flags().Int64VarP(&contractInspectGameID, "game-id", "i", 0, "game id")
	contractInspectCmd.MarkFlagRequired("game-id")
}

func runContractInspect() error {
	if contractInspectConfigPath != "" {
		if err := config.InitCSConfig(contractInspectConfigPath); err != nil {
			return fmt.Errorf("load chain-server config: %w", err)
		}
		chains := config.CSGConf.EffectiveChains()
		if contractInspectRPC == "" && len(chains) > 0 {
			contractInspectRPC = chains[0].HttpRpc
		}
		if contractInspectAddress == "" && len(chains) > 0 {
			contractInspectAddress = chains[0].RoomV3ContractAddress
		}
	}

	if contractInspectRPC == "" {
		return fmt.Errorf("rpc is required (flag --rpc or chain-server config chains[].node.http-rpc)")
	}
	if contractInspectAddress == "" {
		return fmt.Errorf("contract-address is required (flag --contract-address or chain-server config chains[].contract.room-v3-contract-address)")
	}
	if !common.IsHexAddress(contractInspectAddress) {
		return fmt.Errorf("invalid contract-address: %s", contractInspectAddress)
	}
	if contractInspectGameID <= 0 {
		return fmt.Errorf("game-id must be greater than 0")
	}

	client, err := ethclient.Dial(contractInspectRPC)
	if err != nil {
		return fmt.Errorf("dial rpc: %w", err)
	}
	defer client.Close()

	roomCtr, err := contract.NewRoomV3Contract(common.HexToAddress(contractInspectAddress), client)
	if err != nil {
		return fmt.Errorf("create RoomV3 contract client: %w", err)
	}

	roomData, err := roomCtr.RoomData(nil, big.NewInt(contractInspectGameID))
	if err != nil {
		return fmt.Errorf("call roomData: %w", err)
	}

	fmt.Printf("RoomData(game-id=%d)\n", contractInspectGameID)
	fmt.Printf("  current_round: %s\n", roomData.CurrentRound.String())
	fmt.Printf("  current_card_index: %s\n", roomData.CurrentCardIndex.String())
	fmt.Printf("  total_round: %s\n", roomData.TotalRound.String())
	fmt.Printf("  total_card_index: %s\n", roomData.TotalCardIndex.String())
	fmt.Printf("  creator: %s\n", roomData.Creator.Hex())
	fmt.Printf("  player1_id: %s\n", roomData.Player1Id.String())
	fmt.Printf("  player2_id: %s\n", roomData.Player2Id.String())
	fmt.Printf("  player1_temp: %s\n", roomData.Player1Temp.Hex())
	fmt.Printf("  player2_temp: %s\n", roomData.Player2Temp.Hex())
	fmt.Printf("  launch_time_unix: %s\n", roomData.LaunchTime.String())
	if roomData.LaunchTime.Sign() > 0 {
		fmt.Printf("  launch_time_utc: %s\n", time.Unix(roomData.LaunchTime.Int64(), 0).UTC().Format(time.RFC3339))
	}
	fmt.Printf("  round_timeout: %s\n", roomData.RoundTimeout.String())
	fmt.Printf("  initial_hp: %s\n", roomData.InitialHP.String())
	fmt.Printf("  tournament: %s\n", roomData.Tournament.String())
	fmt.Printf("  tier_no: %s\n", roomData.TierNo.String())

	return nil
}
