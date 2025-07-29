package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	contract "github.com/CryptoElementals/common/contracts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

// submitterTestCmd represents the submitter-test command
var submitterTestCmd = &cobra.Command{
	Use:   "submitter-test",
	Short: "测试卡牌提交功能",
	Long: `submitter-test 命令用于测试向区块链提交卡牌哈希和卡牌信息的功能。

支持两种操作：
1. submit-hash: 提交卡牌哈希到智能合约
2. submit-cards: 提交实际卡牌内容和盐值到智能合约`,
}

var (
	// 命令行参数
	rpcEndpoint  string
	contractAddr string
	userAddr     string
	tempAddr     string
	round        uint64
	privateKey   string
)

// submitHashCmd 提交卡牌哈希命令
var submitHashCmd = &cobra.Command{
	Use:   "submit-hash [卡牌1] [卡牌2] [卡牌3]",
	Short: "提交卡牌哈希到智能合约",
	Long:  `根据卡牌内容计算哈希并提交到智能合约`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSubmitHash(args[0], args[1], args[2])
	},
}

// submitCardsCmd 提交卡牌命令
var submitCardsCmd = &cobra.Command{
	Use:   "submit-cards [卡牌1] [卡牌2] [卡牌3]",
	Short: "提交卡牌到智能合约",
	Long:  `向指定的智能合约提交三个卡牌参数和盐值`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSubmitCards(args[0], args[1], args[2])
	},
}

func init() {
	rootCmd.AddCommand(submitterTestCmd)
	submitterTestCmd.AddCommand(submitHashCmd)
	submitterTestCmd.AddCommand(submitCardsCmd)

	// 添加全局标志
	submitterTestCmd.PersistentFlags().StringVarP(&rpcEndpoint, "rpc", "r", "http://152.32.231.145:8545", "区块链RPC端点")
	submitterTestCmd.PersistentFlags().StringVarP(&contractAddr, "contract", "a", "", "合约地址")
	submitterTestCmd.PersistentFlags().StringVarP(&userAddr, "user", "u", "", "用户地址")
	submitterTestCmd.PersistentFlags().StringVarP(&tempAddr, "temp", "t", "", "临时地址")
	submitterTestCmd.PersistentFlags().Uint64VarP(&round, "round", "n", 1, "回合数")
	submitterTestCmd.PersistentFlags().StringVarP(&privateKey, "private-key", "p", "", "钱包私钥")

	// 标记必需参数
	submitterTestCmd.MarkPersistentFlagRequired("contract")
	submitterTestCmd.MarkPersistentFlagRequired("user")
	submitterTestCmd.MarkPersistentFlagRequired("temp")
	submitterTestCmd.MarkPersistentFlagRequired("private-key")
}

// 解析私钥
func parsePrivateKey(privateKeyHex string) (*ecdsa.PrivateKey, error) {
	privateKeyBytes := common.FromHex(privateKeyHex)
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %v", err)
	}
	return privateKey, nil
}

// 运行提交卡牌哈希
func runSubmitHash(card1, card2, card3 string) error {
	// 连接区块链
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		return fmt.Errorf("连接区块链失败: %v", err)
	}

	// 获取链ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("获取链ID失败: %v", err)
	}

	// 创建合约实例
	contractInstance, err := contract.NewRoomContract(common.HexToAddress(contractAddr), client)
	if err != nil {
		return fmt.Errorf("创建合约实例失败: %v", err)
	}

	// 组合卡牌
	cards := strings.Join([]string{card1, card2, card3}, ",")
	salt := "salt"

	// 计算承诺哈希
	hh := sha3.NewLegacyKeccak256()
	hh.Write([]byte(cards))
	hh.Write([]byte(salt))
	commitment := hh.Sum(nil)

	fmt.Printf("卡牌组合: %s\n", cards)
	fmt.Printf("盐值: %s\n", salt)
	fmt.Printf("计算出的承诺哈希: %x\n", commitment)

	// 解析私钥
	privateKey, err := parsePrivateKey(privateKey)
	if err != nil {
		return err
	}

	// 创建交易选项
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("创建交易选项失败: %v", err)
	}

	// 设置发送方地址为指定的临时地址（转换为小写）
	auth.From = common.HexToAddress(strings.ToLower(tempAddr))

	fmt.Printf("提交卡牌哈希，回合: %d\n", round)
	fmt.Printf("卡牌哈希: %s\n", commitment)
	fmt.Printf("发送方地址: %s\n", auth.From.Hex())

	// 提交交易
	tx, err := contractInstance.SubmitCardsHash(auth, [32]byte(commitment), big.NewInt(int64(round)))
	if err != nil {
		return fmt.Errorf("提交卡牌哈希失败: %v", err)
	}

	fmt.Printf("交易提交成功，交易哈希: %s\n", tx.Hash().Hex())
	fmt.Printf("等待交易确认...\n")

	// 等待交易确认
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return fmt.Errorf("等待交易确认失败: %v", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("交易执行失败")
	}

	fmt.Printf("交易确认成功，区块号: %d\n", receipt.BlockNumber)
	return nil
}

// 运行提交卡牌
func runSubmitCards(card1, card2, card3 string) error {
	// 连接区块链
	client, err := ethclient.Dial(rpcEndpoint)
	if err != nil {
		return fmt.Errorf("连接区块链失败: %v", err)
	}

	// 获取链ID
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return fmt.Errorf("获取链ID失败: %v", err)
	}

	// 创建合约实例
	contractInstance, err := contract.NewRoomContract(common.HexToAddress(contractAddr), client)
	if err != nil {
		return fmt.Errorf("创建合约实例失败: %v", err)
	}

	// 组合卡牌
	cards := strings.Join([]string{card1, card2, card3}, ",")
	salt := "salt"

	// 计算承诺哈希
	hh := sha3.NewLegacyKeccak256()
	hh.Write([]byte(cards))
	hh.Write([]byte(salt))
	commitment := hh.Sum(nil)

	fmt.Printf("卡牌组合: %s\n", cards)
	fmt.Printf("盐值: %s\n", salt)
	fmt.Printf("计算出的承诺哈希: %x\n", commitment)

	// 解析私钥
	privateKey, err := parsePrivateKey(privateKey)
	if err != nil {
		return err
	}

	// 创建交易选项
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("创建交易选项失败: %v", err)
	}

	// 设置发送方地址为指定的临时地址（转换为小写）
	auth.From = common.HexToAddress(strings.ToLower(tempAddr))

	fmt.Printf("提交卡牌，回合: %d\n", round)
	fmt.Printf("发送方地址: %s\n", auth.From.Hex())

	// 提交交易
	tx, err := contractInstance.SubmitCards(auth, cards, salt, big.NewInt(int64(round)))
	if err != nil {
		return fmt.Errorf("提交卡牌失败: %v", err)
	}

	fmt.Printf("交易提交成功，交易哈希: %s\n", tx.Hash().Hex())
	fmt.Printf("等待交易确认...\n")

	// 等待交易确认
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return fmt.Errorf("等待交易确认失败: %v", err)
	}

	if receipt.Status == 0 {
		return fmt.Errorf("交易执行失败")
	}

	fmt.Printf("交易确认成功，区块号: %d\n", receipt.BlockNumber)
	return nil
}
