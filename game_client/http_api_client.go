package gameclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/rpc/proto"
	"github.com/CryptoElementals/common/server/api"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/google/uuid"
	"golang.org/x/net/publicsuffix"
)

// HttpApiClient handles HTTP API requests to the game server
type HttpApiClient struct {
	client        *http.Client
	baseURL       string
	sessionCookie string
	ctx           context.Context
}

// GetBaseURL returns the base URL
func (c *HttpApiClient) GetBaseURL() string {
	return c.baseURL
}

// NewHttpApiClient creates a new HTTP API client
func NewHttpApiClient(ctx context.Context, baseURL string) (*HttpApiClient, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	return &HttpApiClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
		baseURL: baseURL,
		ctx:     ctx,
	}, nil
}

// GetSessionCookie returns the current session cookie
func (c *HttpApiClient) GetSessionCookie() string {
	return c.sessionCookie
}

// SetSessionCookie sets the session cookie
func (c *HttpApiClient) SetSessionCookie(cookie string) {
	c.sessionCookie = cookie
}

// makeRequest makes an HTTP POST request to the API server
func (c *HttpApiClient) makeRequest(action string, req interface{}, resp interface{}, requireAuth bool) error {
	// Convert request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(c.ctx, "POST", c.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if requireAuth && c.sessionCookie != "" {
		httpReq.Header.Set("Cookie", c.sessionCookie)
	}

	httpResp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d: %s", httpResp.StatusCode, string(body))
	}

	// Extract session cookie from response if present
	if cookies := httpResp.Cookies(); len(cookies) > 0 {
		var cookieStrings []string
		for _, cookie := range cookies {
			cookieStrings = append(cookieStrings, cookie.Name+"="+cookie.Value)
		}
		c.sessionCookie = strings.Join(cookieStrings, "; ")
	}

	// Parse response
	if err := json.Unmarshal(body, resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

// GetLoginCode gets the login code (nonce) from the server
func (c *HttpApiClient) GetLoginCode(address string) (int, string, error) {
	req := &api.GetLoginCodeRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.GET_LOGIN_CODE_LABEL,
			RequestUUID: uuid.NewString(),
		},
		Address: address,
	}

	var resp api.GetLoginCodeResponse
	if err := c.makeRequest(api.GET_LOGIN_CODE_LABEL, req, &resp, false); err != nil {
		return 0, "", err
	}

	if resp.RetCode != 0 {
		return 0, "", fmt.Errorf("get login code failed: %s", resp.Message)
	}

	return resp.Nonce, resp.LoginCode, nil
}

// Login logs in with signature and returns refresh token
func (c *HttpApiClient) Login(signature string, address string, nonce int) (string, string, error) {
	req := &api.LoginDillRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.LOGIN_DILL_LABEL,
			RequestUUID: uuid.NewString(),
		},
		Signature: signature,
		Address:   address,
		Nonce:     nonce,
	}

	var resp api.LoginDillResponse
	if err := c.makeRequest(api.LOGIN_DILL_LABEL, req, &resp, false); err != nil {
		return "", "", err
	}

	if resp.RetCode != 0 {
		return "", "", fmt.Errorf("login failed: %s", resp.Message)
	}

	// Return the session cookie and refresh token
	return c.sessionCookie, resp.RefreshToken, nil
}

// IsUserLoggedIn checks if user is logged in and returns player ID
func (c *HttpApiClient) IsUserLoggedIn(refreshToken string) (string, bool, error) {
	req := &api.IsUserLoggedInRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.IS_USER_LOGGED_IN_LABEL,
			RequestUUID: uuid.NewString(),
		},
		RefreshToken: refreshToken,
	}

	var resp api.IsUserLoggedInResponse
	if err := c.makeRequest(api.IS_USER_LOGGED_IN_LABEL, req, &resp, false); err != nil {
		return "", false, err
	}

	if resp.RetCode != 0 {
		return "", false, fmt.Errorf("is user logged in failed: %s", resp.Message)
	}

	if !resp.UserLoggedIn {
		return "", false, nil
	}

	return resp.PlayerID, true, nil
}

// JoinQueue joins the match queue via HTTP API
func (c *HttpApiClient) JoinQueue(mode, tempAddress, playerID string) error {
	req := &api.JoinQueueRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.JOIN_QUEUE_LABEL,
			RequestUUID: uuid.NewString(),
		},
		Mode:        mode,
		TempAddress: tempAddress,
		PlayerID:    playerID,
	}

	var resp api.JoinQueueResponse
	if err := c.makeRequest(api.JOIN_QUEUE_LABEL, req, &resp, true); err != nil {
		return err
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("join queue failed: %s", resp.Message)
	}

	return nil
}

