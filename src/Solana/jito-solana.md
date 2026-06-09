---
title: jito-solana
icon: /Solana.svg
order: 2
---

## 简介

Jito-Solana 是 Jito 对 Solana validator 的一个 fork。

它保留 Solana/Agave validator 的核心链功能，同时加入 Jito 生态需要的低延迟交易流、MEV、bundle、tip 等能力。理解 Jito-Solana 之前，先把 Solana validator 本身的几个基础概念理顺会更容易。

## 一句话理解

Solana validator 负责按 slot 出块、验证 PoH、传播 shred、重放 ledger 并维护共识。

Jito-Solana 在这个基础上增加了一条更适合 MEV 的交易入口：Searcher 可以提交有顺序要求的 bundle，并通过 tip 激励当前 leader 优先打包。

```shell
Searcher 发现机会
  ↓
提交 bundle + tip
  ↓
Jito block engine / relayer 转发
  ↓
Jito-Solana leader 接收并排序
  ↓
执行交易并生成 block
  ↓
切成 shreds 广播给其他 validators
```

## 相关概念

### Solana 的时间模型

#### slot

slot 是 Solana 的出块时间窗口。

每个 slot 会指定一个 leader。这个 leader 在自己的 slot 里负责接收交易、执行交易、生成 PoH entries，并把结果传播给其他 validators。

slot 不等于一定有 block。leader 可能因为网络、性能或其他原因没有成功出块。

#### tick

tick 是 PoH 时间线上的“时间刻度”。

leader 在自己的 slot 里会持续推进 PoH hash 链。即使某一小段时间里没有交易，也会生成 tick 来证明时间确实在前进。

可以把 tick 理解成 Solana 内部时钟的节拍：

```shell
slot
  ├── tick
  ├── transaction entries
  ├── tick
  ├── transaction entries
  └── tick
```

tick 本身不是一笔交易，也不代表用户状态变化。它更像是一个空的 PoH 记录点，用来告诉其他 validators：

- 这段时间已经过去
- PoH hash 链没有断
- 当前 slot 正在按预期推进
- slot 边界可以被验证

因此，slot 是较大的出块时间窗口，tick 是这个窗口里的更细粒度时间刻度。

#### block

block 是某个 slot 中实际成功产生并被验证的区块数据。

可以这样区分：

```shell
slot  = 出块机会 / 时间窗口
block = 这个窗口里真正产出的数据
```

所以，一个 slot 可能有 block，也可能没有 block。

#### epoch

epoch 是由很多个 slot 组成的更大周期。

Solana 用 epoch 组织一些周期性事务，例如：

- leader 排班
- stake 更新
- validator 奖励
- staking 激活与解除

可以把三者关系理解成：

```shell
epoch
  └── many slots
        ├── many ticks
        └── optional block
```

### PoH：Solana 的可验证时钟

PoH 是 Proof of History，中文通常叫历史证明。

它不是共识本身，而是 Solana 用来记录时间推进和事件顺序的可验证时间序列。leader 出块时会持续计算一条串行 hash 链：

```shell
hash_0
hash_1 = sha256(hash_0)
hash_2 = sha256(hash_1)
hash_3 = sha256(hash_2)
...
```

SHA256 hash 链不能被并行跳算。某个 hash 出现在第 N 步，意味着前面确实经历了 N 次顺序计算。

这带来两个作用：

- leader 可以把交易插入到这条时间线上，形成明确顺序
- leader 可以用 tick 标记时间推进和 slot 边界
- validator 后续可以重算 hash 链，验证时间推进、slot 边界和事件顺序是否可信

更直观地说，PoH 是 Solana 出块过程里的“时钟”和“排序骨架”。

### 从交易到 ledger

#### entry

leader 执行交易后，会把交易结果和 PoH hash 组织成 entries。

entry 可以理解为 ledger 里的连续记录单元：它记录了某段 PoH 时间内发生了哪些交易，或者记录一个没有交易的 tick。

所以，entry 可以粗略分成两类：

```shell
transaction entry = 带交易的 PoH 记录
tick entry        = 不带交易的 PoH 时间刻度
```

validators replay ledger 时，不只是执行 transaction entries，也会验证 tick entries，确认 PoH 时间线和 slot 边界是连续可信的。

#### ledger

Solana ledger 是 validator 持久化保存的链上历史数据。

