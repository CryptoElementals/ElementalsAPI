package utils

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignTokenCollectorWithdrawRecover(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	depositAddr := crypto.PubkeyToAddress(priv.PublicKey)
	amount := new(big.Int).Mul(big.NewInt(10), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	playerID := big.NewInt(1)
	nonce := big.NewInt(0)

	sig, err := SignTokenCollectorWithdraw(depositAddr, amount, playerID, nonce, priv)
	if err != nil {
		t.Fatal(err)
	}

	ok, err := VerifyTokenCollectorWithdraw(depositAddr, amount, playerID, nonce, sig)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("VerifyTokenCollectorWithdraw returned false")
	}

	payloadHash, err := TokenCollectorWithdrawPayloadHash(depositAddr, amount, playerID, nonce)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := recoverWithdrawSigner(payloadHash, sig)
	if err != nil {
		t.Fatal(err)
	}
	if recovered != depositAddr {
		t.Fatalf("recovered %s want %s", recovered.Hex(), depositAddr.Hex())
	}
}

func TestSignTokenCollectorWithdrawNonceAffectsHash(t *testing.T) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	depositAddr := crypto.PubkeyToAddress(priv.PublicKey)
	amount := big.NewInt(1)
	playerID := big.NewInt(42)

	h0, err := TokenCollectorWithdrawPayloadHash(depositAddr, amount, playerID, big.NewInt(0))
	if err != nil {
		t.Fatal(err)
	}
	h1, err := TokenCollectorWithdrawPayloadHash(depositAddr, amount, playerID, big.NewInt(1))
	if err != nil {
		t.Fatal(err)
	}
	if h0 == h1 {
		t.Fatal("payload hash should change when nonce changes")
	}
}
