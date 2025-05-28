package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

// NodeResult 存储节点检查结果
type NodeResult struct {
	URL          string
	ResponseTime time.Duration
	BlockHeight  *big.Int
	Error        error
}

var (
	rpcTimeout   int
	showStats    bool
	outputFormat string
)

// CheckRPCCmd 是检查 RPC 节点的命令
var CheckRPCCmd = &cobra.Command{
	Use:   "check-rpc",
	Short: "检查 BSC RPC 节点的可用性和响应时间",
	Long:  `检查多个 BSC RPC 节点的可用性、响应时间和区块高度。`,
	Run: func(cmd *cobra.Command, args []string) {
		// BSC 节点列表
		nodes := []string{
			"https://bsc-dataseed.binance.org/",
			"https://bsc-dataseed1.defibit.io/",
			"https://bsc-dataseed1.ninicoin.io/",
			"https://bsc-dataseed2.defibit.io/",
			"https://bsc-dataseed3.defibit.io/",
			"https://bsc-dataseed4.defibit.io/",
			"https://bsc-dataseed2.ninicoin.io/",
			"https://bsc-dataseed3.ninicoin.io/",
			"https://bsc-dataseed4.ninicoin.io/",
			"https://bsc-dataseed1.binance.org/",
			"https://bsc-dataseed2.binance.org/",
			"https://bsc-dataseed3.binance.org/",
			"https://bsc-dataseed4.binance.org/",
		}

		// 创建结果通道
		results := make(chan NodeResult, len(nodes))
		var wg sync.WaitGroup

		// 为每个节点启动检查协程
		for _, node := range nodes {
			wg.Add(1)
			go func(nodeURL string) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rpcTimeout)*time.Second)
				defer cancel()
				checkNode(ctx, nodeURL, results)
			}(node)
		}

		// 等待所有检查完成
		go func() {
			wg.Wait()
			close(results)
		}()

		// 收集结果
		var nodeResults []NodeResult
		for result := range results {
			if result.Error == nil {
				nodeResults = append(nodeResults, result)
			}
		}

		// 按响应时间排序
		sort.Slice(nodeResults, func(i, j int) bool {
			return nodeResults[i].ResponseTime < nodeResults[j].ResponseTime
		})

		// 输出结果
		switch outputFormat {
		case "json":
			outputJSON(nodeResults)
		case "csv":
			outputCSV(nodeResults)
		default:
			outputText(nodeResults, showStats)
		}
	},
}

func init() {
	CheckRPCCmd.Flags().IntVar(&rpcTimeout, "timeout", 5, "RPC 请求超时时间（秒）")
	CheckRPCCmd.Flags().BoolVar(&showStats, "stats", false, "显示统计信息")
	CheckRPCCmd.Flags().StringVar(&outputFormat, "format", "text", "输出格式 (text, json, csv)")
}

// checkNode 检查单个节点的状态
func checkNode(ctx context.Context, nodeURL string, results chan<- NodeResult) {
	start := time.Now()
	client, err := ethclient.DialContext(ctx, nodeURL)
	if err != nil {
		results <- NodeResult{URL: nodeURL, Error: err}
		return
	}

	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		results <- NodeResult{URL: nodeURL, Error: err}
		return
	}

	responseTime := time.Since(start)
	results <- NodeResult{
		URL:          nodeURL,
		ResponseTime: responseTime,
		BlockHeight:  big.NewInt(int64(blockNumber)),
		Error:        nil,
	}
}

// outputJSON 以 JSON 格式输出结果
func outputJSON(results []NodeResult) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("JSON 编码失败: %v", err)
	}
	fmt.Println(string(data))
}

// outputCSV 以 CSV 格式输出结果
func outputCSV(results []NodeResult) {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// 写入表头
	writer.Write([]string{"URL", "响应时间(ms)", "区块高度"})

	// 写入数据
	for _, result := range results {
		writer.Write([]string{
			result.URL,
			fmt.Sprintf("%.2f", float64(result.ResponseTime.Microseconds())/1000),
			result.BlockHeight.String(),
		})
	}
}

// outputText 以文本格式输出结果
func outputText(results []NodeResult, showStats bool) {
	fmt.Printf("\nBSC 节点检查结果 (共 %d 个节点):\n\n", len(results))

	for i, result := range results {
		fmt.Printf("%d. %s\n", i+1, result.URL)
		fmt.Printf("   响应时间: %.2f ms\n", float64(result.ResponseTime.Microseconds())/1000)
		fmt.Printf("   区块高度: %s\n", result.BlockHeight.String())
		fmt.Println()
	}

	if showStats {
		var totalTime time.Duration
		for _, result := range results {
			totalTime += result.ResponseTime
		}
		avgTime := totalTime / time.Duration(len(results))

		fmt.Printf("统计信息:\n")
		fmt.Printf("- 平均响应时间: %.2f ms\n", float64(avgTime.Microseconds())/1000)
		fmt.Printf("- 最快节点: %s (%.2f ms)\n", results[0].URL, float64(results[0].ResponseTime.Microseconds())/1000)
		fmt.Printf("- 最慢节点: %s (%.2f ms)\n", results[len(results)-1].URL, float64(results[len(results)-1].ResponseTime.Microseconds())/1000)
	}

	fmt.Println("\n推荐使用的节点:")
	for i, result := range results[:3] {
		fmt.Printf("%d. %s (%.2f ms)\n", i+1, result.URL, float64(result.ResponseTime.Microseconds())/1000)
	}
}
