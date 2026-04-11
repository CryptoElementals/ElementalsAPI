package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/spf13/cobra"
)

const defaultQueueNamespace = "lobby:v1"

var (
	queueConfigPath string
	queueNamespace  string
)

type queueRedisKeys struct {
	queueZSet  string
	pendingMap string
	inGameSet  string
}

func queueKeys(namespace string) queueRedisKeys {
	return queueRedisKeys{
		queueZSet:  namespace + ":queue:zset",
		pendingMap: namespace + ":pending:hash",
		inGameSet:  namespace + ":ingame:set",
	}
}

func initQueueRuntime() error {
	if err := config.InitToolsConfig(queueConfigPath); err != nil {
		return fmt.Errorf("load tools config: %w", err)
	}
	if queueNamespace == defaultQueueNamespace && config.ToolsGConf.Queue.Namespace != "" {
		queueNamespace = config.ToolsGConf.Queue.Namespace
	}
	if err := log.InitGlobalLogger(&log.Config{Level: "info", Development: false}); err != nil {
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

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Inspect and manage lobby queue data in Redis",
}

var queueInspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect queue, match-pending, and in-game data",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initQueueRuntime(); err != nil {
			fmt.Printf("Failed to initialize queue tools: %v\n", err)
			os.Exit(1)
		}
		keys := queueKeys(queueNamespace)

		queueMembers, err := redis.ZRange(keys.queueZSet, 0, -1)
		if err != nil {
			fmt.Printf("Failed to read queue zset: %v\n", err)
			os.Exit(1)
		}
		pendingMembers, err := redis.HGetAll(keys.pendingMap)
		if err != nil {
			fmt.Printf("Failed to read pending hash: %v\n", err)
			os.Exit(1)
		}
		inGameMembers, err := redis.SMembers(keys.inGameSet)
		if err != nil {
			fmt.Printf("Failed to read in-game set: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("namespace: %s\n", queueNamespace)
		fmt.Printf("queue=%d pending_players=%d ingame=%d\n", len(queueMembers), len(pendingMembers), len(inGameMembers))
		fmt.Println("---- queue")
		for _, member := range queueMembers {
			score, scoreErr := redis.ZScore(keys.queueZSet, member)
			if scoreErr != nil {
				fmt.Printf("%s score=error(%v)\n", member, scoreErr)
				continue
			}
			fmt.Printf("%s queued_at_ms=%d\n", member, int64(score))
		}

		fmt.Println("---- match-pendings")
		printPendingMembers(pendingMembers)

		fmt.Println("---- in-game")
		sort.Strings(inGameMembers)
		for _, member := range inGameMembers {
			fmt.Println(member)
		}
	},
}

var queueInspectPendingCmd = &cobra.Command{
	Use:   "inspect-pendings",
	Short: "Inspect current match-pendings",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initQueueRuntime(); err != nil {
			fmt.Printf("Failed to initialize queue tools: %v\n", err)
			os.Exit(1)
		}
		keys := queueKeys(queueNamespace)
		pendingMembers, err := redis.HGetAll(keys.pendingMap)
		if err != nil {
			fmt.Printf("Failed to read pending hash: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("namespace: %s\n", queueNamespace)
		fmt.Printf("pending_players=%d\n", len(pendingMembers))
		fmt.Println("----")
		printPendingMembers(pendingMembers)
	},
}

var queueInspectInGameCmd = &cobra.Command{
	Use:   "inspect-ingame",
	Short: "Inspect current in-game map",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initQueueRuntime(); err != nil {
			fmt.Printf("Failed to initialize queue tools: %v\n", err)
			os.Exit(1)
		}
		keys := queueKeys(queueNamespace)
		inGameMembers, err := redis.SMembers(keys.inGameSet)
		if err != nil {
			fmt.Printf("Failed to read in-game set: %v\n", err)
			os.Exit(1)
		}
		sort.Strings(inGameMembers)
		fmt.Printf("namespace: %s\n", queueNamespace)
		fmt.Printf("ingame=%d\n", len(inGameMembers))
		fmt.Println("----")
		for _, member := range inGameMembers {
			fmt.Println(member)
		}
	},
}

