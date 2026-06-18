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
	setAllTokensAmount       int32
	setAllTokensLobbyEndpoint string
	setAllTokensDryRun       bool
	setAllTokensLimit        int
	setAllTokensPlayerKey    string
)

var botRegistrySetAllTokensCmd = &cobra.Command{
	Use:   "set-all-tokens",
	Short: "Batch set token balance for all registered bots",
	Long: `Reads bots from the all Redis set, sets each player's token balance
via LobbyService.SetUserTokenAmount. Bots in token_insufficient are promoted to idle
after a successful token update; ingame and idle bots keep their registry state.`,
	Run: func(cmd *cobra.Command, args []string) {
		if setAllTokensAmount <= 0 {
			fmt.Println("--token-amount must be a positive integer")
			os.Exit(1)
		}

		if err := initBotRegistryRuntime(); err != nil {
			fmt.Printf("Failed to initialize bot registry tools: %v\n", err)
			os.Exit(1)
		}

		keys := botRegistryKeys(botRegistryNamespace)
		candidates, err := redis.SMembers(keys.allKey)
		if err != nil {
			fmt.Printf("Failed to read all bots set: %v\n", err)
			os.Exit(1)
		}
		sort.Strings(candidates)

		toProcess, skipped := filterBotRegistryKeys(candidates, strings.TrimSpace(setAllTokensPlayerKey), setAllTokensLimit)
		if skipped > 0 {
			fmt.Printf("skipped: player-key %q not in candidate list\n", setAllTokensPlayerKey)
		}

		var (
			okCount     int
			failedCount int
			processed   int
			lobbyClient proto.LobbyServiceClient
			closeLobby  func()
		)

		if !setAllTokensDryRun {
			var dialErr error
			lobbyClient, closeLobby, dialErr = dialLobbyGRPC(resolveLobbyEndpoint(setAllTokensLobbyEndpoint))
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

			if setAllTokensDryRun {
				fmt.Printf("player_key=%s player_id=%d status=dry-run token_amount=%d\n",
					botKey, parsed.Id, setAllTokensAmount)
				okCount++
				continue
			}

			inTokenInsufficient, err := redis.SIsMember(keys.tokenInsufficientKey, botKey)
			if err != nil {
				failedCount++
				fmt.Printf("player_key=%s player_id=%d status=failed err=check_token_insufficient:%v\n",
					botKey, parsed.Id, err)
				continue
			}

			callCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			tokensBefore := uint64(0)
			if beforeResp, beforeErr := lobbyClient.GetPlayerToken(callCtx, &proto.GetPlayerTokenRequest{Id: parsed.Id}); beforeErr == nil {
				tokensBefore = beforeResp.GetTokens()
			}
			resp, setErr := lobbyClient.SetUserTokenAmount(callCtx, &proto.SetUserTokenAmountRequest{
				PlayerID:    parsed.Id,
				TokenAmount: setAllTokensAmount,
			})
			cancel()
			if setErr != nil {
				failedCount++
				fmt.Printf("player_key=%s player_id=%d status=failed err=set_token:%v\n",
					botKey, parsed.Id, setErr)
				continue
			}

			status := "ok"
			if inTokenInsufficient {
				if err := promoteBotRegistryToIdle(keys, botKey, nowMs); err != nil {
					failedCount++
					fmt.Printf("player_key=%s player_id=%d tokens_before=%d tokens_after=%d status=failed err=promote_idle:%v\n",
						botKey, parsed.Id, tokensBefore, resp.GetTokens(), err)
					continue
				}
				status = "idle"
			}

			okCount++
			fmt.Printf("player_key=%s player_id=%d tokens_before=%d tokens_after=%d status=%s\n",
				botKey, parsed.Id, tokensBefore, resp.GetTokens(), status)
		}

		fmt.Printf("summary: candidates=%d processed=%d ok=%d skipped=%d failed=%d dry_run=%t\n",
			len(candidates), processed, okCount, skipped, failedCount, setAllTokensDryRun)
		if failedCount > 0 {
			os.Exit(1)
		}
	},
}

func init() {
	botRegistryCmd.AddCommand(botRegistrySetAllTokensCmd)

	botRegistrySetAllTokensCmd.Flags().Int32Var(&setAllTokensAmount, "token-amount", 0, "target token balance for each bot (required)")
	botRegistrySetAllTokensCmd.Flags().StringVar(&setAllTokensLobbyEndpoint, "lobby-endpoint", "", "lobby gRPC address (default: game.lobby-server-endpoint from config)")
	botRegistrySetAllTokensCmd.Flags().BoolVar(&setAllTokensDryRun, "dry-run", false, "list bots that would be updated without writing")
	botRegistrySetAllTokensCmd.Flags().IntVar(&setAllTokensLimit, "limit", 0, "maximum number of bots to process (0 = no limit)")
	botRegistrySetAllTokensCmd.Flags().StringVar(&setAllTokensPlayerKey, "player-key", "", "process only this bot key (must be in all set)")
	_ = botRegistrySetAllTokensCmd.MarkFlagRequired("token-amount")
}
