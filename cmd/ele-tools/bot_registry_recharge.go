package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
)

var (
	rechargeTokenAmount    int32
	rechargeLobbyEndpoint  string
	rechargeDryRun         bool
	rechargeLimit          int
	rechargePlayerKey      string
)

var botRegistryRechargeInsufficientCmd = &cobra.Command{
	Use:   "recharge-insufficient",
	Short: "Batch recharge token_insufficient bots and promote them to idle",
	Long: `Reads bots from the token_insufficient Redis set, sets each player's token balance
via LobbyService.SetUserTokenAmount, then promotes the bot registry entry to idle.`,
	Run: func(cmd *cobra.Command, args []string) {
		if rechargeTokenAmount <= 0 {
			fmt.Println("--token-amount must be a positive integer")
			os.Exit(1)
		}

		if err := initBotRegistryRuntime(); err != nil {
			fmt.Printf("Failed to initialize bot registry tools: %v\n", err)
			os.Exit(1)
		}

		keys := botRegistryKeys(botRegistryNamespace)
		candidates, err := redis.SMembers(keys.tokenInsufficientKey)
		if err != nil {
			fmt.Printf("Failed to read token-insufficient bots set: %v\n", err)
			os.Exit(1)
		}
		sort.Strings(candidates)

		toProcess, skipped := filterBotRegistryKeys(candidates, strings.TrimSpace(rechargePlayerKey), rechargeLimit)
		if skipped > 0 {
			fmt.Printf("skipped: player-key %q not in candidate list\n", rechargePlayerKey)
		}

		var (
			okCount      int
			failedCount  int
			processed    int
			lobbyClient  proto.LobbyServiceClient
			closeLobby   func()
		)

		if !rechargeDryRun {
			var dialErr error
			lobbyClient, closeLobby, dialErr = dialLobbyGRPC(resolveLobbyEndpoint(rechargeLobbyEndpoint))
			if dialErr != nil {
				fmt.Printf("%v\n", dialErr)
				os.Exit(1)
			}
			defer closeLobby()
		}

		nowMs := time.Now().UnixMilli()
		for _, botKey := range toProcess {
			processed++
			var parsed types.PlayerAddress
			if err := parsed.Parse(botKey); err != nil {
				failedCount++
				fmt.Printf("player_key=%s status=failed err=parse:%v\n", botKey, err)
				continue
			}

			if rechargeDryRun {
				fmt.Printf("player_key=%s player_id=%d status=dry-run token_amount=%d\n",
					botKey, parsed.Id, rechargeTokenAmount)
				okCount++
				continue
			}

			callCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			tokensBefore := uint64(0)
			if beforeResp, beforeErr := lobbyClient.GetPlayerToken(callCtx, &proto.GetPlayerTokenRequest{Id: parsed.Id}); beforeErr == nil {
				tokensBefore = beforeResp.GetTokens()
			}
			resp, setErr := lobbyClient.SetUserTokenAmount(callCtx, &proto.SetUserTokenAmountRequest{
				PlayerID:    parsed.Id,
				TokenAmount: rechargeTokenAmount,
			})
			cancel()
			if setErr != nil {
				failedCount++
				fmt.Printf("player_key=%s player_id=%d status=failed err=set_token:%v\n",
					botKey, parsed.Id, setErr)
				continue
			}

			if err := promoteBotRegistryToIdle(keys, botKey, nowMs); err != nil {
				failedCount++
				fmt.Printf("player_key=%s player_id=%d tokens_before=%d tokens_after=%d status=failed err=promote_idle:%v\n",
					botKey, parsed.Id, tokensBefore, resp.GetTokens(), err)
				continue
			}

			okCount++
			fmt.Printf("player_key=%s player_id=%d tokens_before=%d tokens_after=%d status=idle\n",
				botKey, parsed.Id, tokensBefore, resp.GetTokens())
		}

		fmt.Printf("summary: candidates=%d processed=%d ok=%d skipped=%d failed=%d dry_run=%t\n",
			len(candidates), processed, okCount, skipped, failedCount, rechargeDryRun)
		if failedCount > 0 {
			os.Exit(1)
		}
	},
}

func filterBotRegistryKeys(candidates []string, playerKeyFilter string, limit int) (filtered []string, skipped int) {
	if playerKeyFilter != "" {
		for _, k := range candidates {
			if k == playerKeyFilter {
				filtered = append(filtered, k)
				break
			}
		}
		if len(filtered) == 0 {
			return nil, 1
		}
	} else {
		filtered = append(filtered, candidates...)
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, 0
}

func init() {
	botRegistryCmd.AddCommand(botRegistryRechargeInsufficientCmd)

	botRegistryRechargeInsufficientCmd.Flags().Int32Var(&rechargeTokenAmount, "token-amount", 0, "target token balance for each bot (required)")
	botRegistryRechargeInsufficientCmd.Flags().StringVar(&rechargeLobbyEndpoint, "lobby-endpoint", "", "lobby gRPC address (default: game.lobby-server-endpoint from config)")
	botRegistryRechargeInsufficientCmd.Flags().BoolVar(&rechargeDryRun, "dry-run", false, "list bots that would be recharged without writing")
	botRegistryRechargeInsufficientCmd.Flags().IntVar(&rechargeLimit, "limit", 0, "maximum number of bots to process (0 = no limit)")
	botRegistryRechargeInsufficientCmd.Flags().StringVar(&rechargePlayerKey, "player-key", "", "process only this bot key (must be in token_insufficient set)")
	_ = botRegistryRechargeInsufficientCmd.MarkFlagRequired("token-amount")
}
