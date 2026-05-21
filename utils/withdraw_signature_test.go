package utils

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts"
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

	sig, err := SignTokenCollectorWithdraw(depositAddr, amount, playerID, priv)
	if err != nil {
		t.Fatal(err)
	}

	payloadHash, err := SolidityPackedKeccak256([]any{depositAddr, amount, playerID})
	if err != nil {
		t.Fatal(err)
	}
	sigCopy := append([]byte(nil), sig...)
	if sigCopy[crypto.RecoveryIDOffset] >= 27 {
		sigCopy[crypto.RecoveryIDOffset] -= 27
	}
	pub, err := crypto.SigToPub(accounts.TextHash(payloadHash.Bytes()), sigCopy)
	if err != nil {
		t.Fatal(err)
	}
	if crypto.PubkeyToAddress(*pub) != depositAddr {
		t.Fatalf("recovered %s want %s", crypto.PubkeyToAddress(*pub).Hex(), depositAddr.Hex())
	}
}