它包含 slot、block、entry、transaction、shred 以及相关元数据，用来：

- 恢复历史
- 验证 PoH
- 重放交易
- 推导账户状态
- 支持 validator 重启后继续同步

validator 并不是只保存一个最终状态，它还需要保存足够的历史数据来证明和重放这个状态是怎么来的。

### TPU 和 TVU：validator 的入口与验证流水线

理解 Jito-Solana 时，经常会看到 TPU、TVU 这两个词。它们都不是某种新的共识机制，而是 Solana validator 内部处理数据的两条关键流水线。

可以先用一句话区分：

```shell
TPU = Transaction Processing Unit，leader 用来接收、转发、执行交易并出块
TVU = Transaction Validation Unit，validator 用来接收 shreds、重组 block、replay 验证
```

#### TPU

TPU 是交易进入当前 leader 的主要入口。

当一个 validator 是当前 slot 的 leader 时，它会通过 TPU 接收来自 RPC、其他节点转发、Jito 交易流等来源的交易，然后完成筛选、排序、执行、生成 entries，并最终把 entries 切成 shreds 广播出去。

可以粗略理解成：

```shell
用户 / RPC / searcher
  ↓
交易发送到 leader TPU
  ↓
TPU 接收、过滤、转发、排序
  ↓
BankingStage 执行交易
  ↓
PoH 记录交易顺序
  ↓
生成 entries
  ↓
切成 shreds
  ↓
广播给其他 validators
```

TPU 侧关注的是“交易如何尽快进入 leader 并被打包”。所以在 Jito-Solana 里，bundle、tip、block engine / relayer 这些能力，本质上都是围绕 leader 的交易入口和排序选择展开的。

#### TVU

TVU 是 validator 接收和验证区块数据的流水线。

当其他 leader 正在出块时，普通 validator 会通过网络收到这个 leader 广播出来的 shreds。TVU 负责接收这些 shreds，校验、去重、写入 blockstore，等待数据完整后重组 entries，再交给 replay 逻辑执行和验证。

可以粗略理解成：

```shell
Leader 广播 shreds
  ↓
Validator TVU 接收 shreds
  ↓
校验签名、slot、索引和完整性
  ↓
写入 Blockstore
  ↓
重组 entries
  ↓
ReplayStage 重放交易
  ↓
验证状态并投票
```

TVU 侧关注的是“别人出的 block 是否能被我及时收到、重组、验证并参与投票”。它不负责决定当前 leader 要把哪些新交易打进 block，而是负责验证已经由 leader 产出的 ledger 数据。

#### TPU 和 TVU 的关系

同一个 validator 在不同 slot 中会扮演不同角色：

- 轮到自己当 leader 时，TPU 更关键：它要尽快接收交易、执行交易并产出 shreds。
- 没轮到自己当 leader 时，TVU 更关键：它要尽快接收别人的 shreds、重组 block 并 replay。
- 一个 validator 通常同时维护这些组件，因为它既要准备未来自己的 leader slot，也要持续验证其他 leader 的 block。

用一张图串起来：

```shell
当前 leader
  TPU 接收交易
  ↓
执行交易 + PoH 排序
  ↓
entries → shreds
  ↓ Turbine
其他 validators
  TVU 接收 shreds
  ↓
Blockstore 重组
  ↓
ReplayStage 验证
  ↓
投票 / 更新状态
```

因此，TPU 和 TVU 不是互相替代的概念，而是 Solana 高性能流水线的两侧：

```shell
TPU 偏生产：交易进入 leader，形成 block
TVU 偏消费：接收 shreds，验证和重放 block
```

理解这一点后，再看 Jito-Solana 会更清楚：Jito 主要优化的是高价值交易、bundle 和 tip 如何进入 leader 并参与排序，也就是偏 TPU 入口侧；而 block 一旦被产出，仍然要通过 shred 传播、TVU 接收、ReplayStage 验证，并接受 Solana 原有共识流程的约束。

### shred：Solana 的区块传播单位

leader 不会把一个完整 block 一次性发给所有 validators。

它会把 PoH 产生的 entries 序列化后切成很多小片，每一片就是一个 shred。这样可以让区块数据更快地通过网络传播，也方便在部分数据丢失时恢复。

