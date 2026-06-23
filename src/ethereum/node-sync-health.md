# Ethereum 节点同步与健康检查

## 主网节点需要执行层和共识层

The Merge 之后，以太坊主网节点被拆成两层客户端共同工作：

- 执行层客户端：Geth、Nethermind、Besu、Erigon 等。
- 共识层客户端：Prysm、Lighthouse、Teku、Nimbus、Lodestar 等。

Geth 是执行层客户端，负责执行交易、验证 EVM 状态转换、维护账户状态、
交易收据和日志，并提供 `eth_call`、`eth_getLogs`、`eth_getBlockByNumber`
等 JSON-RPC 接口。

Prysm 是共识层客户端，负责跟随 beacon chain、处理 slot 和 epoch、执行
fork choice、跟踪 finality，并告诉执行层应该导入哪些 execution payload，
以及当前 head、safe、finalized block 分别是谁。

两层客户端通过 Engine API 通信，通常使用本机 `8551` 端口，并通过共享的
JWT secret 认证：

```text
共识层客户端，例如 Prysm
        |
        | Engine API: engine_newPayload, engine_forkchoiceUpdated
        v
执行层客户端，例如 Geth
```

这意味着 Geth 不能独立完整地跟随以太坊主网。Geth 可以下载和验证执行层
数据，但它不能自己决定 PoS 主网的 canonical head。必须由 Prysm 这类共识层
客户端告诉它当前 head、safe block、finalized block，以及要导入的 execution
payload。

节点宕机一段时间后恢复，通常的追块流程是：

1. Geth 启动，恢复本地执行层数据库。
2. Prysm 启动，通过 `8551` 连接 Geth。
3. Prysm 从共识层 peers 下载缺失的 beacon blocks。
4. 每个 beacon block 里带有 execution payload，Prysm 会把它交给 Geth。
5. Geth 验证并导入对应的 execution block。
6. Prysm 通过 fork choice 更新 Geth 的 head、safe、finalized block。
7. 两层都追平后，节点恢复健康。

因此，`geth eth_syncing=false` 不能单独证明整个节点已经追上最新区块。它只
表示 Geth 当前没有处在自己的执行层同步模式里。

## 健康检查命令

下面命令都在节点机器上执行。

### 共识层：Prysm

查看 Prysm 同步状态：

```bash
curl -s http://127.0.0.1:3500/eth/v1/node/syncing
```

健康结果应该类似：

```json
{
  "data": {
    "head_slot": "14614172",
    "sync_distance": "0",
    "is_syncing": false,
    "is_optimistic": false,
    "el_offline": false
  }
}
```

字段含义：

- `is_syncing=false`：共识层没有在追块。
- `sync_distance=0`：距离网络最新 beacon head 为 0。
- `is_optimistic=false`：当前 head 对应的 execution payload 已被执行层验证。
- `el_offline=false`：Prysm 能通过 Engine API 正常连接 Geth。

这是最重要的健康检查。完整健康状态需要同时满足：

```text
is_syncing=false
sync_distance=0
is_optimistic=false
el_offline=false
```

查看 Prysm peer 数：

```bash
curl -s http://127.0.0.1:3500/eth/v1/node/peer_count
```

健康节点通常会有多个已连接的共识层 peer。不需要固定某个数字，但
`connected` 不应该是 `0`。

查看 Prysm 当前 head：

```bash
curl -s http://127.0.0.1:3500/eth/v1/beacon/headers/head
```

如果返回里有 `"execution_optimistic": true`，说明共识层已经看到 head，但
这个 head 对应的 execution payload 还没有被执行层完全验证。

### 执行层：Geth

查看 Geth 是否处在执行层同步模式：

```bash
curl -s -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}' \
  http://127.0.0.1:8545
```

Geth 没有主动同步时，健康结果通常是：

```json
{"jsonrpc":"2.0","id":1,"result":false}
```

如果 Geth 正在同步，`result` 会是一个对象，里面包含 `startingBlock`、
`currentBlock`、`highestBlock` 等进度字段。

查看 Geth 当前执行层块高：

```bash
curl -s -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":2}' \
  http://127.0.0.1:8545
```

返回值是十六进制，例如：

```json
{"jsonrpc":"2.0","id":2,"result":"0x1833e79"}
```

可以转成十进制：

```bash
printf "%d\n" 0x1833e79
```

查看 Geth peer 数：

```bash
curl -s -H "Content-Type: application/json" \
  --data '{"jsonrpc":"2.0","method":"net_peerCount","params":[],"id":3}' \
  http://127.0.0.1:8545
```

返回值也是十六进制。peer 数不应该是 `0`。

### 实际判断方式

节点完全追上时，通常需要同时满足：

```text
Prysm is_syncing=false
Prysm sync_distance=0
Prysm is_optimistic=false
Prysm el_offline=false
Geth eth_syncing=false
Geth blockNumber 持续增长
Prysm 和 Geth 都有 peers
```

常见状态组合：

- `geth eth_syncing=false` 且 `prysm is_syncing=true`：Geth 自己没有在执行层
  主动同步，但整个节点还没追平，因为共识层还在追 beacon slots。
- `prysm is_optimistic=true`：Prysm 已经乐观地接受了某个 head，但 Geth 还
  需要验证对应的 execution payload。
- `prysm el_offline=true`：Prysm 连不上 Geth 的 Engine API。重点检查 Geth、
  `8551` 端口、JWT secret 路径、`--execution-endpoint` 配置。
- `geth eth_syncing` 返回对象：Geth 正在做执行层同步。需要等它变成 `false`，
  然后继续检查 Prysm 的状态。

## 常见日志

Prysm 正在追 beacon blocks：

```text
initial-sync: Processing block 14614272/14614368
```

Prysm 正在把 execution payload 发给 Geth：

```text
Called new payload with optimistic block
```

Geth 正在导入执行层区块：

```text
Imported new potential chain segment
Chain head was updated
```

看到这些日志，一般说明节点正在追块，不一定是卡住了。
