package config

import (
	"fmt"

	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/internal/tokenunits"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
)

var LedgerGConf = LedgerServerConfig{}

type LedgerServerConfig struct {
	LogCfg                       log.Config   `mapstructure:"log"`
	DbCfg                        db.Config    `mapstructure:"database"`
	RedisCfg                     redis.Config `mapstructure:"redis"`
	UChainCfg                    UChainConfig `mapstructure:"u-chain"`
	ListenPort                   int64        `mapstructure:"listen-port"`
	WithdrawAuditThresholdTokens int32        `mapstructure:"withdraw-audit-threshold-tokens"`
}

type UChainConfig struct {
	ChainServerAddress string `mapstructure:"chain-server-address"`
	ChainID            int64  `mapstructure:"chain-id"`
}

func InitLedgerServerConfig(configPath string) error {
	if err := InitConfig(configPath, &LedgerGConf); err != nil {
		return err
	}
	setLedgerServerDefaults(&LedgerGConf)
	if LedgerGConf.WithdrawAuditThresholdTokens < 0 {
		return fmt.Errorf("withdraw-audit-threshold-tokens must be non-negative")
	}
	return nil
}

func setLedgerServerDefaults(cfg *LedgerServerConfig) {
	if cfg == nil {
		return
	}
	if cfg.WithdrawAuditThresholdTokens <= 0 {
		cfg.WithdrawAuditThresholdTokens = tokenunits.DefaultWithdrawAuditThresholdTokens
	}
}

// ResolvedWithdrawAuditThreshold returns the configured withdraw audit threshold.
func (c *LedgerServerConfig) ResolvedWithdrawAuditThreshold() int32 {
	if c == nil {
		return tokenunits.DefaultWithdrawAuditThresholdTokens
	}
	return tokenunits.ResolveWithdrawAuditThreshold(c.WithdrawAuditThresholdTokens)
}