```shell
Leader 执行交易
  ↓
生成 PoH entries
  ↓
entries 切成 shreds
  ↓
通过 Turbine 广播
  ↓
Validator 收集 shreds
  ↓
Blockstore 重组 entries
  ↓
ReplayStage 验证和执行
```

#### Data shred

Data shred 包含真实的 block 数据，也就是 entries 序列的一部分。

validator 最终需要收集连续的 data shreds，才能重组出完整 entries，并继续 replay。

#### Coding shred

Coding shred 不包含新的 ledger 内容。

它是基于一组 data shreds 生成的纠删码数据，用于在部分 data shred 丢失时恢复缺失内容。

可以简单理解为：

```shell
Data Shred   = 原始区块数据
Coding Shred = 用来恢复丢失数据的冗余数据
```

#### Shred 传播和跳数

Solana 使用 Turbine 传播 shreds。Turbine 不是让 leader 把所有 shreds 直接发给所有 validators，而是把网络组织成类似树状的多层传播结构。

可以粗略理解为：

```shell
Leader
  ↓ 0 跳
Root / 第一批接收节点
  ↓ 1 跳
下一层 validators
  ↓ 2 跳
更下一层 validators
```

这里的“跳”指 shred 从 leader 出发后，经过了几次中继转发：

- 0 跳：直接从当前 slot 的 leader 或非常靠近 leader 的上游源收到 shred
- 1 跳：从第一层接收节点转发过来的 shred
- 2 跳及以上：经过更多层 Turbine 转发后收到的 shred

跳数越少，通常意味着收到 shred 的时间越早。对普通 validator 来说，差几十毫秒通常只是网络同步体验差异；但对 searcher、做市、套利、清算、RPC 索引和高频策略来说，更早看到 shreds 可以更早推断当前 block 里已经包含了哪些交易，以及链上状态即将如何变化。

因此，很多低延迟服务会强调“0 跳 shreds”或“低跳数 shreds”。它们的核心卖点不是改变 Solana 共识，也不是绕过 shred 验证，而是尽量减少从 leader 产出 shred 到客户端收到 shred 之间的传播路径。

需要注意：

- 0 跳是网络传播位置概念，不是共识状态
- 收到 0 跳 shred 不代表交易已经 finality
- shred 仍然需要校验 leader 签名、slot、索引和数据完整性
- 更早收到 shred 只能带来信息延迟优势，不能保证策略一定成交或获利

### Jito 相关概念

#### MEV

MEV 是 Maximal Extractable Value，指通过交易排序、插入、组合或选择性打包可以提取的额外价值。

在 Solana 里，常见场景包括套利、清算、抢先成交、跨市场价格差捕获等。

Jito 的核心目标不是消除 MEV，而是让 MEV 的提交、排序和收益分配更透明、更高效，并减少无序抢跑对网络造成的垃圾流量。

#### searcher

Searcher 是发现链上机会的人或程序。

Searcher 会监听市场、链上状态和交易流，构造可以获利的交易或 bundle，然后提交给 Jito。

#### bundle

bundle 是一组有顺序要求的交易包。

很多策略不是单笔交易能完成的，而是需要多笔交易配合。例如：

```shell
交易 A：在市场 1 买入
交易 B：在市场 2 卖出
交易 C：返还借款或完成结算
```

这些交易必须按顺序执行，并且通常希望要么全部成功，要么全部失败。如果只执行其中一部分，策略可能亏损或留下风险。

#### tip

tip 是 searcher 为了让 bundle 被 leader 优先选择而支付的额外激励。

在 Jito 体系里，leader 可以根据 bundle 的价值和 tip 做排序选择。tip 越高，不代表一定会被打包，但它会影响 bundle 的竞争力。

#### builder

Builder 负责把普通交易、bundle 和其他交易流组合成更有价值的区块候选。

在以太坊语境中，builder 通常是独立角色；在 Solana/Jito 语境下，可以先把它理解成“负责优化区块内容和排序的组件或角色”。

#### validator

Validator 最终负责执行交易、投票、出块并维护网络共识。

当某个 Jito-Solana validator 成为当前 slot 的 leader 时，它可以接收来自 Jito 交易流的 bundle，并把符合条件的交易打进自己生产的 block。

### 角色关系

Searcher、Builder、Validator 的关系可以这样理解：

