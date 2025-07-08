package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"flag"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	privateKeyHex := flag.String("private", "", "私钥十六进制字符串")
	flag.Parse()

	if *privateKeyHex == "" {
		fmt.Println("请提供私钥参数: -private <私钥十六进制>")
		return
	}

	// 解析私钥
	privateKeyBytes, err := hex.DecodeString(*privateKeyHex)
	if err != nil {
		log.Fatal("私钥格式错误:", err)
	}

	// 创建私钥对象
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		log.Fatal("私钥无效:", err)
	}

	// 从私钥获取公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("无法获取公钥")
	}

	// 从公钥生成地址
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	fmt.Printf("私钥: %s\n", *privateKeyHex)
	fmt.Printf("地址: %s\n", address.Hex())
}
