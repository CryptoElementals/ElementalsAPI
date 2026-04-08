package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth_type "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type Wallet struct {
	privateKey *ecdsa.PrivateKey
	address    common.Address
}

func NewWallet(priKeyPath string) (*Wallet, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}

	if len(priKeyPath) > 0 {
		err := crypto.SaveECDSA(priKeyPath, privateKey)
		if err != nil {
			return nil, err
		}
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	wallet := &Wallet{
		privateKey: privateKey,
		address:    address,
	}

	return wallet, nil
}

func NewWalletFromAddr(addr common.Address) *Wallet {
	wallet := &Wallet{
		address: addr,
	}

	return wallet
}

func NewWalletFromPrivKey(privateKey *ecdsa.PrivateKey) *Wallet {
	wallet := &Wallet{
		privateKey: privateKey,
	}
	publicKey := privateKey.Public()
	publicKeyECDSA := publicKey.(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	wallet.address = address
	return wallet
}

func NewWalletFromPrivKeyHex(privateKeyHex string) (*Wallet, error) {
	privateKeyHex = strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, err
	}
	return NewWalletFromPrivKey(privateKey), nil
}

func LoadWallet(priKeyPath string) (*Wallet, error) {
	privateKey, err := crypto.LoadECDSA(priKeyPath)
	if err != nil {
		return nil, err
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	wallet := &Wallet{
		privateKey: privateKey,
		address:    address,
	}

	return wallet, nil
}

func (wallet *Wallet) Sign(message string) (signature []byte, err error) {
	hash := crypto.Keccak256Hash([]byte(message))
	return crypto.Sign(hash.Bytes(), wallet.privateKey) // sign the hash of data
}

func (wallet *Wallet) Verify(message string, signature []byte) (bool, error) {
	hash := crypto.Keccak256Hash([]byte(message))

	sigPublicKey, err := crypto.SigToPub(hash.Bytes(), signature)
	if err != nil {
		return false, err
	}

	sigAddr := crypto.PubkeyToAddress(*sigPublicKey)

	return bytes.Equal(sigAddr.Bytes(), wallet.address.Bytes()), nil
}

func (wallet *Wallet) EthSign(message string) (signature []byte, err error) {
	hash := crypto.Keccak256Hash([]byte("\x19Ethereum Signed Message:\n" + strconv.Itoa(len(message)) + message))
	sig, err := crypto.Sign(hash.Bytes(), wallet.privateKey) // sign the hash of data
	if err != nil {
		return nil, err
	}
	sig[crypto.RecoveryIDOffset] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper
	return sig, nil
}

/*
hash = keccak256("\x19Ethereum Signed Message:\n"${message length}${message})
addr = ecrecover(hash, signature)
*/
func (wallet *Wallet) EthVerify(message string, sig []byte) (bool, error) {
	if len(sig) != crypto.SignatureLength {
		return false, fmt.Errorf("signature must be %d bytes long", crypto.SignatureLength)
	}
	if sig[crypto.RecoveryIDOffset] != 27 && sig[crypto.RecoveryIDOffset] != 28 {
		return false, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1

	fmt.Printf("message: %s\n", message)
	fmt.Printf("len(sig): %d\n", len(sig))
	fmt.Printf("sig: %s\n", string(sig))
	fmt.Printf("sig: %v\n", sig)
	rpk, err := crypto.SigToPub(accounts.TextHash([]byte(message)), sig)
	if err != nil {
		return false, err
	}

	sigAddr := crypto.PubkeyToAddress(*rpk)
	fmt.Printf("sigAddr: %s\n", sigAddr.String())
	fmt.Printf("address: %s\n", string(wallet.address.Bytes()))

	return bytes.Equal(sigAddr.Bytes(), wallet.address.Bytes()), nil
}

func EthVerify(message string, sig []byte, address []byte) (bool, error) {
	if len(sig) != crypto.SignatureLength {
		return false, fmt.Errorf("signature must be %d bytes long", crypto.SignatureLength)
	}
	if sig[crypto.RecoveryIDOffset] != 27 && sig[crypto.RecoveryIDOffset] != 28 {
		return false, fmt.Errorf("invalid Ethereum signature (V is not 27 or 28)")
	}
	sig[crypto.RecoveryIDOffset] -= 27 // Transform yellow paper V from 27/28 to 0/1

	// fmt.Printf("message: %s\n", message)
	// fmt.Printf("len(sig): %d\n", len(sig))
	// fmt.Printf("sig: %s\n", string(sig))
	// fmt.Printf("sig: %v\n", sig)
	rpk, err := crypto.SigToPub(accounts.TextHash([]byte(message)), sig)
	if err != nil {
		return false, err
	}

	sigAddr := crypto.PubkeyToAddress(*rpk)
	// fmt.Printf("sigAddr: %s\n", sigAddr.String())
	// fmt.Printf("address: %s\n", string(address))

	return bytes.Equal(sigAddr.Bytes(), address), nil
}

func (wallet *Wallet) GetAddrHex() string {
	return wallet.address.Hex()
}

func (wallet *Wallet) GetAddr() common.Address {
	return wallet.address
}

func (wallet *Wallet) GetPrivateKeyHex() string {
	privateKeyBytes := crypto.FromECDSA(wallet.privateKey)
	return hexutil.Encode(privateKeyBytes)[2:]
}

func (wallet *Wallet) GetPrivateKey() *ecdsa.PrivateKey {
	return wallet.privateKey
}

func (wallet *Wallet) SignTx(tx *eth_type.Transaction, chainId *big.Int) (*eth_type.Transaction, error) {
	return eth_type.SignTx(tx, eth_type.NewCancunSigner(chainId), wallet.privateKey)
}

func (w *Wallet) BuildTxSinger(chainId *big.Int) func(a common.Address, t *eth_type.Transaction) (*eth_type.Transaction, error) {
	return func(a common.Address, t *eth_type.Transaction) (*eth_type.Transaction, error) {
		return w.SignTx(t, chainId)
	}
}
