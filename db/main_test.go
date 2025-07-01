package db

import (
	"testing"

	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"go.uber.org/zap"
)

func TestMain(m *testing.M) {
	cfg := &Config{}
	config.InitConfig("../../config.yaml", cfg)
	log.InitGlobalLogger(&log.Config{
		Level:       "debug",
		Development: true,
	}, zap.AddCallerSkip(1), zap.AddStacktrace(zap.DebugLevel))
	Init(cfg)
	m.Run()
}
