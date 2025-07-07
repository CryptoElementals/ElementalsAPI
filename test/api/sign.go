package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/CryptoElementals/common/wallet"
)

func main() {
	address := flag.String("address", "", "钱包地址")
	privatePath := flag.String("private", "", "私钥文件路径")
	nonce := flag.Int("nonce", 0, "nonce值")
	flag.Parse()

	if *address == "" || *privatePath == "" || *nonce == 0 {
		fmt.Println("参数缺失，示例: go run test/sign.go -address 0x... -private /path/to/key -nonce 123456")
		os.Exit(1)
	}

	w, err := wallet.LoadWallet(*privatePath)
	if err != nil {
		fmt.Printf("加载钱包失败: %v\n", err)
		os.Exit(1)
	}

	// 构造签名模板
	str := getSigningData(*address, *nonce)

	signature, err := w.EthSign(str)
	if err != nil {
		fmt.Printf("签名失败: %v\n", err)
		os.Exit(1)
	}

	// 强制修正V值
	if signature[len(signature)-1] < 27 {
		signature[len(signature)-1] += 27
		fmt.Printf("V值已修正为: %d\n", signature[len(signature)-1])
	}

	fmt.Printf("签名原始字节: %v\n", signature)
	fmt.Printf("签名长度: %d\n", len(signature))
	fmt.Printf("签名最后一字节(V值): %d\n", signature[len(signature)-1])
	fmt.Printf("签名(hex): %x\n", signature)
}

// 参考服务端签名模板
func getSigningData(addr string, nonce int) string {
	template := `Welcome to beast-royale-server!

This request will not trigger a blockchain transaction or cost any gas fees. It is only used to authorise logging into beast-royale-server.

Your authentication status will reset after 12 hours.

Wallet address:
ADDRESS

Nonce:
NONCE`
	str := strings.ReplaceAll(template, "ADDRESS", addr)
	str = strings.ReplaceAll(str, "NONCE", strconv.Itoa(nonce))
	return str
}
