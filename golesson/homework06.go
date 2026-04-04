// 任务2：Solidity 合约 + abigen 生成绑定 + Sepolia 交互示例。
//
// 整体流程：Dial(RPC) → 解析私钥 → 分支「部署新合约」或「绑定已有地址」→
// 读 count（eth_call）→ 发 increment（签名交易）→ WaitMined → 再读 count。
//
// 注意：golesson 目录下多个 main，请只执行：go run homework06.go [参数]，不要用 go build .
//
// 重新生成绑定（在 golesson 目录下）：
//
//	npx --yes solc@0.8.20 --optimize --bin --abi -o contracts/out contracts/Counter.sol
//	abigen --abi contracts/out/contracts_Counter_sol_Counter.abi --bin contracts/out/contracts_Counter_sol_Counter.bin --pkg counter --type Counter -out counter/counter.go
//
// 环境变量：
//
//	SEPOLIA_RPC_URL      必填，Sepolia HTTPS RPC
//	SEPOLIA_PRIVATE_KEY  必填（部署或写操作），64 位 hex 私钥
//
// 用法：
//
//	部署新合约并调用 increment，再打印 count：
//	  go run homework06.go -deploy
//	与已部署合约交互：
//	  go run homework06.go -addr=0x你的合约地址
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

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golesson/counter" // abigen 生成的包：DeployCounter / NewCounter / Count / Increment
)

// Sepolia 链 ID，用于 NewKeyedTransactorWithChainID（EIP-155，防跨链重放）。
var sepoliaChainID = big.NewInt(11155111)

func main() {
	rpc := flag.String("rpc", os.Getenv("SEPOLIA_RPC_URL"), "Sepolia RPC，或环境变量 SEPOLIA_RPC_URL")
	deploy := flag.Bool("deploy", false, "部署 Counter 合约到 Sepolia")
	addrStr := flag.String("addr", os.Getenv("COUNTER_ADDRESS"), "已部署合约地址，或环境变量 COUNTER_ADDRESS")
	flag.Parse()

	if strings.TrimSpace(*rpc) == "" {
		log.Fatal("请设置 -rpc 或 SEPOLIA_RPC_URL")
	}

	// ethclient 实现 bind.ContractBackend：可发交易、调合约、等回执。
	client, err := ethclient.Dial(*rpc)
	if err != nil {
		log.Fatalf("连接节点失败: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := mustPrivateKey() // 与 homework05 一致：64 hex，勿用地址/助记词

	if *deploy {
		runDeployAndCall(ctx, client, key)
		return
	}

	if strings.TrimSpace(*addrStr) == "" {
		log.Fatal("请使用 -deploy 部署合约，或用 -addr / COUNTER_ADDRESS 指定已部署合约地址")
	}
	if !common.IsHexAddress(*addrStr) {
		log.Fatal("-addr 不是合法地址")
	}
	runInteract(ctx, client, key, common.HexToAddress(*addrStr))
}

// runDeployAndCall：发送合约创建交易（携带 bytecode），确认后再演示 increment。
func runDeployAndCall(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey) {
	auth := mustTransactor(ctx, client, key)

	fmt.Println("正在部署 Counter 合约…")
	// DeployCounter 由 abigen 生成：内部 eth_sendRawTransaction 部署链上合约。
	contractAddr, tx, ctr, err := counter.DeployCounter(auth, client)
	if err != nil {
		log.Fatalf("部署失败: %v", err)
	}
	fmt.Println("部署交易已发送:", tx.Hash().Hex())
	// 阻塞直到交易被打进区块；Status==1 表示 EVM 执行成功。
	rec, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Fatalf("等待部署确认失败: %v", err)
	}
	if rec.Status != 1 {
		log.Fatalf("部署交易执行失败 status=%d", rec.Status)
	}
	fmt.Println("合约地址:", contractAddr.Hex())
	fmt.Println("浏览器: https://sepolia.etherscan.io/address/" + contractAddr.Hex())

	callAndIncrement(ctx, client, key, ctr)
}

// runInteract：不部署，仅按已知合约地址构造绑定并调用。
func runInteract(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, addr common.Address) {
	// NewCounter：已部署合约用 ABI + 地址封装为 Go 对象。
	ctr, err := counter.NewCounter(addr, client)
	if err != nil {
		log.Fatalf("绑定合约失败: %v", err)
	}
	callAndIncrement(ctx, client, key, ctr)
}

// callAndIncrement：演示只读 Count → 写 Increment → 再读 Count（验证状态 +1）。
func callAndIncrement(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, ctr *counter.Counter) {
	// Count：view 方法，走 eth_call，不发交易、不耗 gas（由节点模拟执行）。
	n0, err := ctr.Count(&bind.CallOpts{Context: ctx})
	if err != nil {
		log.Fatalf("读取 count 失败: %v", err)
	}
	fmt.Println("increment 前 count =", n0.String())

	auth := mustTransactor(ctx, client, key)
	fmt.Println("正在发送 increment()…")
	// Increment：改状态，需签名 + gas，上链后 count 才会变。
	tx, err := ctr.Increment(auth)
	if err != nil {
		log.Fatalf("increment 失败: %v", err)
	}
	fmt.Println("increment 交易哈希:", tx.Hash().Hex())
	rec, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Fatalf("等待确认失败: %v", err)
	}
	if rec.Status != 1 {
		log.Fatalf("increment 执行失败 status=%d", rec.Status)
	}

	// 上链后再读一次，应与 n0 相差 1（在无其它并发修改的前提下）。
	n1, err := ctr.Count(&bind.CallOpts{Context: ctx})
	if err != nil {
		log.Fatalf("再次读取 count 失败: %v", err)
	}
	fmt.Println("increment 后 count =", n1.String())
	fmt.Println("浏览器: https://sepolia.etherscan.io/tx/" + tx.Hash().Hex())
}

// mustTransactor：构造「可发合约交易」的选项（链 ID、gas、上下文）。
// 每次写操作前新建一份，便于使用节点最新的建议 gasPrice / nonce 行为。
func mustTransactor(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey) *bind.TransactOpts {
	auth, err := bind.NewKeyedTransactorWithChainID(key, sepoliaChainID)
	if err != nil {
		log.Fatalf("创建 transactor 失败: %v", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		log.Fatalf("SuggestGasPrice: %v", err)
	}
	auth.GasPrice = gasPrice
	auth.GasLimit = 300000 // 部署/简单方法足够；复杂合约可改大或改用 EstimateGas
	auth.Context = ctx
	return auth
}

// mustPrivateKey：从环境变量读取并校验格式（勿提交私钥到仓库）。
func mustPrivateKey() *ecdsa.PrivateKey {
	raw := strings.TrimSpace(os.Getenv("SEPOLIA_PRIVATE_KEY"))
	raw = strings.Trim(raw, `"'`)
	pk := strings.TrimPrefix(strings.TrimPrefix(raw, "0x"), "0X")
	if len(pk) != 64 {
		log.Fatal("SEPOLIA_PRIVATE_KEY 应为 64 位十六进制（与 homework05 相同）")
	}
	key, err := crypto.HexToECDSA(pk)
	if err != nil {
		log.Fatalf("解析私钥失败: %v", err)
	}
	return key
}
