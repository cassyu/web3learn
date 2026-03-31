# 链上小练习：Solidity + 测试 + 部署

对应学习路线「Solidity 小合约 + 测试部署」的最小可运行示例。

## Hardhat（当前目录，推荐先跑通）

```bash
cd onchain
npm install
npm run compile
npm run test
npm run deploy:local
```

- **合约**：`contracts/MyToken.sol`（ERC721 + URI + Ownable，与仓库根目录 `nft.sol` 思路一致）
- **测试**：`test/MyToken.test.js`
- **部署脚本**：`scripts/deploy.js`（默认连内置 Hardhat 网络）

部署到测试网时，在 `hardhat.config.js` 里配置 `networks` 与账户（例如 `PRIVATE_KEY` / Alchemy URL），再：

```bash
npx hardhat run scripts/deploy.js --network <你的网络名>
```

## Foundry（可选）

本机需先安装 [Foundry](https://book.getfoundry.sh/getting-started/installation)，然后在**单独目录**执行：

```bash
forge init my-forge --no-commit
cd my-forge
forge install OpenZeppelin/openzeppelin-contracts
# 将本仓库 contracts/MyToken.sol 拷到 src/，编写 test/MyToken.t.sol 后：
forge test
forge script script/Deploy.s.sol --rpc-url <RPC> --broadcast
```

Foundry 与 Hardhat 二选一即可入门；熟练后可同一项目混用。
