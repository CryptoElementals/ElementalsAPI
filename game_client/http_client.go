package gameclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"reflect"
	"strings"

	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/room_server/worker/types"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/events"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type HttpClient struct {
	ctx           context.Context
	endpoint      string
	client        *http.Client
	accountWallet *wallet.Wallet
}

func NewHttpClient(ctx context.Context, endpoint string, accountWallet *wallet.Wallet) *HttpClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{
		Jar: jar,
	}
	return &HttpClient{
		ctx:           ctx,
		endpoint:      endpoint,
		client:        client,
		accountWallet: accountWallet,
	}
}

func (c *HttpClient) Start() error {
	_, err := c.doLogin(c.accountWallet)
	return err
}

func (c *HttpClient) doPost(req, resp any) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}
	r := bytes.NewBuffer(reqBody)
	res, err := c.client.Post(c.endpoint, "application/json", r)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return err
	}
	reflectType := reflect.TypeOf(resp)
	_, found := reflectType.FieldByName("BaseResponse")
	if !found {
		return nil
	}
	reflectVal := reflect.ValueOf(resp)
	baseRespReflect := reflectVal.FieldByName("BaseResponse")
	baseResp, ok := baseRespReflect.Interface().(api.BaseResponse)
	if !ok {
		return nil
	}
	if baseResp.RetCode != 0 {
		return fmt.Errorf("ret code not zero, reqest id: %s, addr: %s, retcode: %d, message: %s",
			baseResp.RequestUUID, baseResp.RetCode, baseResp.Message)
	}
	return nil
}

func (c *HttpClient) prepareGetCodeRequest(addr string) (io.Reader, error) {
	getCodeReq := &api.GetLoginCodeRequest{
		Address: addr,
	}
	getCodeReq.Action = api.GET_LOGIN_CODE_LABEL
	reqBody, err := json.Marshal(getCodeReq)
	if err != nil {
		return nil, err
	}
	r := bytes.NewBuffer(reqBody)
	return r, nil
}

func (c *HttpClient) doGetCodeRequest(addr string) (string, int, error) {
	r, err := c.prepareGetCodeRequest(addr)
	if err != nil {
		return "", 0, err
	}
	res, err := c.client.Post(c.endpoint, "application/json", r)
	if err != nil {
		return "", 0, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", 0, err
	}
	getCodeResp := &api.GetLoginCodeResponse{}
	err = json.Unmarshal(body, getCodeResp)
	if err != nil {
		return "", 0, err
	}
	return getCodeResp.LoginCode, getCodeResp.Nonce, nil
}

func (c *HttpClient) prepareLoginRequest(addr string, nonce int, sig string) (io.Reader, error) {
	loginReq := &api.LoginDillRequest{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	}
	loginReq.Action = api.LOGIN_DILL_LABEL
	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		return nil, err
	}
	r := bytes.NewBuffer(reqBody)
	return r, nil
}

func (c *HttpClient) doLogin(w *wallet.Wallet) (string, error) {
	signingData, code, err := c.doGetCodeRequest(w.GetAddrHex())
	if err != nil {
		return "", err
	}
	sig, err := w.EthSign(signingData)
	if err != nil {
		return "", err
	}
	sigStr := hexutil.Encode(sig)
	r, err := c.prepareLoginRequest(w.GetAddrHex(), code, sigStr)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Post(c.endpoint, "application/json", r)
	if err != nil {
		return "", err
	}
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	loginResp := &api.LoginDillResponse{}
	err = json.Unmarshal(respBytes, loginResp)
	if err != nil {
		return "", err
	}
	return loginResp.RefreshToken, nil
}

func (c *HttpClient) ConfirmBattle(ctx context.Context, addr *types.PlayerAddress, gameID uint, roundNumber uint) error {
	req := &api.ConfirmBattleRequest{
		BaseRequest: api.BaseRequest{
			Action: api.CONFIRM_BATTLE_LABEL,
		},
		GameID:      uint32(gameID),
		Round:       roundNumber,
		TempAddress: addr.TemporaryAddress,
		Address:     addr.WalletAddress,
	}
	resp := &api.ConfirmBattleResponse{}
	err := c.doPost(req, resp)
	if err != nil {
		return fmt.Errorf("confirm battle failed, addr: %s, game id: %d, round number: %d, err: %w",
			addr.String(), gameID, roundNumber, err)
	}
	return nil
}

func (c *HttpClient) ContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error {
	return nil
}

func (c *HttpClient) ExitQueue(ctx context.Context, addr *types.PlayerAddress) error {
	return nil
}

