package cmd

import (
	"AccountSplitting/lib"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	numWallets int
	outputFile string
	walletDir  string
)

// GenWalletCmd 是生成钱包的命令
var GenWalletCmd = &cobra.Command{
	Use:   "genwallet",
	Short: "批量生成钱包",
	Run: func(cmd *cobra.Command, args []string) {
		if walletDir == "" {
			walletDir = "./wallets"
		}
		if err := os.MkdirAll(walletDir, 0755); err != nil {
			fmt.Println("创建目录失败:", err)
			return
		}
		outputPath := filepath.Join(walletDir, outputFile)
		err := lib.GWalletsAndWirte(numWallets, outputPath)
		if err != nil {
			fmt.Println("生成失败:", err)
		} else {
			fmt.Println("生成成功，写入文件：", outputPath)
		}
	},
}

func init() {
	GenWalletCmd.Flags().IntVarP(&numWallets, "number", "n", 10, "生成钱包数量")
	GenWalletCmd.Flags().StringVarP(&outputFile, "output", "o", "wallets.csv", "输出文件名")
	GenWalletCmd.Flags().StringVarP(&walletDir, "dir", "d", "./wallets", "输出目录")
}