// ExitQueue exits the match queue via HTTP API
func (c *HttpApiClient) ExitQueue(tempAddress, playerID string) error {
	req := &api.ExitQueueRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.EXIT_QUEUE_LABEL,
			RequestUUID: uuid.NewString(),
		},
		Mode:        "PvP", // ExitQueue requires Mode field
		TempAddress: tempAddress,
		PlayerID:    playerID,
	}

	var resp api.ExitQueueResponse
	if err := c.makeRequest(api.EXIT_QUEUE_LABEL, req, &resp, true); err != nil {
		return err
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("exit queue failed: %s", resp.Message)
	}

	return nil
}

// ConfirmBattle confirms battle via HTTP API
func (c *HttpApiClient) ConfirmBattle(gameID uint, roundNumber, turnNumber uint32, tempAddress, playerID string) error {
	req := &api.ConfirmBattleRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.CONFIRM_BATTLE_LABEL,
			RequestUUID: uuid.NewString(),
		},
		GameID:      uint32(gameID),
		RoundNumber: uint(roundNumber),
		TurnNumber:  uint(turnNumber),
		TempAddress: tempAddress,
		PlayerID:    playerID,
	}

	var resp api.ConfirmBattleResponse
	if err := c.makeRequest(api.CONFIRM_BATTLE_LABEL, req, &resp, true); err != nil {
		return err
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("confirm battle failed: %s", resp.Message)
	}

	return nil
}

// ConfirmMatch confirms a pending game_match. HTTP API support is not wired; use gRPC LobbyService.ConfirmMatch.
func (c *HttpApiClient) ConfirmMatch(matchID int64, tempAddress, playerID string) error {
	return fmt.Errorf("ConfirmMatch is not available on HttpApiClient (match_id=%d); use gRPC lobby or add server API action", matchID)
}

// HasCollectedDailyReward checks if the daily reward has been collected
func (c *HttpApiClient) HasCollectedDailyReward(playerID string) (bool, error) {
	req := &api.HasCollectedDailyRewardRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.HAS_COLLECTED_DAILY_REWARD_LABEL,
			RequestUUID: uuid.NewString(),
		},
		PlayerID: playerID,
	}

	var resp api.HasCollectedDailyRewardResponse
	if err := c.makeRequest(api.HAS_COLLECTED_DAILY_REWARD_LABEL, req, &resp, true); err != nil {
		return false, err
	}

	if resp.RetCode != 0 {
		return false, fmt.Errorf("check daily reward collection failed: %s", resp.Message)
	}

	return resp.Collected, nil
}

// CollectDailyReward collects the daily reward
func (c *HttpApiClient) CollectDailyReward(playerID string) error {
	req := &api.CollectDailyRewardRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.COLLECT_DAILY_REWARD_LABEL,
			RequestUUID: uuid.NewString(),
		},
		PlayerID: playerID,
	}

	var resp api.CollectDailyRewardResponse
	if err := c.makeRequest(api.COLLECT_DAILY_REWARD_LABEL, req, &resp, true); err != nil {
		return err
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("collect daily reward failed: %s", resp.Message)
	}

	return nil
}

// GetUserProfile gets user profile information
func (c *HttpApiClient) GetUserProfile(playerID string) (*api.UserInfo, error) {
	req := &api.GetUserProfileRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.GET_USER_PROFILE_LABEL,
			RequestUUID: uuid.NewString(),
		},
		PlayerID: playerID,
	}

	var resp api.GetUserProfileResponse
	if err := c.makeRequest(api.GET_USER_PROFILE_LABEL, req, &resp, false); err != nil {
		return nil, err
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("get user profile failed: %s", resp.Message)
	}

	return &resp.UserInfo, nil
}

// SetUserProfile sets user profile information
func (c *HttpApiClient) SetUserProfile(playerID, name, avatar string) error {
	req := &api.SetUserProfileRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.SET_USER_PROFILE_LABEL,
			RequestUUID: uuid.NewString(),
		},
		PlayerID: playerID,
		Name:     name,
		Avatar:   avatar,
	}

	var resp api.SetUserProfileResponse
	if err := c.makeRequest(api.SET_USER_PROFILE_LABEL, req, &resp, true); err != nil {
		return err
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("set user profile failed: %s", resp.Message)
	}

	return nil
}

