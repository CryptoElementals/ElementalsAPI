package main

import (
	"encoding/base64"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/CryptoElementals/common/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var (
	signWithdrawDepositAddr string
	signWithdrawAmountWei   string
	signWithdrawPlayerID    string
	signWithdrawPrivKeyFile string
)

var contractSignWithdrawCmd = &cobra.Command{
	Use:   "sign-withdraw",
	Short: "Generate TokenCollector withdraw signature (65-byte r||s||v, v=27|28)",
	Long: `Generate an ECDSA signature for TokenCollector._withdraw.

Solidity checks:
  payloadHash = keccak256(abi.encodePacked(depositAddr, amount, playerId))
  ethSignedHash = keccak256(abi.encodePacked("\x19Ethereum Signed Message:\n32", payloadHash))
  ecrecover(ethSignedHash, v, r, s) == depositAddr

--deposit-addr must be Credited(playerId).depositAddr (the "to" in encodePacked) and must match the private key.
--amount-wei must equal the withdraw amount passed on-chain / to ledger RequestWithdraw (in wei).

Outputs:
  signature_hex    0x-prefixed 65 bytes for HTTP API / chain Withdraw calldata
  signature_base64 same bytes for grpcurl RequestWithdraw JSON "signature" field`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runContractSignWithdraw(); err != nil {
			fmt.Printf("sign withdraw failed: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	contractCmd.AddCommand(contractSignWithdrawCmd)

	contractSignWithdrawCmd.Flags().StringVar(&signWithdrawDepositAddr, "deposit-addr", "", "Credited.depositAddr (to in encodePacked); must match private key")
	contractSignWithdrawCmd.Flags().StringVar(&signWithdrawAmountWei, "amount-wei", "", "withdraw amount in wei")
	contractSignWithdrawCmd.Flags().StringVar(&signWithdrawPlayerID, "player-id", "", "player id")
	contractSignWithdrawCmd.Flags().StringVar(&signWithdrawPrivKeyFile, "private-key-file", "", "file containing private key hex (with or without 0x)")

	_ = contractSignWithdrawCmd.MarkFlagRequired("deposit-addr")
	_ = contractSignWithdrawCmd.MarkFlagRequired("amount-wei")
	_ = contractSignWithdrawCmd.MarkFlagRequired("player-id")
	_ = contractSignWithdrawCmd.MarkFlagRequired("private-key-file")
}

func runContractSignWithdraw() error {
	if !common.IsHexAddress(signWithdrawDepositAddr) {
		return fmt.Errorf("invalid deposit-addr: %s", signWithdrawDepositAddr)
	}
	depositAddr := common.HexToAddress(signWithdrawDepositAddr)

	amountWei, ok := new(big.Int).SetString(strings.TrimSpace(signWithdrawAmountWei), 10)
	if !ok || amountWei.Sign() <= 0 {
		return fmt.Errorf("invalid amount-wei: %s", signWithdrawAmountWei)
	}

	playerID, ok := new(big.Int).SetString(strings.TrimSpace(signWithdrawPlayerID), 10)
	if !ok || playerID.Sign() <= 0 {
		return fmt.Errorf("invalid player-id: %s", signWithdrawPlayerID)
	}

	privHex, err := loadPrivateKeyHexFromFile(signWithdrawPrivKeyFile)
	if err != nil {
		return err
	}
	priv, err := crypto.HexToECDSA(privHex)
	if err != nil {
		return fmt.Errorf("invalid private key hex: %w", err)
	}
	signerAddr := crypto.PubkeyToAddress(priv.PublicKey)
	if signerAddr != depositAddr {
		return fmt.Errorf("deposit-addr %s does not match private key address %s (must be Credited.depositAddr)", depositAddr.Hex(), signerAddr.Hex())
	}

	payloadHash, err := utils.TokenCollectorWithdrawPayloadHash(depositAddr, amountWei, playerID)
	if err != nil {
		return fmt.Errorf("payload hash: %w", err)
	}

	sig, err := utils.SignTokenCollectorWithdraw(depositAddr, amountWei, playerID, priv)
	if err != nil {
		return fmt.Errorf("sign withdraw payload: %w", err)
	}
	verified, err := utils.VerifyTokenCollectorWithdraw(depositAddr, amountWei, playerID, sig)
	if err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}
	if !verified {
		return fmt.Errorf("local ecrecover verification failed")
	}

	fmt.Printf("deposit_addr=%s\n", depositAddr.Hex())
	fmt.Printf("signer_address=%s\n", signerAddr.Hex())
	fmt.Printf("payload_hash=%s\n", payloadHash.Hex())
	fmt.Printf("signature_len=%d\n", len(sig))
	fmt.Printf("signature_hex=%s\n", "0x"+common.Bytes2Hex(sig))
	fmt.Printf("signature_base64=%s\n", base64.StdEncoding.EncodeToString(sig))
	fmt.Printf("signature_verified=true\n")
	return nil
}

func loadPrivateKeyHexFromFile(path string) (string, error) {
	rawPath := strings.TrimSpace(path)
	if rawPath == "" {
		return "", fmt.Errorf("private-key-file is required")
	}
	content, err := os.ReadFile(rawPath)
	if err != nil {
		return "", fmt.Errorf("read private-key-file: %w", err)
	}
	pk := strings.TrimSpace(string(content))
	pk = strings.TrimPrefix(pk, "0x")
	if pk == "" {
		return "", fmt.Errorf("private-key-file is empty")
	}
	return pk, nil
}
