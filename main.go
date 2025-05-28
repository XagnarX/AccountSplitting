package main

import (
	"fmt"
	"os"

	"AccountSplitting/cmd"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "account-splitting",
	Short: "账户拆分工具",
	Long:  `一个用于批量转账和检查 RPC 节点的命令行工具。`,
}

func init() {
	rootCmd.AddCommand(cmd.BatchTransferCmd)
	rootCmd.AddCommand(cmd.CheckRPCCmd)
	rootCmd.AddCommand(cmd.GenMnemonicCmd)
	rootCmd.AddCommand(cmd.GenWalletCmd)
	rootCmd.AddCommand(cmd.SingleTransferCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