// SubmitPlayerCommitment submits a player commitment via HTTP API
func (c *HttpApiClient) SubmitPlayerCommitment(gameID uint, roundNumber, turnNumber uint32, commitment []byte, signature []byte, tempAddress, playerID string) error {
	req := &api.SubmitPlayerCommitmentRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.SUBMIT_PLAYER_COMMITMENT_LABEL,
			RequestUUID: uuid.NewString(),
		},
		GameID:      uint32(gameID),
		RoundNumber: roundNumber,
		TurnNumber:  turnNumber,
		Commitment:  hexutil.Encode(commitment),
		Signature:   hexutil.Encode(signature),
		TempAddress: tempAddress,
		PlayerID:    playerID,
	}

	var resp api.SubmitPlayerCommitmentResponse
	if err := c.makeRequest(api.SUBMIT_PLAYER_COMMITMENT_LABEL, req, &resp, true); err != nil {
		return err
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("submit player commitment failed: %s", resp.Message)
	}

	return nil
}

// SubmitPlayerCard submits a player card via HTTP API
func (c *HttpApiClient) SubmitPlayerCard(gameID uint, roundNumber, turnNumber uint32, card uint32, salt string, signature []byte, tempAddress, playerID string) error {
	req := &api.SubmitPlayerCardRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.SUBMIT_PLAYER_CARD_LABEL,
			RequestUUID: uuid.NewString(),
		},
		GameID:      uint32(gameID),
		RoundNumber: roundNumber,
		TurnNumber:  turnNumber,
		Card:        card,
		Salt:        salt, // Salt can be hex string or regular string
		Signature:   hexutil.Encode(signature),
		TempAddress: tempAddress,
		PlayerID:    playerID,
	}

	var resp api.SubmitPlayerCardResponse
	if err := c.makeRequest(api.SUBMIT_PLAYER_CARD_LABEL, req, &resp, true); err != nil {
		return err
	}

	if resp.RetCode != 0 {
		return fmt.Errorf("submit player card failed: %s", resp.Message)
	}

	return nil
}

// SubscribeGameInfo subscribes to game events via SSE
// Returns event channel, error channel, and cancel function
func (c *HttpApiClient) SubscribeGameInfo(tempAddress, playerID string, duration int) (<-chan *proto.Event, <-chan error, context.CancelFunc, error) {
	if duration <= 0 {
		duration = 86400 // Default to 1 day
	}

	req := &api.SubscribeGameInfoRequest{
		BaseRequest: api.BaseRequest{
			Action:      api.SUBSCRIBE_GAME_INFO_LABEL,
			RequestUUID: uuid.NewString(),
		},
		TempAddress: tempAddress,
		PlayerID:    playerID,
		Duration:    duration,
	}

	eventChan := make(chan *proto.Event, 10)
	errChan := make(chan error, 10)
	ctx, cancel := context.WithCancel(c.ctx)

	// Start SSE connection in goroutine
	go c.handleSSE(ctx, req, eventChan, errChan)

	return eventChan, errChan, cancel, nil
}

// handleSSE handles Server-Sent Events stream
func (c *HttpApiClient) handleSSE(ctx context.Context, req *api.SubscribeGameInfoRequest, eventChan chan<- *proto.Event, errChan chan<- error) {
	defer close(eventChan)
	defer close(errChan)

	// Marshal request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		errChan <- fmt.Errorf("failed to marshal SSE request: %w", err)
		return
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(jsonData))
	if err != nil {
		errChan <- fmt.Errorf("failed to create SSE request: %w", err)
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.sessionCookie != "" {
		httpReq.Header.Set("Cookie", c.sessionCookie)
	}

	// Use a client with no timeout for SSE
	sseClient := &http.Client{Timeout: 0}
	resp, err := sseClient.Do(httpReq)
	if err != nil {
		errChan <- fmt.Errorf("failed to connect to SSE stream: %w", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errChan <- fmt.Errorf("SSE connection failed with status: %d", resp.StatusCode)
		return
	}

	// Read SSE stream line by line
	scanner := &sseScanner{reader: resp.Body}
	var currentEvent strings.Builder

	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := scanner.readLine()
			if err != nil {
				if err != io.EOF {
					errChan <- err
				}
				return
			}

			if strings.HasPrefix(line, "data: ") {
				eventData := strings.TrimPrefix(line, "data: ")
				currentEvent.WriteString(eventData)
			} else if line == "" && currentEvent.Len() > 0 {
				// Empty line indicates end of event
				protoEvent := c.parseSSEEvent(currentEvent.String())
				if protoEvent != nil {
					select {
					case eventChan <- protoEvent:
					default:
						log.Warnw("event channel full, dropping event", "type", protoEvent.Type)
					}
				}
				currentEvent.Reset()
			}
		}
	}
}

