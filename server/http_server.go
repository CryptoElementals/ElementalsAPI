package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
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
	err := api.InitLoginApi(cfg.SessionMaxAge, cfg.RefreshTokenMaxAge, sessionName, refreshTokenCache)
	if err != nil {
		log.Fatal("login api initiation failed: %s", err.Error())
	}
	r := newRouter(wg, cfg, sessionName, store)
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

func newRouter(wg *sync.WaitGroup, cfg *config.ServerConfig, serviceName string, store sessions.Store) *gin.Engine {
	mode := strings.ToLower(cfg.ServerMode)
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
	r.POST("/", middlewares.PreJobMiddleware(), middlewares.AuthMiddleware(cfg.ServerMode), handler.Handle)

	// Google OAuth endpoints
	r.GET("/auth/login", googleLoginHandler(cfg))
	r.GET("/auth/callback", googleCallbackHandler(cfg))
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
			"https://d.elementra.xyz",
			"http://d.elementra.xyz",
			"https://elementra.xyz",
			"http://elementra.xyz",
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

// Helpers for Google OAuth
func googleLoginHandler(cfg *config.ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "google oauth not configured"})
			return
		}
		// generate state and store in session
		state, _ := randomString(24)
		session := sessions.Default(c)
		session.Set("oauth_state", state)
		_ = session.Save()

		q := url.Values{}
		q.Set("client_id", cfg.GoogleClientID)
		q.Set("redirect_uri", cfg.GoogleRedirectURL)
		q.Set("response_type", "code")
		q.Set("scope", "openid email profile")
		q.Set("state", state)

		if c.Query("force") == "true" {
			q.Set("prompt", "consent")
		}

		authURL := "https://accounts.google.com/o/oauth2/auth?" + q.Encode()
		c.Redirect(http.StatusFound, authURL)
	}
}

func googleCallbackHandler(cfg *config.ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryState := c.Query("state")
		code := c.Query("code")
		if queryState == "" || code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing state or code"})
			return
		}
		session := sessions.Default(c)
		state := session.Get("oauth_state")
		if state == nil || state.(string) != queryState {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
			return
		}
		// clear state
		session.Delete("oauth_state")
		_ = session.Save()

		// exchange code for token
		form := url.Values{}
		form.Set("code", code)
		form.Set("client_id", cfg.GoogleClientID)
		form.Set("client_secret", cfg.GoogleClientSecret)
		form.Set("redirect_uri", cfg.GoogleRedirectURL)
		form.Set("grant_type", "authorization_code")
		tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", form)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "exchange failed"})
			return
		}
		defer tokenResp.Body.Close()
		if tokenResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(tokenResp.Body)
			c.JSON(http.StatusBadGateway, gin.H{"error": "exchange failed", "detail": string(body)})
			return
		}
		var tokenPayload struct {
			AccessToken string `json:"access_token"`
			IdToken     string `json:"id_token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   int    `json:"expires_in"`
		}
		if err := json.NewDecoder(tokenResp.Body).Decode(&tokenPayload); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid token response"})
			return
		}
		req, _ := http.NewRequest("GET", "https://openidconnect.googleapis.com/v1/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+tokenPayload.AccessToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			c.JSON(http.StatusBadGateway, gin.H{"error": "userinfo fetch failed"})
			return
		}
		defer resp.Body.Close()
		var payload struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "invalid userinfo"})
			return
		}
		log.Infof("googleCallbackHandler: payload: %+v", payload)
		log.Infof("googleCallbackHandler: email: %s", payload.Email)
		log.Infof("googleCallbackHandler: name: %s", payload.Name)
		log.Infof("googleCallbackHandler: access_token: %s", tokenPayload.AccessToken)
		log.Infof("googleCallbackHandler: id_token: %s", tokenPayload.IdToken)
		log.Infof("googleCallbackHandler: token_type: %s", tokenPayload.TokenType)
		log.Infof("googleCallbackHandler: expires_in: %d", tokenPayload.ExpiresIn)
		// 获取或创建用户档案（邮箱已存在则直接视为登录）
		userProfile, err := db.GetOrCreateUserProfileByEmail(payload.Email, payload.Name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create user profile failed"})
			return
		}
		playerIDStr := fmt.Sprintf("%d", userProfile.PlayerID)
		session.Set(api.SESSION_USER_KEY, playerIDStr)
		_ = session.Save()
		// issue refresh token
		token, err := api.SaveRefreshTokenForUserId(playerIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "issue refresh token failed"})
			return
		}
		// 生成短期code（5分钟），映射到refresh_token；重定向时仅携带code
		tempCode, err := api.SaveTempCodeForRefreshToken(token, 300)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "issue temp code failed"})
			return
		}
		// 重定向，并在 URL 参数中附带 code
		if cfg.GoogleFrontendURL == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "frontend redirect url not configured"})
			return
		}
		u, err := url.Parse(cfg.GoogleFrontendURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid frontend redirect url"})
			return
		}
		q := u.Query()
		q.Set("code", tempCode)
		u.RawQuery = q.Encode()
		c.Redirect(http.StatusFound, u.String())
		return
	}
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
