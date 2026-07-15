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
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/config"
	"github.com/CryptoElementals/common/db"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/handler"
	"github.com/CryptoElementals/common/server/invite"
	"github.com/CryptoElementals/common/server/middlewares"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/memstore"
	"github.com/gin-gonic/gin"
)

type Server struct {
	e      *gin.Engine
	server *http.Server
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
	sessionName := "dill"
	if cfg.ServiceName != "" {
		sessionName = cfg.ServiceName
	}
	err := api.InitLoginApi(cfg.SessionMaxAge, cfg.RefreshTokenMaxAge, sessionName, refreshTokenCache)
	if err != nil {
		log.Fatal("login api initiation failed: %s", err.Error())
	}
	r := newRouter(cfg, sessionName, store)
	return &Server{
		cfg: cfg,
		e:   r,
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
	return s.server.Shutdown(context.Background())
}

func newRouter(cfg *config.ServerConfig, serviceName string, store sessions.Store) *gin.Engine {
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
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// 添加CORS中间件
	r.Use(corsMiddleware())

	//注册session 中间件
	r.Use(sessions.Sessions(serviceName+"_session", store))
	// register apis here
	r.POST("/", middlewares.PreJobMiddleware(), middlewares.AuthMiddleware(cfg.ServerMode), handler.Handle)

	// Google OAuth endpoints
	r.GET("/auth/google/login", googleLoginLogMiddleware(), googleLoginHandler(cfg))
	r.GET("/auth/google/callback", googleCallbackLogMiddleware(), googleCallbackHandler(cfg))
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
			"https://a.elementra.xyz",
			"http://a.elementra.xyz",
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

func ginLogger() gin.HandlerFunc {
	w := log.GlobalLogger().Writer()
	return gin.LoggerWithWriter(w)
}

// Helpers for Google OAuth

func googleLoginLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Infow("google oauth login request",
			"client", c.ClientIP(),
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"invite_code", c.Query("invite_code"),
			"env", c.Query("env"),
			"force", c.Query("force"),
		)
		c.Next()
	}
}

func googleCallbackLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		sessionInviteCode := ""
		if v := session.Get("oauth_invite_code"); v != nil {
			if s, ok := v.(string); ok {
				sessionInviteCode = s
			}
		}
		sessionEnv := ""
		if v := session.Get("oauth_env"); v != nil {
			if s, ok := v.(string); ok {
				sessionEnv = s
			}
		}
		log.Infow("google oauth callback request",
			"client", c.ClientIP(),
			"path", c.Request.URL.Path,
			"query", c.Request.URL.RawQuery,
			"state", c.Query("state"),
			"has_code", c.Query("code") != "",
			"session_invite_code", sessionInviteCode,
			"session_env", sessionEnv,
		)
		c.Next()
	}
}

func logGoogleOAuthJSONResponse(handler string, status int, body gin.H) {
	payload, _ := json.Marshal(body)
	log.Infow("google oauth response",
		"handler", handler,
		"status", status,
		"body", string(payload),
	)
}

func logGoogleOAuthRedirectResponse(handler string, status int, location string) {
	log.Infow("google oauth response",
		"handler", handler,
		"status", status,
		"redirect", location,
	)
}

func googleLoginHandler(cfg *config.ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
			body := gin.H{"error": "google oauth not configured"}
			logGoogleOAuthJSONResponse("googleLoginHandler", http.StatusBadRequest, body)
			c.JSON(http.StatusBadRequest, body)
			return
		}
		// generate state and store in session
		state, _ := randomString(24)
		// optional environment hint, used in callback to decide frontend redirect target
		env := c.Query("env")

		session := sessions.Default(c)
		session.Set("oauth_state", state)
		if env != "" {
			session.Set("oauth_env", env)
		}
		if inviteCode := strings.TrimSpace(c.Query("invite_code")); inviteCode != "" {
			session.Set("oauth_invite_code", inviteCode)
		}
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
		logGoogleOAuthRedirectResponse("googleLoginHandler", http.StatusFound, authURL)
		c.Redirect(http.StatusFound, authURL)
	}
}

