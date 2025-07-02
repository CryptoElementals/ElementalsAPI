package e2etest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"testing"
	"time"

	"github.com/CryptoElementals/common/cache"
	"github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/api/login"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

const testSessionName = "e2e_test"
const testSessionExpire = 3
const testRefreshExpire = 5

func prepareFooRequest(t *testing.T) io.Reader {
	fooReq := &TestFooRequest{}
	fooReq.Action = TestAction
	fooReq.Foo = "FOO"
	reqBody, err := json.Marshal(fooReq)
	require.NoError(t, err)
	r := bytes.NewBuffer(reqBody)
	return r
}

func prepareGetCodeRequest(t *testing.T, addr string) io.Reader {
	getCodeReq := &login.GetLoginCodeRequest{
		Address: addr,
	}
	getCodeReq.Action = login.GET_LOGIN_CODE_LABEL
	reqBody, err := json.Marshal(getCodeReq)
	require.NoError(t, err)
	r := bytes.NewBuffer(reqBody)
	return r
}

func doGetCodeRequest(t *testing.T, client *http.Client, addr string) (string, int) {
	r := prepareGetCodeRequest(t, addr)
	res, err := client.Post("http://localhost:19999", "application/json", r)
	require.NoError(t, err)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NotNil(t, res)
	getCodeResp := &login.GetLoginCodeResponse{}
	err = json.Unmarshal(body, getCodeResp)
	require.NoError(t, err)
	return getCodeResp.LoginCode, getCodeResp.Nonce
}

func prepareLoginRequest(t *testing.T, addr string, nonce int, sig string) io.Reader {
	loginReq := &login.LoginDillRequest{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	}
	loginReq.Action = login.LOGIN_DILL_LABEL
	reqBody, err := json.Marshal(loginReq)
	require.NoError(t, err)
	r := bytes.NewBuffer(reqBody)
	return r
}

func checkCookieSet(t *testing.T, resp *http.Response) {
	hasSet := false
	cookies := resp.Cookies()
	for _, c := range cookies {
		if c.Name == testSessionName+"_session" {
			hasSet = true
		}
	}
	require.True(t, hasSet)
}

func doLogin(t *testing.T, w *wallet.Wallet, client *http.Client) string {
	signingData, code := doGetCodeRequest(t, client, w.GetAddrHex())
	sig, err := w.EthSign(signingData)
	require.NoError(t, err)
	sigStr := hexutil.Encode(sig)
	r := prepareLoginRequest(t, w.GetAddrHex(), code, sigStr)
	resp, err := client.Post("http://localhost:19999", "application/json", r)
	require.NoError(t, err)
	require.Equal(t, resp.StatusCode, 200)
	checkCookieSet(t, resp)
	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	loginResp := &login.LoginDillResponse{}
	err = json.Unmarshal(respBytes, loginResp)
	require.NoError(t, err)
	require.NotEmpty(t, loginResp.RefreshToken)
	require.Equal(t, loginResp.RefreshTokenExpiresIn, testRefreshExpire)
	return loginResp.RefreshToken
}

func doFooBar(t *testing.T, client *http.Client, expectedCode int, checkBody bool) {
	r := prepareFooRequest(t)
	resp, err := client.Post("http://localhost:19999", "application/json", r)
	require.NoError(t, err)
	require.Equal(t, expectedCode, resp.StatusCode)
	if !checkBody {
		return
	}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	barResp := &TestbarResponse{}
	err = json.Unmarshal(body, barResp)
	require.NoError(t, err)
	require.Equal(t, "BAR", barResp.Bar)
}

func doRefresh(t *testing.T, client *http.Client, expectedCode int, refreshToken string) *http.Response {
	refreshReq := login.RefreshDillRequest{
		RefreshToken: refreshToken,
	}
	refreshReq.Action = login.REFRESH_LABEL
	reqBody, err := json.Marshal(refreshReq)
	require.NoError(t, err)
	r := bytes.NewBuffer(reqBody)
	resp, err := client.Post("http://localhost:19999", "application/json", r)
	require.NoError(t, err)
	require.Equal(t, expectedCode, resp.StatusCode)
	return resp
}

func checkRefreshRespSuccessBody(t *testing.T, resp *http.Response, token string) {
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	refreshResp := &login.RefreshDillResponse{}
	err = json.Unmarshal(respBody, refreshResp)
	require.NoError(t, err)
	require.Equal(t, token, refreshResp.RefreshToken)
	require.Equal(t, testRefreshExpire, refreshResp.RefreshTokenExpiresIn)
}

func prepareMockServer() func() error {
	log.InitGlobalLogger(&log.Config{Development: true})
	EnableTestApi()
	cfg := &server.Config{
		Port:               19999,
		ServerMode:         "debug",
		SessionMaxAge:      testSessionExpire,
		RefreshTokenMaxAge: testRefreshExpire,
		ServiceName:        testSessionName,
	}

	svr := server.New(cfg, server.DefaultSessionStore(), cache.NewMemCache())
	svr.Run()
	return svr.Stop
}

func TestNoLoginFailed(t *testing.T) {
	stop := prepareMockServer()
	defer stop()
	doFooBar(t, http.DefaultClient, 401, false)
}

func TestLoginSuccess(t *testing.T) {
	stop := prepareMockServer()
	defer stop()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
	}
	w, err := wallet.NewWallet("")
	require.NoError(t, err)
	doLogin(t, w, client)
}

func TestLoginFooBarSuccess(t *testing.T) {
	stop := prepareMockServer()
	defer stop()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
	}
	w, err := wallet.NewWallet("")
	require.NoError(t, err)
	doLogin(t, w, client)
	doFooBar(t, client, 200, true)
}

func TestLoginExpireFooBarFailed(t *testing.T) {
	stop := prepareMockServer()
	defer stop()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
	}
	w, err := wallet.NewWallet("")
	require.NoError(t, err)
	doLogin(t, w, client)
	time.Sleep((testSessionExpire + 1) * time.Second)
	doFooBar(t, client, 401, false)
}

func TestFooBarSuccessAfterRefresh(t *testing.T) {
	stop := prepareMockServer()
	defer stop()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
	}
	w, err := wallet.NewWallet("")
	require.NoError(t, err)
	refreshToken := doLogin(t, w, client)
	time.Sleep((testSessionExpire + 1) * time.Second)
	doFooBar(t, client, 401, false)
	refreshResp := doRefresh(t, client, 200, refreshToken)
	checkCookieSet(t, refreshResp)
	checkRefreshRespSuccessBody(t, refreshResp, refreshToken)
	doFooBar(t, client, 200, true)
}

func TestRefreshFailureDueToExpire(t *testing.T) {
	stop := prepareMockServer()
	defer stop()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
	}
	w, err := wallet.NewWallet("")
	require.NoError(t, err)
	refreshToken := doLogin(t, w, client)
	time.Sleep((testRefreshExpire + 1) * time.Second)
	resp := doRefresh(t, client, 200, refreshToken)
	refreshResp := &login.RefreshDillResponse{}
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, refreshResp)
	require.NoError(t, err)
	refreshErr := errors.RefreshTokenInvalid(refreshToken)

	// server returns action error
	actionErr := errors.ActionError(refreshErr.Error())
	errResp := api.MakeErrorResponse(actionErr)
	require.EqualValues(t, actionErr.Code(), refreshResp.RetCode)
	require.Equal(t, errResp.Message, refreshResp.Message)
	doFooBar(t, client, 401, false)
}
