package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

// BatchTransfer 合约 ABI 中的关键函数定义
const batchTransferABI = `[{"inputs":[{"internalType":"address[]","name":"recipients","type":"address[]"},{"internalType":"uint256[]","name":"amounts","type":"uint256[]"}],"name":"batchSend","outputs":[],"stateMutability":"payable","type":"function"}]`

// 配置结构体
type Config struct {
	RPCURL          string
	ContractAddress string
	CSVFilePath     string
	AmountPerWallet *big.Int // 每个钱包转账金额（以 Wei 为单位）
	GasLimit        uint64   // 如果大于 0，则使用固定值
	GasPrice        *big.Int
	MaxWallets      int        // 最大处理钱包数量，0 表示不限制
	SenderWallet    WalletInfo // 新增：发送者钱包信息
}

// 钱包信息结构体
type WalletInfo struct {
	Address    string
	PrivateKey string
	Mnemonic   string
}

// 读取 CSV 文件
func readWalletsFromCSV(filePath string) ([]WalletInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 CSV 文件失败: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("读取 CSV 文件失败: %v", err)
	}

	if len(records) < 2 { // 至少需要表头和一行数据
		return nil, fmt.Errorf("CSV 文件为空或格式不正确")
	}

	// 验证表头
	headers := records[0]
	expectedHeaders := []string{"Address", "Private Key", "Mnemonic"}
	for i, header := range expectedHeaders {
		if headers[i] != header {
			return nil, fmt.Errorf("CSV 表头不正确，期望: %v, 实际: %v", expectedHeaders, headers)
		}
	}

	var wallets []WalletInfo
	for i, record := range records[1:] {
		if len(record) != 3 {
			return nil, fmt.Errorf("第 %d 行数据格式不正确", i+2)
		}
		wallets = append(wallets, WalletInfo{
			Address:    strings.TrimSpace(record[0]),
			PrivateKey: strings.TrimSpace(record[1]),
			Mnemonic:   strings.TrimSpace(record[2]),
		})
	}

	return wallets, nil
}

// 执行批量转账
func ExecuteBatchTransfer(cfg *Config) error {
	// 1. 读取接收者钱包信息
	wallets, err := readWalletsFromCSV(cfg.CSVFilePath)
	if err != nil {
		return fmt.Errorf("读取接收者钱包信息失败: %v", err)
	}

	totalWallets := len(wallets)
	if cfg.MaxWallets > 0 && totalWallets > cfg.MaxWallets {
		log.Printf("CSV 文件中包含 %d 个钱包，将只处理前 %d 个钱包", totalWallets, cfg.MaxWallets)
		wallets = wallets[:cfg.MaxWallets]
		totalWallets = cfg.MaxWallets
	}

	batchSize := 300
	totalBatches := (totalWallets + batchSize - 1) / batchSize

	log.Printf("总共处理 %d 个钱包地址，将分 %d 批处理，每批最多 %d 个地址", totalWallets, totalBatches, batchSize)

	// 2. 连接以太坊网络
	client, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return fmt.Errorf("连接以太坊网络失败: %v", err)
	}

	// 3. 解析 ABI
	parsedABI, err := abi.JSON(strings.NewReader(batchTransferABI))
	if err != nil {
		return fmt.Errorf("解析 ABI 失败: %v", err)
	}

	// 4. 创建合约实例
	contractAddress := common.HexToAddress(cfg.ContractAddress)
	contract := bind.NewBoundContract(contractAddress, parsedABI, client, client, client)

	// 5. 使用配置的发送者钱包创建交易选项
	auth, err := getTransactOpts(client, cfg.SenderWallet.PrivateKey, cfg.GasPrice, cfg.GasLimit)
	if err != nil {
		return fmt.Errorf("创建交易选项失败: %v", err)
	}

	// 6. 分批处理
	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		start := batchIndex * batchSize
		end := (batchIndex + 1) * batchSize
		if end > totalWallets {
			end = totalWallets
		}

		currentBatch := wallets[start:end]
		log.Printf("处理第 %d/%d 批，包含 %d 个地址", batchIndex+1, totalBatches, len(currentBatch))

		// 准备当前批次的转账数据
		var recipients []common.Address
		var amounts []*big.Int
		for _, wallet := range currentBatch {
			recipients = append(recipients, common.HexToAddress(wallet.Address))
			amounts = append(amounts, cfg.AmountPerWallet)
		}

		// 计算当前批次的总金额
		batchTotalAmount := new(big.Int).Mul(cfg.AmountPerWallet, big.NewInt(int64(len(currentBatch))))
		auth.Value = batchTotalAmount

		// 如果没有设置固定的 gas limit，则进行估算
		if cfg.GasLimit == 0 {
			// 准备调用数据
			data, err := parsedABI.Pack("batchSend", recipients, amounts)
			if err != nil {
				return fmt.Errorf("第 %d 批打包调用数据失败: %v", batchIndex+1, err)
			}

			// 估算 gas
			msg := ethereum.CallMsg{
				From:  auth.From,
				To:    &contractAddress,
				Value: batchTotalAmount,
				Data:  data,
			}
			gasLimit, err := client.EstimateGas(context.Background(), msg)
			if err != nil {
				return fmt.Errorf("第 %d 批估算 gas 限制失败: %v", batchIndex+1, err)
			}

			// 增加 20% 的 gas 限制作为缓冲
			gasLimit = gasLimit * 12 / 10
			auth.GasLimit = gasLimit

			log.Printf("第 %d 批估算 gas 限制: %d (包含 20%% 缓冲)", batchIndex+1, gasLimit)
		} else {
			log.Printf("第 %d 批使用固定 gas 限制: %d", batchIndex+1, cfg.GasLimit)
		}

		// 发送交易
		tx, err := contract.Transact(auth, "batchSend", recipients, amounts)
		if err != nil {
			return fmt.Errorf("第 %d 批发送交易失败: %v", batchIndex+1, err)
		}

		log.Printf("第 %d 批交易已发送，交易哈希: %s", batchIndex+1, tx.Hash().Hex())

		// 等待交易确认
		receipt, err := bind.WaitMined(context.Background(), client, tx)
		if err != nil {
			return fmt.Errorf("第 %d 批等待交易确认失败: %v", batchIndex+1, err)
		}

		if receipt.Status == 0 {
			return fmt.Errorf("第 %d 批交易执行失败，交易哈希: %s", batchIndex+1, receipt.TxHash.Hex())
		}

		log.Printf("第 %d 批转账成功！交易哈希: %s，实际使用 gas: %d",
			batchIndex+1,
			receipt.TxHash.Hex(),
			receipt.GasUsed,
		)

		// 如果不是最后一批，等待一段时间再处理下一批
		if batchIndex < totalBatches-1 {
			waitTime := 5 * time.Second
			log.Printf("等待 %v 后处理下一批...", waitTime)
			time.Sleep(waitTime)
		}
	}

	log.Printf("所有批次处理完成！总共处理 %d 个钱包地址", totalWallets)
	return nil
}

