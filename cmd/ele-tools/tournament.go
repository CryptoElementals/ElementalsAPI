package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	dao "github.com/CryptoElementals/common/models"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"
)

var (
	tournamentSchedulingConfigPath string
	tournamentLobbyEndpoint        string

	tournamentDbEndpoint string
	tournamentDbUser     string
	tournamentDbPassword string
	tournamentDbName     string

	tournamentListHours float64
	tournamentID        string
)

var tournamentSchedulingCmd = &cobra.Command{
	Use:   "tournament",
	Short: "Control lobby tournament creation scheduling",
}

var tournamentSchedulingStatusCmd = &cobra.Command{
	Use:   "scheduling-status",
	Short: "Show whether tournament scheduling is enabled",
	Run: func(cmd *cobra.Command, args []string) {
		cli, closeFn, err := newLobbyClientForTournamentScheduling()
		if err != nil {
			fmt.Printf("Failed to init lobby client: %v\n", err)
			os.Exit(1)
		}
		defer closeFn()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := cli.GetTournamentSchedulingStatus(ctx, &emptypb.Empty{})
		if err != nil {
			fmt.Printf("GetTournamentSchedulingStatus failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("tournament_scheduling_enabled=%v\n", resp.GetEnabled())
	},
}

var tournamentSchedulingEnableCmd = &cobra.Command{
	Use:   "enable-scheduling",
	Short: "Enable creation of new tournaments",
	Run: func(cmd *cobra.Command, args []string) {
		setTournamentSchedulingEnabled(true)
	},
}

var tournamentSchedulingDisableCmd = &cobra.Command{
	Use:   "disable-scheduling",
	Short: "Disable creation of new tournaments",
	Run: func(cmd *cobra.Command, args []string) {
		setTournamentSchedulingEnabled(false)
	},
}

var tournamentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tournaments in a recent time window (all statuses)",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initTournamentDbFromConfig(); err != nil {
			fmt.Printf("Failed to init tournament db config: %v\n", err)
			os.Exit(1)
		}
		if tournamentListHours <= 0 {
			tournamentListHours = 1
		}
		now := time.Now().UTC()
		from := now.Add(-time.Duration(tournamentListHours * float64(time.Hour)))

		var rows []dao.Tournament
		if err := db.Get().
			Where("created_at >= ? AND created_at <= ?", from, now).
			Order("scheduled_start_at desc").
			Find(&rows).Error; err != nil {
			fmt.Printf("List tournaments failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("window_hours=%.2f total=%d\n", tournamentListHours, len(rows))
		for _, t := range rows {
			fmt.Printf("tournament_id=%s status=%s scheduled_start_at=%s registration_deadline=%s scheduled_end_deadline=%s entry_fee=%d\n",
				t.TournamentID,
				t.Status,
				t.ScheduledStartAt.UTC().Format(time.RFC3339),
				t.RegistrationDeadline.UTC().Format(time.RFC3339),
				t.ScheduledEndDeadline.UTC().Format(time.RFC3339),
				t.EntryFee,
			)
		}
	},
}

var tournamentShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show tournament details by tournament-id",
	Run: func(cmd *cobra.Command, args []string) {
		if err := initTournamentDbFromConfig(); err != nil {
			fmt.Printf("Failed to init tournament db config: %v\n", err)
			os.Exit(1)
		}
		if tournamentID == "" {
			fmt.Println("--tournament-id is required")
			os.Exit(1)
		}

		var t dao.Tournament
		if err := db.Get().Where("tournament_id = ?", tournamentID).First(&t).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				fmt.Printf("tournament not found: %s\n", tournamentID)
				os.Exit(1)
			}
			fmt.Printf("load tournament failed: %v\n", err)
			os.Exit(1)
		}
		var participants []dao.TournamentParticipant
		if err := db.Get().
			Where("tournament_id = ?", tournamentID).
			Order("created_at asc, id asc").
			Find(&participants).Error; err != nil {
			fmt.Printf("load participants failed: %v\n", err)
			os.Exit(1)
		}
		var rounds []dao.TournamentRound
		if err := db.Get().
			Where("tournament_id = ?", tournamentID).
			Order("round_no asc, id asc").
			Find(&rounds).Error; err != nil {
			fmt.Printf("load rounds failed: %v\n", err)
			os.Exit(1)
		}
		var matches []dao.TournamentMatch
		if err := db.Get().
			Where("tournament_id = ?", tournamentID).
			Order("round_no asc, match_no asc, id asc").
			Find(&matches).Error; err != nil {
			fmt.Printf("load matches failed: %v\n", err)
			os.Exit(1)
		}

		out := struct {
			Tournament   dao.Tournament              `json:"Tournament"`
			Rounds       []dao.TournamentRound       `json:"Rounds"`
			Participants []dao.TournamentParticipant `json:"Participants"`
			Matches      []dao.TournamentMatch       `json:"Matches"`
		}{
			Tournament:   t,
			Rounds:       rounds,
			Participants: participants,
			Matches:      matches,
		}
		fmt.Printf("%s\n", types.ToJsonLoggableIndent(out))
	},
}

