package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/redis"
	"github.com/CryptoElementals/common/server"
	"github.com/CryptoElementals/common/session"
)

type TestConfig struct {
	LogCfg    log.Config    `mapstructure:"log"`
	RedisCfg  redis.Config  `mapstructure:"redis"`
	ServerCfg server.Config `mapstructure:"server"`
}

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Printf("usage: %s <config-file-path>\n", args[0])
		os.Exit(1)
	}
	cfgPath := args[1]
	cfg := &TestConfig{}
	err := config.InitConfig(cfgPath, cfg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("init config done, %+v", cfg)
	err = log.InitGlobalLogger(&cfg.LogCfg)
	if err != nil {
		panic(err)
	}
	fmt.Println("init logger done")
	err = redis.Init(&cfg.RedisCfg)
	if err != nil {
		log.Fatal(err)
	}
	pool, err := redis.GetRedigoPool()
	if err != nil {
		log.Fatal(err)
	}
	sessionStore, err := session.New(pool)
	if err != nil {
		log.Fatal(err)
	}

	redisCache, err := cache.NewRedisCache()
	if err != nil {
		log.Fatal(err)
	}
	log.Info("starting test server")
	svr := server.New(&cfg.ServerCfg, sessionStore, redisCache)
	svr.Run()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Info("got os signal, quitting...")
	err = svr.Stop()
	if err != nil {
		log.Errorf("cannot close server: %s", err.Error())
	}
}
