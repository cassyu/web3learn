// Sepolia 链上读写示例（go-ethereum ethclient）
//
// 整体流程：
//   1) ethclient.Dial(RPC) 连接 Infura 等提供的 HTTPS 节点；
//   2) query：读区块头 + 完整区块，打印哈希/时间/交易数；
//   3) send：用本地私钥签名一笔 Legacy 转账，广播到 Sepolia（链 ID 11155111）。
//
// 环境变量：
//   SEPOLIA_RPC_URL  必填，例如 https://sepolia.infura.io/v3/<你的KEY>
//   SEPOLIA_PRIVATE_KEY  仅在 mode=send 时需要，0x 开头或不带前缀的 hex
//
// 用法：
//   查询最新区块：go run homework05.go -mode=query
//   查询指定高度：go run homework05.go -mode=query -block=12345
//   发送测试转账（PowerShell 里 -eth 的小数点会被拆开，请用下面任一方式）：
//     go run homework05.go -mode=send -to=0x接收地址 -eth='0.001'
//     或（推荐，无小数点）：go run homework05.go -mode=send -to=0x接收地址 -wei=1000000000000000
//   0.001 ETH = 1000000000000000 wei
//
package main

import (
	"context"
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Sepolia 测试网链 ID，用于 EIP-155 签名（与主网不同，必须匹配否则交易会被拒绝）。
var sepoliaChainID = big.NewInt(11155111)

// loadPrivateKeyFromEnv 从环境变量读取私钥并转为 *ecdsa.PrivateKey。
// 以太坊私钥为 32 字节 = 64 个十六进制字符（可带 0x）；不要用地址（40 位）或助记词。
func loadPrivateKeyFromEnv() *ecdsa.PrivateKey {
	raw := strings.TrimSpace(os.Getenv("SEPOLIA_PRIVATE_KEY"))
	if raw == "" {
		log.Fatal("send 模式需要环境变量 SEPOLIA_PRIVATE_KEY（不要提交到 Git）")
	}
	// 去掉 PowerShell 里误加的双引号
	raw = strings.Trim(raw, `"'`)
	pkHex := strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X")
	pkHex = strings.TrimSpace(pkHex)
	for _, r := range pkHex {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			log.Fatalf("私钥只能包含 0-9、a-f（不要粘贴助记词或 JSON）")
		}
	}
	if len(pkHex) != 64 {
		log.Fatalf(
			"私钥长度错误：需要恰好 64 个十六进制字符（256 bit），当前 %d 个。\n"+
				"请从钱包导出「私钥」一行 hex，不要填助记词；不要多空格/换行。示例长度：64（或带 0x 共 66）。",
			len(pkHex),
		)
	}
	key, err := crypto.HexToECDSA(pkHex)
	if err != nil {
		log.Fatalf("解析私钥失败：%v", err)
	}
	return key
}