```shell
Searcher
  发现机会，提交交易或 bundle
  ↓
Builder / Jito block engine
  聚合交易流，排序，选择更优组合
  ↓
Validator / Leader
  执行交易，生成 block，广播 shreds，参与共识
```

更短地说：

- Searcher 负责找机会
- Builder 负责组织交易
- Validator 负责真正出块和维护链

### Jito-Solana 改了什么

Jito-Solana 不是一条新链，而是 Solana validator 的增强版本。

它主要增强的是交易进入 leader 和被排序打包的路径：

- 支持 bundle 提交
- 支持 tip 激励
- 支持更低延迟的交易转发
- 支持和 Jito block engine / relayer 协作
- 减少 searcher 直接向网络狂发交易造成的拥堵

可以把普通 Solana 和 Jito-Solana 的差异粗略理解成：

```shell
普通 Solana:
用户交易 → RPC / TPU → Leader → Block

Jito-Solana:
用户交易 / Searcher bundle → Jito 交易流 → Jito-Solana Leader → Block
```

### 总结

理解 Jito-Solana 可以按这个顺序：

1. Solana 用 slot 安排谁来出块。
2. leader 用 PoH 记录时间推进和交易顺序。
3. tick 是 PoH 时间线里的时间刻度，用来证明 slot 正在推进。
4. 执行结果和 tick 会形成 entries。
5. entries 被切成 shreds 在网络中传播。
6. validators 收集 shreds，重组 ledger，并 replay 验证状态。
7. Jito-Solana 在 leader 接收交易的路径上增加 bundle、tip 和 MEV 相关能力。

所以，Jito-Solana 的核心不是改变 Solana 的共识模型，而是改进高价值交易如何进入 leader、如何排序、以及收益如何分配。

## 编译 jito-solana

```shell
#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/cargo-docker-build.sh [--shell] [--image IMAGE] [--] [COMMAND...]

Run the Cargo GitHub Actions job locally in Docker.

By default this reproduces .github/workflows/cargo.yml's Alpine nightly clippy
job:

  scripts/cargo-docker-build.sh

Pass a command to run something else after the CI dependencies are installed:

  scripts/cargo-docker-build.sh -- ./cargo build --workspace
  scripts/cargo-docker-build.sh -- scripts/cargo-clippy-nightly.sh

Options:
  --image IMAGE Docker image to use (default: docker.io/rust:1-alpine3.22)
  -h, --help     Show this help
EOF
}

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
image="${CARGO_DOCKER_IMAGE:-docker.io/rust:1-alpine3.22}"

while [[ $# -gt 0 ]]; do
  case "$1" in
  --image)
    if [[ $# -lt 2 ]]; then
      echo "error: --image requires a value" >&2
      exit 1
    fi
    image="$2"
    shift 2
    ;;
  --)
    shift
    break
    ;;
  -h | --help)
    usage
    exit 0
    ;;
  *)
    break
    ;;
  esac
done

if ! command -v docker >/dev/null 2>&1; then
  echo "error: docker is not installed or is not on PATH" >&2
  exit 1
fi

host_uid="$(id -u)"
host_gid="$(id -g)"

container_script='
set -euo pipefail

finish() {
  chown -R "$HOST_UID:$HOST_GID" /solana/target /usr/local/cargo/registry /usr/local/cargo/git 2>/dev/null || true
}
trap finish EXIT

cd /solana

git config --global --add safe.directory /solana

# Match the GitHub Actions job setup from .github/workflows/cargo.yml.
source .github/scripts/install-all-deps.sh Linux
source ci/rust-version.sh nightly
rustup component add clippy --toolchain "$rust_nightly"


if [[ $# -eq 0 ]]; then
  exec scripts/cargo-clippy-nightly.sh
fi

exec "$@"
'

bootstrap_script='
set -eu

apk update
apk add bash git

bash -c "$CONTAINER_SCRIPT" bash "$@"
'

docker_args=(
  -it
  --workdir /solana
  --volume "$repo_root:/solana"
  --env CONTAINER_SCRIPT="$container_script"
  --env PATH=/usr/local/cargo/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
  --env SHELL=/bin/bash
  --env RUSTFLAGS="-C target-feature=-crt-static"
  --env RUSTC_WRAPPER=
  --env HOST_UID="$host_uid"
  --env HOST_GID="$host_gid"
)

exec docker run "${docker_args[@]}" "$image" sh -c "$bootstrap_script" sh "$@"
```
