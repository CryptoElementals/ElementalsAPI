package dnet

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

func SendHttpPostRequest(url_path string, body_type string, body io.Reader, timeOut uint32) (res []byte, err error) {
	client := newTimeoutHTTPClient(time.Duration(timeOut) * time.Second)
	result, err := client.Post(url_path, body_type, body)
	if err != nil {
		return
	}
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK { // 检查状态码是否为 200
		return nil, fmt.Errorf("HTTP request failed with status: %s", result.Status)
	}

	res, err = ioutil.ReadAll(result.Body)
	return
}

func SendHttpGetRequest(url_path string, timeOut uint32) (res []byte, err error) {
	client := newTimeoutHTTPClient(time.Duration(timeOut) * time.Second)
	result, err := client.Get(url_path)
	if err != nil {
		return
	}
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK { // 检查状态码是否为 200
		return nil, fmt.Errorf("HTTP request failed with status: %s", result.Status)
	}

	res, err = ioutil.ReadAll(result.Body)
	return
}

func SendHttpMethodRequest(method string, url_path string, body io.Reader, timeOut uint32) (res []byte, err error) {
	http_request, err := http.NewRequest(method, url_path, body)
	if err != nil {
		return
	}
	client := newTimeoutHTTPClient(time.Duration(timeOut) * time.Second)
	result, err := client.Do(http_request)
	if err != nil {
		return
	}
	defer result.Body.Close()

	if result.StatusCode != http.StatusOK { // 检查状态码是否为 200
		return nil, fmt.Errorf("HTTP request failed with status: %s", result.Status)
	}

	res, err = ioutil.ReadAll(result.Body)
	return
}

func dialHTTPTimeout(timeOut time.Duration) func(net, addr string) (net.Conn, error) {
	return func(network, addr string) (c net.Conn, err error) {
		c, err = net.DialTimeout(network, addr, timeOut)
		if err != nil {
			return
		}
		if timeOut > 0 {
			c.SetDeadline(time.Now().UTC().Add(timeOut))
		}
		return
	}
}

func newTimeoutHTTPClient(timeOut time.Duration) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial:              dialHTTPTimeout(timeOut),
			DisableKeepAlives: true,
		},
	}
}
