package cmd

import (
	"AccountSplitting/lib"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	numMws      int
	outCsv      string
	mnemonicDir string
)

var genMnemonicCmd = &cobra.Command{
	Use:   "genmnemonic",
	Short: "批量生成带助记词的钱包",
	Run: func(cmd *cobra.Command, args []string) {
		if mnemonicDir == "" {
			mnemonicDir = "./wallets"
		}
		if err := os.MkdirAll(mnemonicDir, 0755); err != nil {
			fmt.Println("创建目录失败:", err)
			return
		}
		outputPath := filepath.Join(mnemonicDir, outCsv)
		err := lib.GmwsAndWirte(numMws, outputPath)
		if err != nil {
			fmt.Println("生成失败:", err)
		} else {
			fmt.Println("生成成功，写入文件：", outputPath)
		}
	},
}

func init() {
	genMnemonicCmd.Flags().IntVarP(&numMws, "number", "n", 10, "生成钱包数量")
	genMnemonicCmd.Flags().StringVarP(&outCsv, "output", "o", "mnemonic.csv", "输出文件名")
	genMnemonicCmd.Flags().StringVarP(&mnemonicDir, "dir", "d", "./wallets", "输出目录")
	rootCmd.AddCommand(genMnemonicCmd)
}