func (c *HttpClient) GetBattleInfo(ctx context.Context, gameID uint, roundNumber uint) (*api.GetBattleInfoResponse, error) {
	req := &api.GetBattleInfoRequest{
		BaseRequest: api.BaseRequest{
			Action: api.GET_BATTLE_INFO_LABEL,
		},
		GameID:  uint32(gameID),
		Round:   uint32(roundNumber),
		Address: c.accountWallet.GetAddrHex(),
	}
	resp := &api.GetBattleInfoResponse{}
	c.doPost(req, resp)
	err := c.doPost(req, resp)
	if err != nil {
		return nil, fmt.Errorf("get battle info failed, game id: %d, round number: %d, err: %w", gameID, roundNumber, err)
	}
	return resp, nil
}

func (c *HttpClient) GetGamePhase(ctx context.Context, addr *types.PlayerAddress) (*api.GetGamePhaseResponse, error) {
	req := &api.GetGamePhaseRequest{
		BaseRequest: api.BaseRequest{
			Action: api.GET_GAME_PHASE_LABEL,
		},
		Address:     addr.WalletAddress,
		TempAddress: addr.TemporaryAddress,
	}
	resp := &api.GetGamePhaseResponse{}
	err := c.doPost(req, resp)
	if err != nil {
		return nil, fmt.Errorf("get game phase failed, addr: %s, err: %w", addr.String(), err)
	}
	return resp, nil
}

func (c *HttpClient) JoinQueue(ctx context.Context, addr *types.PlayerAddress) error {
	req := &api.JoinQueueRequest{
		BaseRequest: api.BaseRequest{
			Action: api.JOIN_QUEUE_LABEL,
		},
		Address:     addr.WalletAddress,
		TempAddress: addr.TemporaryAddress,
	}
	resp := &api.JoinQueueResponse{}
	err := c.doPost(req, resp)
	if err != nil {
		return fmt.Errorf("join queue failed, addr: %s, err: %w", addr.String(), err)
	}
	return nil
}
func (c *HttpClient) RefuseContinueGame(ctx context.Context, addr *types.PlayerAddress, gameID uint) error {
	req := &api.RefuseContinueGameRequest{
		BaseRequest: api.BaseRequest{
			Action: api.REFUSE_CONTINUE_GAME_LABEL,
		},
		GameID:      uint(gameID),
		TempAddress: addr.TemporaryAddress,
		Address:     addr.WalletAddress,
	}
	resp := &api.RefuseContinueGameResponse{}
	err := c.doPost(req, resp)
	if err != nil {
		return fmt.Errorf("refuse continue game failed, addr: %s, game id: %d, err: %w", addr.String(), gameID, err)
	}
	return nil
}
func (c *HttpClient) Subscribe(topic string, subscriberID string, evtChan chan *events.Event) error {
	addr := types.PlayerAddress{}
	err := addr.Parse(topic)
	if err != nil {
		return fmt.Errorf("failed to parse player address from topic: %s, err: %w", topic, err)
	}
	req := &api.SubscribeGameInfoRequest{
		BaseRequest: api.BaseRequest{
			Action: api.SUBSCRIBE_GAME_INFO_LABEL,
		},
		Address:     addr.WalletAddress,
		TempAddress: addr.TemporaryAddress,
		Duration:    86400, // 1 day in seconds
	}
	reqBody, err := json.Marshal(req)
	if err != nil {
		return err
	}
	r := bytes.NewBuffer(reqBody)
	go func() {
		for {
			resp, err := c.client.Post(c.endpoint, "application/json", r)
			if err != nil {
				log.Errorw("request SSE stream failed", "error", err)
			}
			if resp.StatusCode != http.StatusOK {
				log.Errorw("SSE stream returned non-200 status code", "status", resp.StatusCode)
			}

			reader := bufio.NewReader(resp.Body)
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						log.Error("SSE connection closed by server. Reconnecting...")
						break // Exit inner loop to trigger reconnection
					}
					log.Error("Error reading SSE stream: %v. Reconnecting...", err)
					break // Exit inner loop to trigger reconnection
				}

				line = strings.TrimSpace(line)
				if line == "" { // End of an event
					continue
				}

				// Basic parsing (can be extended for 'event', 'id' fields)
				if strings.HasPrefix(line, "data:") {
					data := strings.TrimPrefix(line, "data:")
					evt := events.Event{}
					err := json.Unmarshal([]byte(strings.TrimSpace(data)), &evt)
					if err != nil {
						log.Error("Error unmarshalling SSE event: %v", err)
						continue
					}
					// Send the event to the channel
					evtChan <- &evt
				}
			}
		}
	}()
	return nil
}

func (c *HttpClient) Unsubscribe(s string, subId string) error {
	return nil
}
