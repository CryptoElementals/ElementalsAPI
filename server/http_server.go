package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/CryptoElementals/common/server/api/login"
	"github.com/CryptoElementals/common/server/middlewares"
	"github.com/CryptoElementals/common/session"
	"github.com/gin-contrib/sessions"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/handler"
)

type Config struct {
	Port       int    `mapstructure:"port"`
	ServerMode string `mapstructure:"server-mode"`
}

type Web3FormationServer struct {
	e      *gin.Engine
	server *http.Server
	wg     *sync.WaitGroup
	cfg    *Config
}

func NewServer(cfg *Config, store *session.SessionStore) *Web3FormationServer {
	wg := &sync.WaitGroup{}
	r := newRouter(wg, cfg.ServerMode, store)
	return &Web3FormationServer{
		e:  r,
		wg: wg,
	}
}

func (s *Web3FormationServer) Run() {
	s.server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", s.cfg.Port),
		Handler: s.e,
	}
	go s.server.ListenAndServe()
}

func (s *Web3FormationServer) Stop() error {
	err := s.server.Shutdown(context.Background())
	// whether the err is nil, we should wait for wait group
	s.wg.Wait()
	return err
}

func newRouter(wg *sync.WaitGroup, serverMode string, store *session.SessionStore) *gin.Engine {
	mode := strings.ToLower(serverMode)
	switch mode {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(ginLogger())
	r.Use(gin.Recovery())
	r.Use(ginWaitGroup(wg))
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	//注册session 中间件
	r.Use(sessions.Sessions("dill_session", store))

	// register apis here
	r.POST("/", middlewares.PreJobMiddleware(), middlewares.AuthMiddleware(serverMode), handler.Handle)

	// enable login api
	login.UseWalletLogin(store.MaxAge())
	return r
}

const SERVER_WAIT_GROUP_LABEL = "x-gin-waitgroup"

func ginWaitGroup(wg *sync.WaitGroup) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(SERVER_WAIT_GROUP_LABEL, wg)
	}
}

func ginLogger() gin.HandlerFunc {
	w := log.GlobalLogger().Writer()
	return gin.LoggerWithWriter(w)
}