func googleCallbackHandler(cfg *config.ServerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		queryState := c.Query("state")
		code := c.Query("code")
		if queryState == "" || code == "" {
			body := gin.H{"error": "missing state or code"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusBadRequest, body)
			c.JSON(http.StatusBadRequest, body)
			return
		}
		session := sessions.Default(c)
		state := session.Get("oauth_state")
		if state == nil || state.(string) != queryState {
			body := gin.H{"error": "invalid state"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusBadRequest, body)
			c.JSON(http.StatusBadRequest, body)
			return
		}
		// clear state and read env hint
		session.Delete("oauth_state")
		envVal := session.Get("oauth_env")
		if envVal != nil {
			session.Delete("oauth_env")
		}
		_ = session.Save()

		oauthEnvHint := ""
		if envVal != nil {
			if s, ok := envVal.(string); ok {
				oauthEnvHint = s
			}
		}

		// exchange code for token
		form := url.Values{}
		form.Set("code", code)
		form.Set("client_id", cfg.GoogleClientID)
		form.Set("client_secret", cfg.GoogleClientSecret)
		form.Set("redirect_uri", cfg.GoogleRedirectURL)
		form.Set("grant_type", "authorization_code")
		tokenResp, err := http.PostForm("https://oauth2.googleapis.com/token", form)
		if err != nil {
			body := gin.H{"error": "exchange failed"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusBadGateway, body)
			c.JSON(http.StatusBadGateway, body)
			return
		}
		defer tokenResp.Body.Close()
		if tokenResp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(tokenResp.Body)
			body := gin.H{"error": "exchange failed", "detail": string(bodyBytes)}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusBadGateway, body)
			c.JSON(http.StatusBadGateway, body)
			return
		}
		var tokenPayload struct {
			AccessToken string `json:"access_token"`
			IdToken     string `json:"id_token"`
			TokenType   string `json:"token_type"`
			ExpiresIn   int    `json:"expires_in"`
		}
		if err := json.NewDecoder(tokenResp.Body).Decode(&tokenPayload); err != nil {
			body := gin.H{"error": "invalid token response"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusBadGateway, body)
			c.JSON(http.StatusBadGateway, body)
			return
		}
		req, _ := http.NewRequest("GET", "https://openidconnect.googleapis.com/v1/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+tokenPayload.AccessToken)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			body := gin.H{"error": "userinfo fetch failed"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusBadGateway, body)
			c.JSON(http.StatusBadGateway, body)
			return
		}
		defer resp.Body.Close()
		var payload struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			body := gin.H{"error": "invalid userinfo"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusBadGateway, body)
			c.JSON(http.StatusBadGateway, body)
			return
		}
		log.Infof("googleCallbackHandler: payload: %+v", payload)
		log.Infof("googleCallbackHandler: email: %s", payload.Email)
		log.Infof("googleCallbackHandler: name: %s", payload.Name)
		log.Infof("googleCallbackHandler: access_token: %s", tokenPayload.AccessToken)
		log.Infof("googleCallbackHandler: id_token: %s", tokenPayload.IdToken)
		log.Infof("googleCallbackHandler: token_type: %s", tokenPayload.TokenType)
		log.Infof("googleCallbackHandler: expires_in: %d", tokenPayload.ExpiresIn)
		profile, isNewUser, err := db.GetOrCreateUserProfileByEmail(payload.Email, payload.Name)
		if err != nil {
			body := gin.H{"error": "create user profile failed"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusInternalServerError, body)
			c.JSON(http.StatusInternalServerError, body)
			return
		}
		serverType := db.EffectiveServerType(profile)
		inviteCode := ""
		if v := session.Get("oauth_invite_code"); v != nil {
			if s, ok := v.(string); ok {
				inviteCode = strings.TrimSpace(s)
			}
			session.Delete("oauth_invite_code")
			_ = session.Save()
		}
		if inviteErr := invite.EnsureInviterCodeOnLogin(profile.PlayerID, serverType); inviteErr != nil {
			body := gin.H{"error": "ensure inviter code failed"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusInternalServerError, body)
			c.JSON(http.StatusInternalServerError, body)
			return
		}
		var warnCode int
		var warnMessage string
		if inviteCode != "" {
			var inviteErr error
			warnCode, warnMessage, inviteErr = invite.TryApplyOnLogin(isNewUser, profile.PlayerID, inviteCode)
			if inviteErr != nil {
				body := gin.H{"error": "apply invite failed"}
				logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusInternalServerError, body)
				c.JSON(http.StatusInternalServerError, body)
				return
			}
		}
		playerIDStr := fmt.Sprintf("%d", profile.PlayerID)
		token, err := api.SaveRefreshTokenForUserId(playerIDStr)
		if err != nil {
			body := gin.H{"error": "issue refresh token failed"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusInternalServerError, body)
			c.JSON(http.StatusInternalServerError, body)
			return
		}
		tempCode, err := api.SaveTempCodeForRefreshToken(token, 300)
		if err != nil {
			body := gin.H{"error": "issue temp code failed"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusInternalServerError, body)
			c.JSON(http.StatusInternalServerError, body)
			return
		}

		// decide frontend redirect target based on env hint
		env := oauthEnvHint

		var frontendURL string
		switch env {
		case "local":
			// local frontend for developer debugging
			frontendURL = "http://localhost:5173/"
		default:
			frontendURL = cfg.GoogleFrontendURL
		}

		if frontendURL == "" {
			body := gin.H{"error": "frontend redirect url not configured"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusInternalServerError, body)
			c.JSON(http.StatusInternalServerError, body)
			return
		}
		u, err := url.Parse(frontendURL)
		if err != nil {
			body := gin.H{"error": "invalid frontend redirect url"}
			logGoogleOAuthJSONResponse("googleCallbackHandler", http.StatusInternalServerError, body)
			c.JSON(http.StatusInternalServerError, body)
			return
		}
		q := u.Query()
		q.Set("code", tempCode)
		if warnCode > 0 {
			q.Set("warn_code", strconv.Itoa(warnCode))
			q.Set("warn_message", warnMessage)
		}
		u.RawQuery = q.Encode()
		redirectURL := u.String()
		logGoogleOAuthRedirectResponse("googleCallbackHandler", http.StatusFound, redirectURL)
		c.Redirect(http.StatusFound, redirectURL)
	}
}

func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
