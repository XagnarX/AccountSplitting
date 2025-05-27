package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var verifyFile string

var verifyCmd = &cobra.Command{
	Use:   "verifycsv",
	Short: "校验CSV文件中的以太坊地址和私钥是否匹配",
	Run: func(cmd *cobra.Command, args []string) {
		if verifyFile == "" {
			fmt.Println("请使用 --file 或 -f 指定要校验的CSV文件路径")
			return
		}
		verifyCSV(verifyFile)
	},
}

func init() {
	verifyCmd.Flags().StringVarP(&verifyFile, "file", "f", "", "要校验的CSV文件路径")
	rootCmd.AddCommand(verifyCmd)
}

func verifyCSV(filePath string) {
	fmt.Println("开始验证地址和私钥匹配...")
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("错误: 找不到文件 %s\n", filePath)
		return
	}
	defer file.Close()
	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		fmt.Println("读取CSV标题失败:", err)
		return
	}
	fmt.Printf("CSV 标题: %v\n", header)
	if len(header) < 2 {
		fmt.Println("错误: CSV 文件格式不正确，至少需要地址和私钥两列")
		return
	}
	total := 0
	matched := 0
	mismatched := make([][2]string, 0)
	for rowNumber := 2; ; rowNumber++ {
		row, err := reader.Read()
		if err != nil {
			break
		}
		if len(row) < 2 {
			fmt.Printf("行 %d: 格式错误 - 列数不足\n", rowNumber)
			continue
		}
		address := strings.TrimSpace(row[0])
		privateKey := strings.TrimSpace(row[1])
		if address == "" {
			fmt.Printf("行 %d: ❌ 地址为空\n", rowNumber)
			mismatched = append(mismatched, [2]string{fmt.Sprint(rowNumber), "地址为空"})
			continue
		}
		result, msg := checkAddressPrivateKey(address, privateKey)
		total++
		if result {
			matched++
			fmt.Printf("行 %d: ✅ 匹配成功 - %s\n", rowNumber, address)
		} else {
			mismatched = append(mismatched, [2]string{fmt.Sprint(rowNumber), address + " (" + msg + ")"})
			fmt.Printf("行 %d: ❌ 匹配失败 - %s (%s)\n", rowNumber, address, msg)
		}
	}
	fmt.Println("\n验证结果总结:")
	fmt.Printf("总计: %d 个地址\n", total)
	fmt.Printf("匹配成功: %d 个\n", matched)
	fmt.Printf("匹配失败: %d 个\n", len(mismatched))
	if len(mismatched) > 0 {
		fmt.Println("\n不匹配的地址列表:")
		for _, item := range mismatched {
			fmt.Printf("行 %s: %s\n", item[0], item[1])
		}
	}
}

func checkAddressPrivateKey(address, privateKey string) (bool, string) {
	if privateKey == "" {
		return false, "私钥为空"
	}
	cleanKey := strings.TrimPrefix(privateKey, "0x")
	if len(cleanKey) != 64 {
		return false, "私钥长度不正确"
	}
	if !strings.HasPrefix(privateKey, "0x") {
		privateKey = "0x" + cleanKey
	}
	pk, err := crypto.HexToECDSA(cleanKey)
	if err != nil {
		return false, "私钥格式错误"
	}
	pubKey := pk.PublicKey
	derivedAddress := crypto.PubkeyToAddress(pubKey).Hex()
	if !common.IsHexAddress(address) {
		return false, "地址格式不正确"
	}
	inputAddress := common.HexToAddress(address).Hex()
	if strings.ToLower(inputAddress) == strings.ToLower(derivedAddress) {
		return true, "地址匹配"
	}
	return false, "地址与私钥不匹配"
}