func setTournamentSchedulingEnabled(enabled bool) {
	cli, closeFn, err := newLobbyClientForTournamentScheduling()
	if err != nil {
		fmt.Printf("Failed to init lobby client: %v\n", err)
		os.Exit(1)
	}
	defer closeFn()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := cli.SetTournamentScheduling(ctx, &proto.SetTournamentSchedulingRequest{Enabled: enabled})
	if err != nil {
		fmt.Printf("SetTournamentScheduling failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("tournament_scheduling_enabled=%v\n", resp.GetEnabled())
}

func newLobbyClientForTournamentScheduling() (proto.LobbyServiceClient, func(), error) {
	endpoint := tournamentLobbyEndpoint
	if endpoint == "" {
		if tournamentSchedulingConfigPath == "" {
			tournamentSchedulingConfigPath = configPath
		}
		if tournamentSchedulingConfigPath == "" {
			return nil, nil, fmt.Errorf("either --lobby-server-endpoint or --config is required")
		}
		if err := config.InitToolsConfig(tournamentSchedulingConfigPath); err != nil {
			return nil, nil, fmt.Errorf("load tools config: %w", err)
		}
		endpoint = config.ToolsGConf.Game.LobbyServerEndpoint
	}
	if endpoint == "" {
		return nil, nil, fmt.Errorf("lobby server endpoint is empty (flag --lobby-server-endpoint or tools config game.lobby-server-endpoint)")
	}

	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, fmt.Errorf("dial lobby server: %w", err)
	}
	return proto.NewLobbyServiceClient(conn), func() { _ = conn.Close() }, nil
}

func initTournamentDbFromConfig() error {
	if tournamentSchedulingConfigPath == "" {
		tournamentSchedulingConfigPath = configPath
	}
	if tournamentSchedulingConfigPath != "" {
		if err := config.InitToolsConfig(tournamentSchedulingConfigPath); err != nil {
			return fmt.Errorf("load tools config: %w", err)
		}
		if tournamentDbEndpoint == "" {
			tournamentDbEndpoint = config.ToolsGConf.DbCfg.Endpoint
		}
		if tournamentDbUser == "" {
			tournamentDbUser = config.ToolsGConf.DbCfg.User
		}
		if tournamentDbPassword == "" {
			tournamentDbPassword = config.ToolsGConf.DbCfg.Password
		}
		if tournamentDbName == "" {
			tournamentDbName = config.ToolsGConf.DbCfg.DbName
		}
	}
	if tournamentDbEndpoint == "" || tournamentDbUser == "" || tournamentDbName == "" {
		return fmt.Errorf("database endpoint/user/db-name are required (flags or tools config)")
	}
	return db.Init(&db.Config{
		Endpoint: tournamentDbEndpoint,
		User:     tournamentDbUser,
		Password: tournamentDbPassword,
		DbName:   tournamentDbName,
	})
}

func init() {
	rootCmd.AddCommand(tournamentSchedulingCmd)
	tournamentSchedulingCmd.PersistentFlags().StringVarP(&tournamentSchedulingConfigPath, "config", "c", "", "tools config path")
	tournamentSchedulingCmd.PersistentFlags().StringVarP(&tournamentLobbyEndpoint, "lobby-server-endpoint", "l", "", "lobby server endpoint")

	tournamentSchedulingCmd.AddCommand(tournamentSchedulingStatusCmd)
	tournamentSchedulingCmd.AddCommand(tournamentSchedulingEnableCmd)
	tournamentSchedulingCmd.AddCommand(tournamentSchedulingDisableCmd)
	tournamentSchedulingCmd.AddCommand(tournamentListCmd)
	tournamentSchedulingCmd.AddCommand(tournamentShowCmd)

	tournamentListCmd.Flags().Float64Var(&tournamentListHours, "hours", 1, "lookback window in hours")
	tournamentListCmd.Flags().StringVarP(&tournamentDbEndpoint, "endpoint", "e", "", "endpoint of mysql")
	tournamentListCmd.Flags().StringVarP(&tournamentDbUser, "user", "u", "", "user of mysql")
	tournamentListCmd.Flags().StringVarP(&tournamentDbPassword, "password", "p", "", "password of mysql")
	tournamentListCmd.Flags().StringVarP(&tournamentDbName, "db-name", "d", "", "db name of mysql")

	tournamentShowCmd.Flags().StringVar(&tournamentID, "tournament-id", "", "business tournament id")
	tournamentShowCmd.Flags().StringVarP(&tournamentDbEndpoint, "endpoint", "e", "", "endpoint of mysql")
	tournamentShowCmd.Flags().StringVarP(&tournamentDbUser, "user", "u", "", "user of mysql")
	tournamentShowCmd.Flags().StringVarP(&tournamentDbPassword, "password", "p", "", "password of mysql")
	tournamentShowCmd.Flags().StringVarP(&tournamentDbName, "db-name", "d", "", "db name of mysql")
}
