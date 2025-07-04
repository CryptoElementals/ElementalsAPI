package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path"
	"time"

	cerrs "github.com/CryptoElementals/common/errors"
	"github.com/CryptoElementals/common/log"
	"github.com/CryptoElementals/common/server/api"
	"github.com/CryptoElementals/common/server/api/login"
	e2etest "github.com/CryptoElementals/common/server/e2e_test"
	"github.com/CryptoElementals/common/wallet"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var w *wallet.Wallet
var client *http.Client
var addr string
var endpoint string

func main() {
	log.InitGlobalLogger(&log.Config{Development: true})
	args := os.Args
	if len(args) != 3 {
		fmt.Printf("usage: %s <private-key-path> port", args[0])
		os.Exit(1)
	}
	endpoint = "http://localhost:" + args[2]
	privateKeyPath := args[1]
	info, err := os.Stat(privateKeyPath)
	if errors.Is(err, os.ErrNotExist) {
		w, err = wallet.NewWallet(privateKeyPath)
		if err != nil {
			log.Fatal(err)
		}
	} else if err != nil {
		log.Fatal(err)
	} else {
		if info.IsDir() {
			privateKeyPath = path.Join(privateKeyPath, "priv_key")
		}
		w, err = wallet.LoadWallet(privateKeyPath)
		if err != nil {
			log.Fatal(err)
		}
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	client = &http.Client{
		Jar: jar,
	}
	addr = w.GetAddrHex()

	log.Info("foo request fails due to no login")
	// foo request failed due to node login
	doFooBar(401, false)

	log.Info("foo request success after login")
	// login
	refreshToken, refreshTokenExpireTimeStamp, accessTokenExpire := doLogin()
	// add one second to gurantee
	refreshTokenExpireAt := time.Unix(refreshTokenExpireTimeStamp+1, 0)
	accessTokenExpireAt := time.Now().Add(time.Duration(accessTokenExpire+1) * time.Second)
	// foo request success
	doFooBar(200, true)

	log.Info("foo request fails after accessk key expire")
	// wait until access token expire
	waitUntil(accessTokenExpireAt)
	// foo request fail
	doFooBar(401, false)

	log.Info("foo request success after refresh")
	// refresh access token
	refrehResp := doRefresh(200, refreshToken)
	checkRefreshRespSuccessBody(refrehResp, refreshToken)
	// foo request success
	doFooBar(200, true)

	log.Info("refresh success after first refresh time exceeded")
	// wait until first refresh token expire
	waitUntil(refreshTokenExpireAt)
	// can still refresh
	refrehResp = doRefresh(200, refreshToken)
	newRefreshTokenExpireTimestamp := checkRefreshRespSuccessBody(refrehResp, refreshToken)
	newRefreshTokenExpireAt := time.Unix(newRefreshTokenExpireTimestamp+1, 0)
	// foo request success
	doFooBar(200, true)

	log.Info("refresh failed after last refresh time exceeded")
	// wait until the latest refresh token expire
	waitUntil(newRefreshTokenExpireAt)
	refrehResp = doRefresh(200, refreshToken)
	checkRefreshRespFailBody(refrehResp, refreshToken)
	// access key expire should be shorter, so foo request fail
	doFooBar(401, false)

	log.Info("foo request success after re-login")
	// login and recheck
	doLogin()
	doFooBar(200, true)
}

func waitUntil(t time.Time) {
	time.Sleep(time.Until(t))
}

func prepareFooRequest() io.Reader {
	fooReq := &e2etest.TestFooRequest{}
	fooReq.Action = e2etest.TestAction
	fooReq.Foo = "FOO"
	reqBody, err := json.Marshal(fooReq)
	if err != nil {
		log.Fatal(err)
	}
	r := bytes.NewBuffer(reqBody)
	return r
}

func prepareGetCodeRequest(addr string) io.Reader {
	getCodeReq := &login.GetLoginCodeRequest{
		Address: addr,
	}
	getCodeReq.Action = login.GET_LOGIN_CODE_LABEL
	reqBody, err := json.Marshal(getCodeReq)
	if err != nil {
		log.Fatal(err)
	}
	r := bytes.NewBuffer(reqBody)
	return r
}

func doGetCodeRequest() (string, int) {
	r := prepareGetCodeRequest(addr)
	res, err := client.Post(endpoint, "application/json", r)
	if err != nil {
		log.Fatal(err)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	getCodeResp := &login.GetLoginCodeResponse{}
	err = json.Unmarshal(body, getCodeResp)
	if err != nil {
		log.Fatal(err)
	}
	return getCodeResp.LoginCode, getCodeResp.Nonce
}

func prepareLoginRequest(nonce int, sig string) io.Reader {
	loginReq := &login.LoginDillRequest{
		Address:   addr,
		Nonce:     nonce,
		Signature: sig,
	}
	loginReq.Action = login.LOGIN_DILL_LABEL
	reqBody, err := json.Marshal(loginReq)
	if err != nil {
		log.Fatal(err)
	}
	r := bytes.NewBuffer(reqBody)
	return r
}

func checkCookieSet(resp *http.Response) int {
	cookies := resp.Cookies()
	for _, c := range cookies {
		if c.Name == "test-service"+"_session" {
			return int(time.Until(c.Expires) / time.Second)
		}
	}
	log.Fatal("cookie not set")
	return 0
}

func doLogin() (string, int64, int) {
	signingData, code := doGetCodeRequest()
	sig, err := w.EthSign(signingData)
	if err != nil {
		log.Fatal(err)
	}
	sigStr := hexutil.Encode(sig)
	r := prepareLoginRequest(code, sigStr)
	resp, err := client.Post(endpoint, "application/json", r)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode != 200 {
		log.Fatal("http not 200")
	}
	accessTokenExpire := checkCookieSet(resp)
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	loginResp := &login.LoginDillResponse{}
	err = json.Unmarshal(respBytes, loginResp)
	if err != nil {
		log.Fatal(err)
	}
	if loginResp.RefreshToken == "" {
		log.Fatal("refresh token empty")
	}
	log.Infof("refresh token expire at: %d", loginResp.RefreshTokenExpirationTime)
	return loginResp.RefreshToken, loginResp.RefreshTokenExpirationTime, accessTokenExpire
}

func doFooBar(expectedCode int, checkBody bool) {
	r := prepareFooRequest()
	resp, err := client.Post(endpoint, "application/json", r)
	if err != nil {
		log.Fatal(err)
	}
	if expectedCode != resp.StatusCode {
		log.Fatal("code not equal")
	}
	if !checkBody {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	barResp := &e2etest.TestbarResponse{}
	err = json.Unmarshal(body, barResp)
	if err != nil {
		log.Fatal(err)
	}
	if barResp.Bar != "BAR" {
		log.Fatal("response not BAR")
	}
}

func doRefresh(expectedCode int, refreshToken string) *http.Response {
	refreshReq := login.RefreshDillRequest{
		RefreshToken: refreshToken,
	}
	refreshReq.Action = login.REFRESH_LABEL
	reqBody, err := json.Marshal(refreshReq)
	if err != nil {
		log.Fatal(err)
	}
	r := bytes.NewBuffer(reqBody)
	resp, err := client.Post(endpoint, "application/json", r)
	if err != nil {
		log.Fatal(err)
	}
	if expectedCode != resp.StatusCode {
		log.Fatal("code not equal")
	}
	return resp
}

func checkRefreshRespSuccessBody(resp *http.Response, token string) int64 {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	refreshResp := &login.RefreshDillResponse{}
	err = json.Unmarshal(respBody, refreshResp)
	if err != nil {
		log.Fatal(err)
	}
	if token != refreshResp.RefreshToken {
		log.Fatal("refresh token not equal")
	}
	return refreshResp.RefreshTokenExpirationTime
}

func checkRefreshRespFailBody(resp *http.Response, token string) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	refreshResp := &login.RefreshDillResponse{}
	err = json.Unmarshal(respBody, refreshResp)
	if err != nil {
		log.Fatal(err)
	}
	refreshErr := cerrs.RefreshTokenInvalid(token)

	// server returns action error
	actionErr := cerrs.ActionError(refreshErr.Error())
	errResp := api.MakeErrorResponse(actionErr)
	if int(actionErr.Code()) != refreshResp.RetCode {
		log.Fatal("code not equal")
	}
	if errResp.Message != refreshResp.Message {
		log.Fatal("response not equal")
	}
}