func main() {
	// 命令行参数：RPC 优先读 -rpc，否则读环境变量 SEPOLIA_RPC_URL
	rpc := flag.String("rpc", os.Getenv("SEPOLIA_RPC_URL"), "Sepolia HTTPS RPC（也可用环境变量 SEPOLIA_RPC_URL）")
	mode := flag.String("mode", "query", "query：查区块 | send：发 ETH 转账")
	blockStr := flag.String("block", "", "区块高度（十进制），留空表示最新块")
	toAddr := flag.String("to", "", "send 模式：接收方地址")
	ethStr := flag.String("eth", "0", "send 模式：转账金额（ETH）；PowerShell 请写成 -eth='0.001'")
	weiStr := flag.String("wei", "", "send 模式：转账金额（wei，十进制整数），与 -eth 二选一；可避免 PowerShell 拆小数点")
	flag.Parse()

	if strings.TrimSpace(*rpc) == "" {
		log.Fatal("请设置 -rpc 或环境变量 SEPOLIA_RPC_URL（Infura Sepolia HTTPS 地址）")
	}

	// ethclient 封装了 JSON-RPC 调用（eth_blockNumber、eth_getBlockByNumber 等）
	client, err := ethclient.Dial(*rpc)
	if err != nil {
		log.Fatalf("连接节点失败: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	switch *mode {
	case "query":
		runQuery(ctx, client, *blockStr)
	case "send":
		runSend(ctx, client, *toAddr, *ethStr, *weiStr)
	default:
		log.Fatal("-mode 只能是 query 或 send")
	}
}

// runQuery 查询指定高度或最新块：打印区块哈希、时间戳、交易笔数。
func runQuery(ctx context.Context, client *ethclient.Client, blockStr string) {
	var num *big.Int
	if strings.TrimSpace(blockStr) == "" {
		num = nil // 最新块
	} else {
		var ok bool
		num, ok = new(big.Int).SetString(blockStr, 10)
		if !ok || num.Sign() < 0 {
			log.Fatal("-block 必须是十进制非负整数")
		}
	}

	// HeaderByNumber：轻量信息（nil 表示 latest）
	header, err := client.HeaderByNumber(ctx, num)
	if err != nil {
		log.Fatalf("读取区块头失败: %v", err)
	}

	// BlockByNumber：含完整交易列表，用于统计笔数（也可只用 TransactionCount 按哈希查）
	body, err := client.BlockByNumber(ctx, header.Number)
	if err != nil {
		log.Fatalf("读取完整区块失败: %v", err)
	}

	ts := time.Unix(int64(header.Time), 0).UTC()
	txCount := len(body.Transactions())

	fmt.Println("=== Sepolia 区块查询 ===")
	if num == nil {
		fmt.Println("高度: 最新")
	} else {
		fmt.Println("高度:", header.Number.String())
	}
	fmt.Println("区块哈希:", header.Hash().Hex())
	fmt.Println("时间戳(UTC):", ts.Format(time.RFC3339))
	fmt.Println("交易笔数:", txCount)
}

// runSend 构造一笔「纯 ETH 转账」：Legacy 交易、21000 gas、当前建议 gasPrice、EIP-155 签名后广播。
func runSend(ctx context.Context, client *ethclient.Client, toHex, ethAmount, weiAmount string) {
	key := loadPrivateKeyFromEnv()

	toHex = strings.TrimSpace(toHex)
	if !common.IsHexAddress(toHex) {
		log.Fatal("-to 必须是合法以太坊地址（0x...）")
	}

	var amount *big.Int
	var err error
	if strings.TrimSpace(weiAmount) != "" {
		amount, ok := new(big.Int).SetString(strings.TrimSpace(weiAmount), 10)
		if !ok || amount.Sign() <= 0 {
			log.Fatal("-wei 必须是正整数字符串（单位 wei）")
		}
	} else {
		amount, err = etherToWei(ethAmount)
		if err != nil {
			log.Fatalf("解析金额失败: %v", err)
		}
		if amount.Sign() <= 0 {
			log.Fatal("金额必须大于 0：PowerShell 请使用 -eth='0.001' 或 -wei=1000000000000000（0.001 ETH）")
		}
	}

	// 私钥 -> 发送方地址（必须与水龙头充值、余额所在地址一致）
	from := crypto.PubkeyToAddress(key.PublicKey)

	to := common.HexToAddress(toHex)

	// 账户当前待发交易序号，每笔交易递增 1
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		log.Fatalf("获取 nonce 失败: %v", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatalf("获取 gas price 失败: %v", err)
	}

	// LegacyTx：类型 0 交易；纯转账 data 为空；21000 为标准 ETH 转账 gas
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		GasPrice: gasPrice,
		Gas:      21000,
		To:       &to,
		Value:    amount,
		Data:     nil,
	})

	// EIP155Signer：把 chainId 纳入签名，防止跨链重放
	signed, err := types.SignTx(tx, types.NewEIP155Signer(sepoliaChainID), key)
	if err != nil {
		log.Fatalf("签名失败: %v", err)
	}

	// eth_sendRawTransaction：把已签名交易发到节点，进入 mempool
	if err := client.SendTransaction(ctx, signed); err != nil {
		log.Fatalf("发送交易失败: %v", err)
	}

	fmt.Println("=== Sepolia 转账已广播 ===")
	fmt.Println("交易哈希:", signed.Hash().Hex())
	fmt.Println("可在浏览器查看: https://sepolia.etherscan.io/tx/" + signed.Hash().Hex())
}

// etherToWei 把十进制 ETH 字符串（如 "0.001"）转为 wei（1 ETH = 1e18 wei）。
// 实现方式：把小数部分补到 18 位后与整数部分拼接，再转成 big.Int。
func etherToWei(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty amount")
	}

	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	if intPart == "" {
		intPart = "0"
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > 18 {
		return nil, fmt.Errorf("小数位过多（最多 18 位）")
	}
	// 小数部分右侧补零，使「小数位」总长度为 18，对应 wei 精度
	for len(frac) < 18 {
		frac += "0"
	}
	combined := intPart + frac
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		return big.NewInt(0), nil
	}
	w, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("invalid number")
	}
	return w, nil
}
