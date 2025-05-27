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
	outFile    string
	dir        string
)

var genWalletCmd = &cobra.Command{
	Use:   "genwallet",
	Short: "批量生成钱包（私钥+地址）",
	Run: func(cmd *cobra.Command, args []string) {
		if dir == "" {
			dir = "./wallets"
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Println("创建目录失败:", err)
			return
		}
		outputPath := filepath.Join(dir, outFile)
		err := lib.GWalletsAndWirte(numWallets, outputPath)
		if err != nil {
			fmt.Println("生成失败:", err)
		} else {
			fmt.Println("生成成功，写入文件：", outputPath)
		}
	},
}

func init() {
	genWalletCmd.Flags().IntVarP(&numWallets, "number", "n", 10, "生成钱包数量")
	genWalletCmd.Flags().StringVarP(&outFile, "output", "o", "secret.csv", "输出文件名")
	genWalletCmd.Flags().StringVarP(&dir, "dir", "d", "./wallets", "输出目录")
	rootCmd.AddCommand(genWalletCmd)
}
