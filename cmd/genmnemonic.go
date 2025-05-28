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

// GenMnemonicCmd 是生成助记词和钱包的命令
var GenMnemonicCmd = &cobra.Command{
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
	GenMnemonicCmd.Flags().IntVarP(&numMws, "number", "n", 10, "生成钱包数量")
	GenMnemonicCmd.Flags().StringVarP(&outCsv, "output", "o", "mnemonic.csv", "输出文件名")
	GenMnemonicCmd.Flags().StringVarP(&mnemonicDir, "dir", "d", "./wallets", "输出目录")
}
