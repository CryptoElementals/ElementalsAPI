package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/spf13/cobra"
)

const defaultBotRegistryNamespace = "lobby:v1"

var (
	botRegistryConfigPath string
	botRegistryNamespace  string
)

var botRegistryCmd = &cobra.Command{
	Use:   "bot-registry",
	Short: "Manage lobby bot registry in Redis",
}

var botRegistryInspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect bot registry state",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initBotRegistryRuntime(); err != nil {
			fmt.Printf("Failed to initialize bot registry tools: %v\n", err)
			os.Exit(1)
		}

		freshnessSec, _ := cmd.Flags().GetInt64("freshness-sec")
		freshnessMs := freshnessSec * 1000
		nowMs := time.Now().UnixMilli()

		keys := botRegistryKeys(botRegistryNamespace)
		allBots, err := redis.SMembers(keys.allKey)
		if err != nil {
			fmt.Printf("Failed to read all bots set: %v\n", err)
			os.Exit(1)
		}
		idleBots, err := redis.SMembers(keys.idleKey)
		if err != nil {
			fmt.Printf("Failed to read idle bots set: %v\n", err)
			os.Exit(1)
		}
		inGameBots, err := redis.SMembers(keys.inGameKey)
		if err != nil {
			fmt.Printf("Failed to read in-game bots set: %v\n", err)
			os.Exit(1)
		}

		idleSet := make(map[string]struct{}, len(idleBots))
		for _, b := range idleBots {
			idleSet[b] = struct{}{}
		}
		inGameSet := make(map[string]struct{}, len(inGameBots))
		for _, b := range inGameBots {
			inGameSet[b] = struct{}{}
		}
		sort.Strings(allBots)

		fmt.Printf("namespace: %s\n", botRegistryNamespace)
		fmt.Printf("all=%d idle=%d ingame=%d freshness_sec=%d\n", len(allBots), len(idleBots), len(inGameBots), freshnessSec)
		fmt.Println("----")

		for _, botKey := range allBots {
			var parsed types.PlayerAddress
			parseErr := parsed.Parse(botKey)
			hasSeen, err := redis.ZScoreMemberExists(keys.lastSeenKey, botKey)
			if err != nil {
				fmt.Printf("%s status=error err=%v\n", botKey, err)
				continue
			}
			lastSeen := int64(0)
			if hasSeen {
				score, scoreErr := redis.ZScore(keys.lastSeenKey, botKey)
				if scoreErr != nil {
					fmt.Printf("%s status=error err=%v\n", botKey, scoreErr)
					continue
				}
				lastSeen = int64(score)
			}
			fresh := hasSeen && lastSeen >= nowMs-freshnessMs
			state := "orphan"
			if _, ok := inGameSet[botKey]; ok {
				state = "ingame"
			} else if _, ok := idleSet[botKey]; ok {
				state = "idle"
			}
			if parseErr != nil {
				fmt.Printf("%s state=%s fresh=%t last_seen_ms=%d parse_err=%v\n", botKey, state, fresh, lastSeen, parseErr)
				continue
			}
			fmt.Printf("%d_%s state=%s fresh=%t last_seen_ms=%d\n", parsed.Id, parsed.TemporaryAddress, state, fresh, lastSeen)
		}
	},
}

var botRegistryAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add bot manually into registry",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initBotRegistryRuntime(); err != nil {
			fmt.Printf("Failed to initialize bot registry tools: %v\n", err)
			os.Exit(1)
		}
		addr, err := readSinglePlayerAddress(cmd)
		if err != nil {
			fmt.Printf("Invalid player: %v\n", err)
			os.Exit(1)
		}
		nowMs, _ := cmd.Flags().GetInt64("last-seen-ms")
		if nowMs <= 0 {
			nowMs = time.Now().UnixMilli()
		}
		keys := botRegistryKeys(botRegistryNamespace)
		key := addr.String()
		if _, err := redis.SAdd(keys.allKey, key); err != nil {
			fmt.Printf("Failed to add to all set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.SAdd(keys.idleKey, key); err != nil {
			fmt.Printf("Failed to add to idle set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.SRem(keys.inGameKey, key); err != nil {
			fmt.Printf("Failed to remove from in-game set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.SRem(keys.tokenInsufficientKey, key); err != nil {
			fmt.Printf("Failed to remove from token-insufficient set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.ZAdd(keys.lastSeenKey, float64(nowMs), key); err != nil {
			fmt.Printf("Failed to set last_seen zset score: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Added bot %s as idle with last_seen_ms=%d\n", key, nowMs)
	},
}

var botRegistryRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove bot manually from registry",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initBotRegistryRuntime(); err != nil {
			fmt.Printf("Failed to initialize bot registry tools: %v\n", err)
			os.Exit(1)
		}
		addr, err := readSinglePlayerAddress(cmd)
		if err != nil {
			fmt.Printf("Invalid player: %v\n", err)
			os.Exit(1)
		}
		keys := botRegistryKeys(botRegistryNamespace)
		key := addr.String()
		if _, err := redis.SRem(keys.idleKey, key); err != nil {
			fmt.Printf("Failed to remove from idle set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.SRem(keys.inGameKey, key); err != nil {
			fmt.Printf("Failed to remove from in-game set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.SRem(keys.tokenInsufficientKey, key); err != nil {
			fmt.Printf("Failed to remove from token-insufficient set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.SRem(keys.allKey, key); err != nil {
			fmt.Printf("Failed to remove from all set: %v\n", err)
			os.Exit(1)
		}
		if _, err := redis.ZRem(keys.lastSeenKey, key); err != nil {
			fmt.Printf("Failed to remove from last_seen zset: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed bot %s from registry\n", key)
	},
}

var botRegistryUpdateStatusCmd = &cobra.Command{
	Use:   "update-status",
	Short: "Update bot status manually",
	Long:  "status can be: idle, ingame, touch, stopping",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initBotRegistryRuntime(); err != nil {
			fmt.Printf("Failed to initialize bot registry tools: %v\n", err)
			os.Exit(1)
		}
		addr, err := readSinglePlayerAddress(cmd)
		if err != nil {
			fmt.Printf("Invalid player: %v\n", err)
			os.Exit(1)
		}
		status, _ := cmd.Flags().GetString("status")
		status = strings.ToLower(strings.TrimSpace(status))
		if status == "" {
			fmt.Println("status is required: idle|ingame|touch|stopping")
			os.Exit(1)
		}

		lastSeenMs, _ := cmd.Flags().GetInt64("last-seen-ms")
		if lastSeenMs <= 0 {
			lastSeenMs = time.Now().UnixMilli()
		}

		keys := botRegistryKeys(botRegistryNamespace)
		key := addr.String()
		if _, err := redis.SAdd(keys.allKey, key); err != nil {
			fmt.Printf("Failed to ensure all set membership: %v\n", err)
			os.Exit(1)
		}

		switch status {
		case "idle":
			if _, err := redis.SAdd(keys.idleKey, key); err != nil {
				fmt.Printf("Failed to add idle membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.inGameKey, key); err != nil {
				fmt.Printf("Failed to remove in-game membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.tokenInsufficientKey, key); err != nil {
				fmt.Printf("Failed to remove token-insufficient membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.ZAdd(keys.lastSeenKey, float64(lastSeenMs), key); err != nil {
				fmt.Printf("Failed to set last_seen score: %v\n", err)
				os.Exit(1)
			}
		case "ingame":
			if _, err := redis.SRem(keys.idleKey, key); err != nil {
				fmt.Printf("Failed to remove idle membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SAdd(keys.inGameKey, key); err != nil {
				fmt.Printf("Failed to add in-game membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.tokenInsufficientKey, key); err != nil {
				fmt.Printf("Failed to remove token-insufficient membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.ZAdd(keys.lastSeenKey, float64(lastSeenMs), key); err != nil {
				fmt.Printf("Failed to set last_seen score: %v\n", err)
				os.Exit(1)
			}
		case "touch":
			if _, err := redis.ZAdd(keys.lastSeenKey, float64(lastSeenMs), key); err != nil {
				fmt.Printf("Failed to set last_seen score: %v\n", err)
				os.Exit(1)
			}
		case "stopping":
			if _, err := redis.SRem(keys.idleKey, key); err != nil {
				fmt.Printf("Failed to remove idle membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.inGameKey, key); err != nil {
				fmt.Printf("Failed to remove in-game membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.tokenInsufficientKey, key); err != nil {
				fmt.Printf("Failed to remove token-insufficient membership: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.ZAdd(keys.lastSeenKey, 1, key); err != nil {
				fmt.Printf("Failed to set stopping marker: %v\n", err)
				os.Exit(1)
			}
			lastSeenMs = 1
		default:
			fmt.Printf("unsupported status %q, choose idle|ingame|touch|stopping\n", status)
			os.Exit(1)
		}
		fmt.Printf("Updated bot %s status=%s last_seen_ms=%d\n", key, status, lastSeenMs)
	},
}

var botRegistryCleanLastSeenGreaterThanCmd = &cobra.Command{
	Use:   "clean-last-seen-gt <last_seen_ms>",
	Short: "Remove all bots with last_seen_ms greater than the provided value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := initBotRegistryRuntime(); err != nil {
			fmt.Printf("Failed to initialize bot registry tools: %v\n", err)
			os.Exit(1)
		}

		thresholdMs, err := strconv.ParseInt(strings.TrimSpace(args[0]), 10, 64)
		if err != nil {
			fmt.Printf("Invalid last_seen_ms %q: %v\n", args[0], err)
			os.Exit(1)
		}

		keys := botRegistryKeys(botRegistryNamespace)
		totalRemoved := 0

		for {
			members, err := redis.ZRangeByScore(keys.lastSeenKey, fmt.Sprintf("(%d", thresholdMs), "+inf", 0, 1000)
			if err != nil {
				fmt.Printf("Failed to query last_seen zset: %v\n", err)
				os.Exit(1)
			}
			if len(members) == 0 {
				break
			}

			memberArgs := make([]interface{}, 0, len(members))
			for _, m := range members {
				memberArgs = append(memberArgs, m)
			}

			if _, err := redis.SRem(keys.allKey, memberArgs...); err != nil {
				fmt.Printf("Failed to remove from all set: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.idleKey, memberArgs...); err != nil {
				fmt.Printf("Failed to remove from idle set: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.inGameKey, memberArgs...); err != nil {
				fmt.Printf("Failed to remove from in-game set: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.SRem(keys.tokenInsufficientKey, memberArgs...); err != nil {
				fmt.Printf("Failed to remove from token-insufficient set: %v\n", err)
				os.Exit(1)
			}
			if _, err := redis.ZRem(keys.lastSeenKey, memberArgs...); err != nil {
				fmt.Printf("Failed to remove from last_seen zset: %v\n", err)
				os.Exit(1)
			}

			totalRemoved += len(members)
		}

		fmt.Printf("Removed %d bots with last_seen_ms > %d\n", totalRemoved, thresholdMs)
	},
}

type botRegistryRedisKeys struct {
	allKey                string
	idleKey               string
	inGameKey             string
	lastSeenKey           string
	tokenInsufficientKey  string
}

func botRegistryKeys(namespace string) botRegistryRedisKeys {
	return botRegistryRedisKeys{
		allKey:               namespace + ":bots:all:set",
		idleKey:              namespace + ":bots:idle:set",
		inGameKey:            namespace + ":bots:ingame:set",
		lastSeenKey:          namespace + ":bots:last_seen:zset",
		tokenInsufficientKey: namespace + ":bots:token_insufficient:set",
	}
}

func initBotRegistryRuntime() error {
	if err := config.InitToolsConfig(botRegistryConfigPath); err != nil {
		return fmt.Errorf("load tools config: %w", err)
	}
	if botRegistryNamespace == defaultBotRegistryNamespace && config.ToolsGConf.BotRegistry.Namespace != "" {
		botRegistryNamespace = config.ToolsGConf.BotRegistry.Namespace
	}
	logCfg := &log.Config{Level: "info", Development: false}
	if err := log.InitGlobalLogger(logCfg); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	rc := config.ToolsGConf.RedisCfg
	if rc.Address == "" {
		return fmt.Errorf("redis.address is required in tools config")
	}
	if rc.Size == 0 {
		rc.Size = 10
	}
	if err := redis.Init(&rc); err != nil {
		return fmt.Errorf("init redis: %w", err)
	}
	return nil
}

func readSinglePlayerAddress(cmd *cobra.Command) (types.PlayerAddress, error) {
	var out types.PlayerAddress
	key, _ := cmd.Flags().GetString("player-key")
	playerID, _ := cmd.Flags().GetInt64("player-id")
	tempAddr, _ := cmd.Flags().GetString("temporary-address")

	if strings.TrimSpace(key) != "" {
		if err := out.Parse(strings.TrimSpace(key)); err != nil {
			return types.PlayerAddress{}, err
		}
		return out, nil
	}
	if playerID <= 0 || strings.TrimSpace(tempAddr) == "" {
		return types.PlayerAddress{}, fmt.Errorf("provide either --player-key or both --player-id and --temporary-address")
	}
	return *types.NewPlayerAddress(playerID, tempAddr), nil
}

func init() {
	rootCmd.AddCommand(botRegistryCmd)
	botRegistryCmd.PersistentFlags().StringVarP(&botRegistryConfigPath, "config", "c", "", "tools config path")
	botRegistryCmd.PersistentFlags().StringVar(&botRegistryNamespace, "namespace", defaultBotRegistryNamespace, "redis namespace prefix")
	botRegistryCmd.MarkPersistentFlagRequired("config")

	botRegistryCmd.AddCommand(botRegistryInspectCmd)
	botRegistryCmd.AddCommand(botRegistryAddCmd)
	botRegistryCmd.AddCommand(botRegistryRemoveCmd)
	botRegistryCmd.AddCommand(botRegistryUpdateStatusCmd)
	botRegistryCmd.AddCommand(botRegistryCleanLastSeenGreaterThanCmd)

	botRegistryInspectCmd.Flags().Int64("freshness-sec", 20, "freshness threshold in seconds")
	botRegistryInspectCmd.PreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("freshness-sec") {
			return
		}
		if botRegistryConfigPath == "" {
			return
		}
		if err := config.InitToolsConfig(botRegistryConfigPath); err != nil {
			return
		}
		if config.ToolsGConf.BotRegistry.FreshnessSec > 0 {
			_ = cmd.Flags().Set("freshness-sec", fmt.Sprintf("%d", config.ToolsGConf.BotRegistry.FreshnessSec))
		}
	}

	for _, c := range []*cobra.Command{botRegistryAddCmd, botRegistryRemoveCmd, botRegistryUpdateStatusCmd} {
		c.Flags().String("player-key", "", "player key in format <id>_<temporary_address>")
		c.Flags().Int64("player-id", 0, "player id (used with --temporary-address)")
		c.Flags().String("temporary-address", "", "temporary address (used with --player-id)")
	}

	botRegistryAddCmd.Flags().Int64("last-seen-ms", 0, "last seen timestamp in unix milliseconds (default now)")
	botRegistryUpdateStatusCmd.Flags().String("status", "", "new status: idle|ingame|touch|stopping")
	botRegistryUpdateStatusCmd.Flags().Int64("last-seen-ms", 0, "last seen timestamp in unix milliseconds (default now, ignored for stopping)")
}
