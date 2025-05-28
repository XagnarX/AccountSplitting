package cmd

import (
	"context"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

var (
	singleTransferRPCURL        string
	singleTransferCSVPath       string
	singleTransferTargetAddr    string
	singleTransferAmount        float64
	singleTransferGasMultiplier float64
	singleTransferGasLimit      uint64
	singleTransferMaxWallets    int
	singleTransferDelay         int // 每次转账之间的延迟（秒）
)

// SingleTransferCmd 是单地址转账命令
var SingleTransferCmd = &cobra.Command{
	Use:   "single-transfer",
	Short: "从 CSV 文件中读取钱包，逐个向指定地址转入固定数量的 BNB",
	Long:  `从 CSV 文件中读取钱包信息，逐个向指定地址转入固定数量的 BNB。支持设置 gas 价格倍率和转账延迟。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 验证参数
		if singleTransferCSVPath == "" {
			log.Fatal("请提供钱包 CSV 文件路径 (--csv)")
		}
		if singleTransferTargetAddr == "" {
			log.Fatal("请提供目标地址 (--target)")
		}
		if singleTransferAmount <= 0 {
			log.Fatal("转账金额必须大于 0 (--amount)")
		}
		if singleTransferMaxWallets < 0 {
			log.Fatal("最大钱包数量不能为负数 (--max-wallets)")
		}
		if singleTransferDelay < 0 {
			log.Fatal("转账延迟不能为负数 (--delay)")
		}

		// 连接以太坊网络
		client, err := ethclient.Dial(singleTransferRPCURL)
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
			big.NewInt(int64(singleTransferGasMultiplier*10000)),
		)
		gasPriceWei = gasPriceWei.Div(gasPriceWei, big.NewInt(10000))

		// 转换金额为 Wei
		amountWei := new(big.Int).Mul(
			big.NewInt(int64(singleTransferAmount*1e18)),
			big.NewInt(1),
		)

		// 读取钱包信息
		wallets, err := readWalletsFromCSV(singleTransferCSVPath)
		if err != nil {
			log.Fatalf("读取钱包 CSV 文件失败: %v", err)
		}

		totalWallets := len(wallets)
		if singleTransferMaxWallets > 0 && totalWallets > singleTransferMaxWallets {
			log.Printf("CSV 文件中包含 %d 个钱包，将只处理前 %d 个钱包", totalWallets, singleTransferMaxWallets)
			wallets = wallets[:singleTransferMaxWallets]
			totalWallets = singleTransferMaxWallets
		}

		// 验证目标地址
		targetAddress := common.HexToAddress(singleTransferTargetAddr)
		if !common.IsHexAddress(singleTransferTargetAddr) {
			log.Fatalf("无效的目标地址: %s", singleTransferTargetAddr)
		}

		log.Printf("配置信息:")
		log.Printf("- RPC URL: %s", singleTransferRPCURL)
		log.Printf("- 目标地址: %s", targetAddress.Hex())
		log.Printf("- 每个钱包转账金额: %.4f BNB", singleTransferAmount)
		log.Printf("- 网络建议 Gas 价格: %.1f Gwei", float64(suggestedGasPrice.Int64())/1e9)
		log.Printf("- 实际使用 Gas 价格: %.1f Gwei (%.4f 倍)", float64(gasPriceWei.Int64())/1e9, singleTransferGasMultiplier)
		if singleTransferGasLimit > 0 {
			log.Printf("- 使用固定 Gas 限制: %d", singleTransferGasLimit)
		} else {
			log.Printf("- Gas 限制: 动态估算")
		}
		log.Printf("- 转账延迟: %d 秒", singleTransferDelay)
		log.Printf("- 总钱包数量: %d", totalWallets)

		// 逐个处理钱包
		successCount := 0
		failCount := 0
		for i, wallet := range wallets {
			log.Printf("\n处理第 %d/%d 个钱包: %s", i+1, totalWallets, wallet.Address)

			// 解析私钥
			privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(wallet.PrivateKey, "0x"))
			if err != nil {
				log.Printf("解析私钥失败: %v", err)
				failCount++
				continue
			}

			// 获取发送者地址
			fromAddress := crypto.PubkeyToAddress(privateKey.PublicKey)

			// 获取 nonce
			nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
			if err != nil {
				log.Printf("获取 nonce 失败: %v", err)
				failCount++
				continue
			}

			// 估算 gas
			gasLimit := singleTransferGasLimit
			if gasLimit == 0 {
				msg := ethereum.CallMsg{
					From:  fromAddress,
					To:    &targetAddress,
					Value: amountWei,
				}
				estimatedGas, err := client.EstimateGas(context.Background(), msg)
				if err != nil {
					log.Printf("估算 gas 失败: %v", err)
					failCount++
					continue
				}
				gasLimit = estimatedGas * 12 / 10 // 增加 20% 的缓冲
			}

			// 创建交易
			tx := types.NewTransaction(
				nonce,
				targetAddress,
				amountWei,
				gasLimit,
				gasPriceWei,
				nil,
			)

			// 签名交易
			chainID, err := client.ChainID(context.Background())
			if err != nil {
				log.Printf("获取链 ID 失败: %v", err)
				failCount++
				continue
			}

			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
			if err != nil {
				log.Printf("签名交易失败: %v", err)
				failCount++
				continue
			}

			// 发送交易
			err = client.SendTransaction(context.Background(), signedTx)
			if err != nil {
				log.Printf("发送交易失败: %v", err)
				failCount++
				continue
			}

			log.Printf("交易已发送，交易哈希: %s", signedTx.Hash().Hex())

			// 等待交易确认
			receipt, err := bind.WaitMined(context.Background(), client, signedTx)
			if err != nil {
				log.Printf("等待交易确认失败: %v", err)
				failCount++
				continue
			}

			if receipt.Status == 0 {
				log.Printf("交易执行失败，交易哈希: %s", receipt.TxHash.Hex())
				failCount++
				continue
			}

			log.Printf("转账成功！交易哈希: %s，实际使用 gas: %d",
				receipt.TxHash.Hex(),
				receipt.GasUsed,
			)
			successCount++

			// 如果不是最后一个钱包，等待指定的延迟时间
			if i < totalWallets-1 && singleTransferDelay > 0 {
				log.Printf("等待 %d 秒后处理下一个钱包...", singleTransferDelay)
				time.Sleep(time.Duration(singleTransferDelay) * time.Second)
			}
		}

		log.Printf("\n转账完成！成功: %d，失败: %d", successCount, failCount)
	},
}

func init() {
	SingleTransferCmd.Flags().StringVar(&singleTransferRPCURL, "rpc", "https://bsc-dataseed.binance.org/", "以太坊 RPC URL")
	SingleTransferCmd.Flags().StringVar(&singleTransferCSVPath, "csv", "", "钱包 CSV 文件路径")
	SingleTransferCmd.Flags().StringVar(&singleTransferTargetAddr, "target", "0x774d0d4281217deDB7ae7797D69968D6Ea07c1Ae", "目标地址")
	SingleTransferCmd.Flags().Float64Var(&singleTransferAmount, "amount", 0.0001, "每个钱包转账金额 (BNB)")
	SingleTransferCmd.Flags().Float64Var(&singleTransferGasMultiplier, "gas-multiplier", 1.0001, "Gas 价格倍率 (相对于网络平均 gas 价格)")
	SingleTransferCmd.Flags().Uint64Var(&singleTransferGasLimit, "gas-limit", 0, "固定的 Gas 限制 (如果设置，将跳过估算)")
	SingleTransferCmd.Flags().IntVar(&singleTransferMaxWallets, "max-wallets", 0, "最大处理钱包数量 (0 表示不限制)")
	SingleTransferCmd.Flags().IntVar(&singleTransferDelay, "delay", 30, "每次转账之间的延迟（秒）")

	// 设置必需参数
	SingleTransferCmd.MarkFlagRequired("csv")
	// SingleTransferCmd.MarkFlagRequired("target")
}