var queueRemovePendingsCmd = &cobra.Command{
	Use:   "remove-pendings",
	Short: "Remove match-pending entries",
	Long:  "Remove pending entries by --match-id, by one player (--player-key or --player-id + --temporary-address), or all via --all",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initQueueRuntime(); err != nil {
			fmt.Printf("Failed to initialize queue tools: %v\n", err)
			os.Exit(1)
		}
		keys := queueKeys(queueNamespace)

		removeAll, _ := cmd.Flags().GetBool("all")
		matchID, _ := cmd.Flags().GetInt64("match-id")
		playerKey, _ := cmd.Flags().GetString("player-key")
		playerID, _ := cmd.Flags().GetInt64("player-id")
		tempAddr, _ := cmd.Flags().GetString("temporary-address")
		hasPlayerSelector := strings.TrimSpace(playerKey) != "" || (playerID > 0 && strings.TrimSpace(tempAddr) != "")

		selected := 0
		if removeAll {
			selected++
		}
		if matchID > 0 {
			selected++
		}
		if hasPlayerSelector {
			selected++
		}
		if selected != 1 {
			fmt.Println("Choose exactly one mode: --all OR --match-id OR player selector (--player-key OR --player-id + --temporary-address)")
			os.Exit(1)
		}

		if removeAll {
			allPending, err := redis.HGetAll(keys.pendingMap)
			if err != nil {
				fmt.Printf("Failed to read pending hash: %v\n", err)
				os.Exit(1)
			}
			fields := make([]string, 0, len(allPending))
			for k := range allPending {
				fields = append(fields, k)
			}
			if len(fields) == 0 {
				fmt.Println("No pending entries to remove")
				return
			}
			removed, err := redis.HDel(keys.pendingMap, fields...)
			if err != nil {
				fmt.Printf("Failed to remove pending entries: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Removed %d pending field(s)\n", removed)
			return
		}

		if matchID > 0 {
			allPending, err := redis.HGetAll(keys.pendingMap)
			if err != nil {
				fmt.Printf("Failed to read pending hash: %v\n", err)
				os.Exit(1)
			}
			matchIDStr := strconv.FormatInt(matchID, 10)
			fields := make([]string, 0)
			for k, v := range allPending {
				if v == matchIDStr {
					fields = append(fields, k)
				}
			}
			if len(fields) == 0 {
				fmt.Printf("No pending entries found for match-id=%d\n", matchID)
				return
			}
			removed, err := redis.HDel(keys.pendingMap, fields...)
			if err != nil {
				fmt.Printf("Failed to remove pending entries for match-id=%d: %v\n", matchID, err)
				os.Exit(1)
			}
			fmt.Printf("Removed %d pending field(s) for match-id=%d\n", removed, matchID)
			return
		}

		addr, err := readSinglePlayerAddress(cmd)
		if err != nil {
			fmt.Printf("Invalid player: %v\n", err)
			os.Exit(1)
		}
		removed, err := redis.HDel(keys.pendingMap, addr.String())
		if err != nil {
			fmt.Printf("Failed to remove pending entry for %s: %v\n", addr.String(), err)
			os.Exit(1)
		}
		fmt.Printf("Removed %d pending field(s) for player=%s\n", removed, addr.String())
	},
}

var queueRemoveQueuePlayerCmd = &cobra.Command{
	Use:   "remove-queue-player",
	Short: "Remove player from queue zset",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initQueueRuntime(); err != nil {
			fmt.Printf("Failed to initialize queue tools: %v\n", err)
			os.Exit(1)
		}
		addr, err := readSinglePlayerAddress(cmd)
		if err != nil {
			fmt.Printf("Invalid player: %v\n", err)
			os.Exit(1)
		}
		keys := queueKeys(queueNamespace)
		removed, err := redis.ZRem(keys.queueZSet, addr.String())
		if err != nil {
			fmt.Printf("Failed to remove player from queue: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed %d queue member(s) for player=%s\n", removed, addr.String())
	},
}

var queueRemoveInGamePlayerCmd = &cobra.Command{
	Use:   "remove-ingame-player",
	Short: "Remove player from in-game map",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initQueueRuntime(); err != nil {
			fmt.Printf("Failed to initialize queue tools: %v\n", err)
			os.Exit(1)
		}
		addr, err := readSinglePlayerAddress(cmd)
		if err != nil {
			fmt.Printf("Invalid player: %v\n", err)
			os.Exit(1)
		}
		keys := queueKeys(queueNamespace)
		removed, err := redis.SRem(keys.inGameSet, addr.String())
		if err != nil {
			fmt.Printf("Failed to remove player from in-game map: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed %d in-game member(s) for player=%s\n", removed, addr.String())
	},
}

func printPendingMembers(pendingMembers map[string]string) {
	if len(pendingMembers) == 0 {
		fmt.Println("(empty)")
		return
	}
	type pendingItem struct {
		playerKey string
		matchID   int64
		raw       string
	}
	items := make([]pendingItem, 0, len(pendingMembers))
	for k, v := range pendingMembers {
		matchID, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			items = append(items, pendingItem{playerKey: k, matchID: 0, raw: v})
			continue
		}
		items = append(items, pendingItem{playerKey: k, matchID: matchID, raw: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].matchID == items[j].matchID {
			return items[i].playerKey < items[j].playerKey
		}
		return items[i].matchID < items[j].matchID
	})
	for _, it := range items {
		if it.matchID == 0 && it.raw != "0" {
			fmt.Printf("%s match_id=%s(parse_error)\n", it.playerKey, it.raw)
			continue
		}
		fmt.Printf("%s match_id=%d\n", it.playerKey, it.matchID)
	}
}

func init() {
	rootCmd.AddCommand(queueCmd)
	queueCmd.PersistentFlags().StringVarP(&queueConfigPath, "config", "c", "", "tools config path")
	queueCmd.PersistentFlags().StringVar(&queueNamespace, "namespace", defaultQueueNamespace, "redis namespace prefix")
	queueCmd.MarkPersistentFlagRequired("config")

	queueCmd.AddCommand(queueInspectCmd)
	queueCmd.AddCommand(queueInspectPendingCmd)
	queueCmd.AddCommand(queueInspectInGameCmd)
	queueCmd.AddCommand(queueRemovePendingsCmd)
	queueCmd.AddCommand(queueRemoveQueuePlayerCmd)
	queueCmd.AddCommand(queueRemoveInGamePlayerCmd)

	for _, c := range []*cobra.Command{queueRemovePendingsCmd, queueRemoveQueuePlayerCmd, queueRemoveInGamePlayerCmd} {
		c.Flags().String("player-key", "", "player key in format <id>_<temporary_address>")
		c.Flags().Int64("player-id", 0, "player id (used with --temporary-address)")
		c.Flags().String("temporary-address", "", "temporary address (used with --player-id)")
	}
	queueRemovePendingsCmd.Flags().Int64("match-id", 0, "remove pending entries that belong to this match id")
	queueRemovePendingsCmd.Flags().Bool("all", false, "remove all pending entries")
}