// sseScanner is a simple line scanner for SSE
type sseScanner struct {
	reader io.Reader
	buffer []byte
}

func (s *sseScanner) readLine() (string, error) {
	if s.buffer == nil {
		s.buffer = make([]byte, 1)
	}

	var line strings.Builder
	for {
		n, err := s.reader.Read(s.buffer)
		if err != nil {
			if line.Len() > 0 {
				return line.String(), nil
			}
			return "", err
		}
		if n == 0 {
			break
		}
		char := s.buffer[0]
		if char == '\n' {
			return line.String(), nil
		}
		if char != '\r' {
			line.WriteByte(char)
		}
	}
	return line.String(), nil
}

// parseSSEEvent parses SSE event and converts it to proto.Event
func (c *HttpApiClient) parseSSEEvent(eventData string) *proto.Event {
	var sseEvent struct {
		Type      string                 `json:"type"`
		Data      map[string]interface{} `json:"data"`
		Timestamp time.Time              `json:"timestamp"`
	}

	if err := json.Unmarshal([]byte(eventData), &sseEvent); err != nil {
		log.Warnw("failed to parse SSE event", "error", err, "data", eventData)
		return nil
	}

	// Convert SSE event to proto.Event
	eventType, ok := sseEvent.Data["EventType"].(string)
	if !ok {
		return nil
	}

	protoEvent := &proto.Event{}
	switch eventType {
	case "matched":
		protoEvent.Type = proto.EventType_TYPE_MATCHED
		if msg, ok := sseEvent.Data["Message"].(map[string]interface{}); ok {
			gameMatched := &proto.GameMatched{}
			if mid, ok := msg["MatchId"].(float64); ok {
				gameMatched.MatchId = int64(mid)
			} else if mid, ok := msg["matchId"].(float64); ok {
				gameMatched.MatchId = int64(mid)
			}
			if players, ok := msg["Players"].([]interface{}); ok {
				for _, p := range players {
					if playerMap, ok := p.(map[string]interface{}); ok {
						player := &proto.PlayerAddress{}
						if id, ok := playerMap["Id"].(float64); ok {
							player.Id = int64(id)
						}
						if tempAddr, ok := playerMap["TemporaryAddress"].(string); ok {
							player.TemporaryAddress = tempAddr
						}
						gameMatched.Players = append(gameMatched.Players, player)
					}
				}
			}
			protoEvent.Event = &proto.Event_GameMatched{GameMatched: gameMatched}
		}
	case "gameContinuable":
		protoEvent.Type = proto.EventType_TYPE_GAME_CONTINUABLE
		if msg, ok := sseEvent.Data["Message"].(map[string]interface{}); ok {
			gc := &proto.GameContinuable{}
			if mid, ok := msg["MatchId"].(float64); ok {
				gc.MatchId = int64(mid)
			} else if mid, ok := msg["matchId"].(float64); ok {
				gc.MatchId = int64(mid)
			}
			if lid, ok := msg["LastGameId"].(float64); ok {
				gc.LastGameId = uint32(lid)
			} else if lid, ok := msg["lastGameId"].(float64); ok {
				gc.LastGameId = uint32(lid)
			}
			if players, ok := msg["Players"].([]interface{}); ok {
				for _, p := range players {
					if playerMap, ok := p.(map[string]interface{}); ok {
						player := &proto.PlayerAddress{}
						if id, ok := playerMap["Id"].(float64); ok {
							player.Id = int64(id)
						} else if id, ok := playerMap["id"].(float64); ok {
							player.Id = int64(id)
						}
						if tempAddr, ok := playerMap["TemporaryAddress"].(string); ok {
							player.TemporaryAddress = tempAddr
						} else if tempAddr, ok := playerMap["temporary_address"].(string); ok {
							player.TemporaryAddress = tempAddr
						}
						gc.Players = append(gc.Players, player)
					}
				}
			} else if players, ok := msg["players"].([]interface{}); ok {
				for _, p := range players {
					if playerMap, ok := p.(map[string]interface{}); ok {
						player := &proto.PlayerAddress{}
						if id, ok := playerMap["id"].(float64); ok {
							player.Id = int64(id)
						}
						if tempAddr, ok := playerMap["temporary_address"].(string); ok {
							player.TemporaryAddress = tempAddr
						}
						gc.Players = append(gc.Players, player)
					}
				}
			}
			protoEvent.Event = &proto.Event_GameContinuable{GameContinuable: gc}
		}
	case "partConfirmed":
		protoEvent.Type = proto.EventType_TYPE_PART_CONFIRMED
		protoEvent.Event = &proto.Event_X{}
	case "gameCreated":
		protoEvent.Type = proto.EventType_TYPE_GAME_CREATED
		if msg, ok := sseEvent.Data["Message"].(map[string]interface{}); ok {
			gameReady := &proto.GameReady{}
			if gameId, ok := msg["GameId"].(float64); ok {
				gameReady.GameId = uint32(gameId)
			}
			protoEvent.Event = &proto.Event_GameReady{GameReady: gameReady}
		}
	case "roundReady":
		protoEvent.Type = proto.EventType_TYPE_ROUND_READY
		if msg, ok := sseEvent.Data["Message"].(map[string]interface{}); ok {
			roundReady := &proto.RoundReady{}
			if gameId, ok := msg["GameId"].(float64); ok {
				roundReady.GameId = uint32(gameId)
			}
			if roundNum, ok := msg["RoundNum"].(float64); ok {
				roundReady.RoundNum = uint32(roundNum)
			}
			protoEvent.Event = &proto.Event_RoundReady{RoundReady: roundReady}
		}
	case "turnReady":
		protoEvent.Type = proto.EventType_TYPE_TURN_READY
		if msg, ok := sseEvent.Data["Message"].(map[string]interface{}); ok {
			turnReady := &proto.TurnReady{}
			if gameId, ok := msg["GameId"].(float64); ok {
				turnReady.GameId = uint32(gameId)
			}
			if roundNum, ok := msg["RoundNum"].(float64); ok {
				turnReady.RoundNum = uint32(roundNum)
			}
			if turnNum, ok := msg["TurnNum"].(float64); ok {
				turnReady.TurnNum = uint32(turnNum)
			}
			protoEvent.Event = &proto.Event_TurnReady{TurnReady: turnReady}
		}
	case "commitmentsOnChain":
		protoEvent.Type = proto.EventType_TYPE_COMMITMENTS_ON_CHAIN
		if msg, ok := sseEvent.Data["Message"].(map[string]interface{}); ok {
			commitmentsOnChain := &proto.CommitmentsOnChain{}
			if gameId, ok := msg["GameId"].(float64); ok {
				commitmentsOnChain.GameId = uint32(gameId)
			}
			if roundNum, ok := msg["RoundNum"].(float64); ok {
				commitmentsOnChain.RoundNum = uint32(roundNum)
			}
			if turnNum, ok := msg["TurnNum"].(float64); ok {
				commitmentsOnChain.TurnNum = uint32(turnNum)
			}
			protoEvent.Event = &proto.Event_CommitmentsOnChain{CommitmentsOnChain: commitmentsOnChain}
		}
	case "turnComplete":
		protoEvent.Type = proto.EventType_TYPE_TURN_COMPLETE
		if msg, ok := sseEvent.Data["Message"].(map[string]interface{}); ok {
			turnCompleted := &proto.TurnCompleted{}
			if gameId, ok := msg["GameId"].(float64); ok {
				turnCompleted.GameId = uint32(gameId)
			}
			if roundNum, ok := msg["RoundNum"].(float64); ok {
				turnCompleted.RoundNum = uint32(roundNum)
			}
			if turnNum, ok := msg["TurnNum"].(float64); ok {
				turnCompleted.TurnNum = uint32(turnNum)
			}
			if isRoundComplete, ok := msg["IsRoundComplete"].(bool); ok {
				turnCompleted.IsRoundComplete = isRoundComplete
			}
			if isGameComplete, ok := msg["IsGameComplete"].(bool); ok {
				turnCompleted.IsGameComplete = isGameComplete
			}
			protoEvent.Event = &proto.Event_TurnCompleted{TurnCompleted: turnCompleted}
		}
	default:
		return nil
	}

	return protoEvent
}
