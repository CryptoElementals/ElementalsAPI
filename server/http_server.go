package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api/battle"
	"github.com/CryptoElementals/common/server/api/login"
	"github.com/CryptoElementals/common/server/api/match"
	"github.com/CryptoElementals/common/server/api/system"
	"github.com/CryptoElementals/common/server/api/user"
	"github.com/CryptoElementals/common/server/handler"
	"github.com/CryptoElementals/common/server/middlewares"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
)

const SERVER_WAIT_GROUP_LABEL = "x-gin-waitgroup"

type Server struct {
	e      *gin.Engine
	server *http.Server
	wg     *sync.WaitGroup
	cfg    *config.ServerConfig
}

func handleDefaultValue(cfg *config.ServerConfig) *config.ServerConfig {
	if cfg == nil {
		cfg = &config.ServerConfig{}
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}
	if cfg.ServerMode == "" {
		cfg.ServerMode = "development"
	}
	if cfg.SessionMaxAge == 0 {
		cfg.SessionMaxAge = 180
	}
	if cfg.RefreshTokenMaxAge == 0 {
		cfg.RefreshTokenMaxAge = 300
	}
	if cfg.ServiceName == "" {
		cfg.ServiceName = "test-server"
	}
	return cfg
}

func DefaultConfig() *config.ServerConfig {
	return handleDefaultValue(nil)
}

func DefaultSessionStore() sessions.Store {
	return memstore.NewStore([]byte("test-secret"))
}

func New(cfg *config.ServerConfig, store sessions.Store, refreshTokenCache cache.Cache) *Server {
	if cfg == nil {
		log.Fatal("nil config value")
	}
	if store == nil {
		log.Fatal("nil session store")
	}
	if refreshTokenCache == nil {
		log.Fatal("nil refresh token cache")
	}
	cfg = handleDefaultValue(cfg)
	wg := &sync.WaitGroup{}
	sessionName := "dill"
	if cfg.ServiceName != "" {
		sessionName = cfg.ServiceName
	}
	err := login.InitLoginApi(cfg.SessionMaxAge, cfg.RefreshTokenMaxAge, sessionName, refreshTokenCache)
	if err != nil {
		log.Fatal("login api initiation failed: %s", err.Error())
	}
	// 统一注册所有API
	registerAllApis()
	r := newRouter(wg, cfg.ServerMode, sessionName, store)
	return &Server{
		cfg: cfg,
		e:   r,
		wg:  wg,
	}
}

func (s *Server) Run() {
	s.server = &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", s.cfg.Port),
		Handler: s.e,
	}
	go s.server.ListenAndServe()
}

func (s *Server) Stop() error {
	err := s.server.Shutdown(context.Background())
	// whether the err is nil, we should wait for wait group
	s.wg.Wait()
	return err
}

func newRouter(wg *sync.WaitGroup, serverMode, serviceName string, store sessions.Store) *gin.Engine {
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

	// 添加CORS中间件
	r.Use(corsMiddleware())

	//注册session 中间件
	r.Use(sessions.Sessions(serviceName+"_session", store))
	// register apis here
	r.POST("/", middlewares.PreJobMiddleware(), middlewares.AuthMiddleware(serverMode), handler.Handle)
	return r
}

// corsMiddleware 添加CORS支持
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 允许的来源
		allowedOrigins := []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"http://localhost:8080",
			"http://127.0.0.1:8080",
			"http://beast-royale-fe.prj-console.svc.a4.u4/",
			"https://beast-royale-fe.prj-console.svc.a4.u4/",
		}

		// 检查来源是否被允许
		isAllowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				isAllowed = true
				break
			}
		}

		// 开发模式下允许所有来源
		if !isAllowed {
			// 在开发环境下允许所有localhost和127.0.0.1来源
			if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
				isAllowed = true
			}
		}

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func ginWaitGroup(wg *sync.WaitGroup) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Set(SERVER_WAIT_GROUP_LABEL, wg)
	}
}

func ginLogger() gin.HandlerFunc {
	w := log.GlobalLogger().Writer()
	return gin.LoggerWithWriter(w)
}

// registerAllApis 统一注册所有API
func registerAllApis() {
	// 注册登录相关API
	login.RegisterLoginApis()
	// 注册用户相关API
	user.RegisterUserApis()
	// 注册匹配相关API
	match.RegisterMatchApis()
	// 注册对战相关API
	battle.RegisterBattleApis()
	// 注册系统相关API
	system.RegisterSystemApis()
}