// 辅助函数：创建交易选项
func getTransactOpts(client *ethclient.Client, privateKeyHex string, gasPrice *big.Int, gasLimit uint64) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %v", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("获取链 ID 失败: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("创建交易签名者失败: %v", err)
	}

	auth.GasPrice = gasPrice
	auth.GasLimit = gasLimit
	auth.Context = context.Background()

	return auth, nil
}

var (
	rpcURL             string
	contractAddress    string
	csvFilePath        string
	senderCSVPath      string // 新增：发送者钱包 CSV 文件路径
	senderIndex        int    // 新增：发送者钱包在 CSV 中的索引
	amountPerWallet    float64
	gasPriceMultiplier float64
	batchSize          int
	fixedGasLimit      uint64
	maxWallets         int
)

// BatchTransferCmd 是批量转账命令
var BatchTransferCmd = &cobra.Command{
	Use:   "batch-transfer",
	Short: "执行批量转账操作",
	Long:  `从 CSV 文件中读取钱包地址，并执行批量转账操作。支持分批处理和动态 gas 价格。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 验证必需参数
		if csvFilePath == "" {
			log.Fatal("请提供接收者钱包 CSV 文件路径 (--csv)")
		}
		if senderCSVPath == "" {
			log.Fatal("请提供发送者钱包 CSV 文件路径 (--sender-csv)")
		}
		if senderIndex < 0 {
			log.Fatal("发送者钱包索引不能为负数 (--sender-index)")
		}
		if batchSize <= 0 {
			log.Fatal("批次大小必须大于 0 (--batch-size)")
		}
		if maxWallets < 0 {
			log.Fatal("最大钱包数量不能为负数 (--max-wallets)")
		}

		// 读取发送者钱包信息
		senderWallets, err := readWalletsFromCSV(senderCSVPath)
		if err != nil {
			log.Fatalf("读取发送者钱包 CSV 文件失败: %v", err)
		}
		if senderIndex >= len(senderWallets) {
			log.Fatalf("发送者钱包索引超出范围 (0-%d)", len(senderWallets)-1)
		}
		senderWallet := senderWallets[senderIndex]

		// 连接以太坊网络
		client, err := ethclient.Dial(rpcURL)
		if err != nil {
			log.Fatalf("连接以太坊网络失败: %v", err)
		}

		// 获取当前网络的平均 gas 价格
		suggestedGasPrice, err := client.SuggestGasPrice(context.Background())
		if err != nil {
			log.Fatalf("获取网络 gas 价格失败: %v", err)
		}

		// 应用倍率
		gasPriceWei := new(big.Int).Mul(
			suggestedGasPrice,
			big.NewInt(int64(gasPriceMultiplier*100)),
		)
		gasPriceWei = gasPriceWei.Div(gasPriceWei, big.NewInt(100))

		// 转换金额为 Wei
		amountWei := new(big.Int).Mul(
			big.NewInt(int64(amountPerWallet*1e18)),
			big.NewInt(1),
		)

		cfg := &Config{
			RPCURL:          rpcURL,
			ContractAddress: contractAddress,
			CSVFilePath:     csvFilePath,
			AmountPerWallet: amountWei,
			GasLimit:        fixedGasLimit,
			GasPrice:        gasPriceWei,
			MaxWallets:      maxWallets,
			SenderWallet:    senderWallet, // 新增：设置发送者钱包
		}

		log.Printf("配置信息:")
		log.Printf("- RPC URL: %s", cfg.RPCURL)
		log.Printf("- 合约地址: %s", cfg.ContractAddress)
		log.Printf("- 发送者钱包: %s (索引: %d)", cfg.SenderWallet.Address, senderIndex)
		log.Printf("- 接收者钱包 CSV: %s", cfg.CSVFilePath)
		log.Printf("- 每个钱包转账金额: %.4f ETH", float64(cfg.AmountPerWallet.Int64())/1e18)
		log.Printf("- 网络建议 Gas 价格: %.1f Gwei", float64(suggestedGasPrice.Int64())/1e9)
		log.Printf("- 实际使用 Gas 价格: %.1f Gwei (%.1f 倍)", float64(cfg.GasPrice.Int64())/1e9, gasPriceMultiplier)
		if cfg.GasLimit > 0 {
			log.Printf("- 使用固定 Gas 限制: %d", cfg.GasLimit)
		} else {
			log.Printf("- Gas 限制: 动态估算")
		}
		log.Printf("- 每批处理钱包数量: %d", batchSize)
		if cfg.MaxWallets > 0 {
			log.Printf("- 最大处理钱包数量: %d", cfg.MaxWallets)
		} else {
			log.Printf("- 最大处理钱包数量: 不限制")
		}

		if err := ExecuteBatchTransfer(cfg); err != nil {
			log.Fatalf("批量转账失败: %v", err)
		}
	},
}

func init() {
	BatchTransferCmd.Flags().StringVar(&rpcURL, "rpc", "https://bsc-dataseed.binance.org/", "以太坊 RPC URL")
	BatchTransferCmd.Flags().StringVar(&contractAddress, "contract", "0x61e0336Ba3bEd95deD28b01ef9cD015d7F32437d", "批量转账合约地址")
	BatchTransferCmd.Flags().StringVar(&csvFilePath, "csv", "", "接收者钱包 CSV 文件路径")
	BatchTransferCmd.Flags().StringVar(&senderCSVPath, "sender-csv", "wallets/senders/w1.csv", "发送者钱包 CSV 文件路径")
	BatchTransferCmd.Flags().IntVar(&senderIndex, "sender-index", 0, "发送者钱包在 CSV 中的索引")
	BatchTransferCmd.Flags().Float64Var(&amountPerWallet, "amount", 0.1, "每个钱包转账金额 (ETH)")
	BatchTransferCmd.Flags().Float64Var(&gasPriceMultiplier, "gas-multiplier", 1.0001, "Gas 价格倍率 (相对于网络平均 gas 价格)")
	BatchTransferCmd.Flags().IntVar(&batchSize, "batch-size", 300, "每批处理的钱包数量")
	BatchTransferCmd.Flags().Uint64Var(&fixedGasLimit, "gas-limit", 0, "固定的 Gas 限制 (如果设置，将跳过估算)")
	BatchTransferCmd.Flags().IntVar(&maxWallets, "max-wallets", 0, "最大处理钱包数量 (0 表示不限制)")

	// 只标记 csv 参数为必需
	BatchTransferCmd.MarkFlagRequired("csv")
}

// RunBatchTransfer 是批量转账命令的入口点
func RunBatchTransfer() {
	if err := BatchTransferCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
