package wallet

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var privatePath = "./private"

// var data = "Hello world"
// var data = "Welcome to PINEX"

var data = "Welcome to PINEX!\n\nThis request will not trigger a blockchain transaction or cost any gas fees. It is only used to authorise logging into PINEX.\n\nYour authentication status will reset after 12 hours.\n\nWallet address:\n0x9156541f2c715810E17c33209767D530978976E5\n\nNonce:\n954837"

// 0xac2195dfd7c50a82fce2b3683ad66b29aed47dffa7a5087605d0adff9d94392566f8c7fca50e988f981aa43729e43dfb7ceca5252831a2742862ed3bad3406371c

func TestNewWallet(t *testing.T) {
	w, err := NewWallet("")
	if err != nil {
		t.Fatalf("NewWallet err %s", err.Error())
	}
	fmt.Printf("priv key: %v\n", w.GetPrivateKeyHex())
	fmt.Printf("wallet address: %v\n", w.address)
}

func TestLoadWallet(t *testing.T) {
	w, err := LoadWallet(privatePath)
	if err != nil {
		t.Fatalf("NewWallet err %s", err.Error())
	}
	fmt.Printf("wallet address: %v\n", w.address)
}

func TestSign(t *testing.T) {
	w, err := LoadWallet(privatePath)
	if err != nil {
		t.Fatalf("NewWallet err %s", err.Error())
	}
	fmt.Printf("wallet address: %v\n", w.address)

	signature, err := w.Sign(data)
	if err != nil {
		t.Fatalf("Sign err %s", err.Error())
	}
	fmt.Printf("len(signature): %d\n", len(signature))
	fmt.Printf("signature: %v\n", signature)
	fmt.Printf("signature string: %x\n", string(signature))
}

func TestVerifySign(t *testing.T) {
	w, err := LoadWallet(privatePath)
	if err != nil {
		t.Fatalf("NewWallet err %s", err.Error())
	}
	fmt.Printf("wallet address: %v\n", w.address)

	signature, err := w.Sign(data)
	if err != nil {
		t.Fatalf("Sign err %s", err.Error())
	}
	fmt.Printf("len(signature): %d\n", len(signature))
	fmt.Printf("signature: %v\n", signature)
	fmt.Printf("signature string: %x\n", string(signature))

	b, err := w.Verify(data, signature)
	if err != nil {
		t.Fatalf("Verify err %s", err.Error())
	}

	if !b {
		t.Fatalf("Verify invalid")
	}
}

// Verify method of TestEthVerifySign is the same with metamask
func TestEthVerifySign(t *testing.T) {
	w, err := LoadWallet(privatePath)
	if err != nil {
		t.Fatalf("NewWallet err %s", err.Error())
	}
	fmt.Printf("wallet address: %v\n", w.address)

	signature, err := w.EthSign(data)
	if err != nil {
		t.Fatalf("Sign err %s", err.Error())
	}
	fmt.Printf("len(signature): %d\n", len(signature))
	fmt.Printf("signature: %v\n", signature)
	fmt.Printf("signature string: %x\n", string(signature))

	b, err := w.EthVerify(data, signature)
	if err != nil {
		t.Fatalf("Verify err %s", err.Error())
	}

	if !b {
		t.Fatalf("Verify invalid")
	}
}

func TestSimple(t *testing.T) {
	address := "0x9156541f2c715810E17c33209767D530978976E5"
	signature := []byte{
		0xac, 0x21, 0x95, 0xdf, 0xd7, 0xc5, 0x0a, 0x82, 0xfc, 0xe2, 0xb3, 0x68, 0x3a, 0xd6, 0x6b, 0x29,
		0xae, 0xd4, 0x7d, 0xff, 0xa7, 0xa5, 0x08, 0x76, 0x05, 0xd0, 0xad, 0xff, 0x9d, 0x94, 0x39, 0x25,
		0x66, 0xf8, 0xc7, 0xfc, 0xa5, 0x0e, 0x98, 0x8f, 0x98, 0x1a, 0xa4, 0x37, 0x29, 0xe4, 0x3d, 0xfb,
		0x7c, 0xec, 0xa5, 0x25, 0x28, 0x31, 0xa2, 0x74, 0x28, 0x62, 0xed, 0x3b, 0xad, 0x34, 0x06, 0x37,
		0x1c,
	}
	addr := common.HexToAddress(address)
	wallet := &Wallet{
		address: addr,
	}

	b, err := wallet.EthVerify(data, signature)
	if err != nil {
		t.Fatalf("Verify err %s", err.Error())
	}
	if !b {
		t.Fatalf("Verify invalid")
	}
	fmt.Println("Signature is valid!")
}
