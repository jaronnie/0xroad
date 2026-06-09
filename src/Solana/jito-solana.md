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

### Solana gossip：节点发现和集群信息传播

Gossip 是 Solana validator 之间用来发现彼此、交换节点元数据和维护集群视图的网络协议。

它不是交易执行路径，也不是 block 传播路径。它更像是 Solana 集群的“通讯录”和“状态广播层”：每个节点通过 gossip 知道网络里有哪些节点、这些节点的地址是什么、提供哪些服务端口、当前 shred version 是什么，以及一些和集群运行相关的元信息。

可以粗略理解成：

```shell
Validator A
  宣布自己的身份、公钥、IP、端口、版本等信息
  ↓ gossip
Validator B / C / D
  收到后更新本地集群视图
  ↓ gossip
更多 validators 继续传播这些信息
```

#### Gossip 传播什么

Solana gossip 传播的是集群元数据，不是完整 block 数据。

常见信息包括：

- 节点身份：validator 的 identity pubkey。
- ContactInfo：节点对外暴露的网络地址和端口，例如 gossip、TPU、TVU、repair、RPC 等。
- shred version：用于区分当前网络和 ledger 兼容性，避免错误网络的数据混在一起。
- 节点版本和 feature 信息：帮助其他节点理解对方运行的客户端能力。
- vote 相关信息：validator 的投票身份、投票账户等元数据。
- snapshot、repair、serve repair 等服务入口：帮助新节点或落后节点同步和修复数据。

这些信息通常会被组织在 CRDS 中，也就是 Cluster Replicated Data Store。可以把 CRDS 理解成每个节点本地维护的一份“集群成员和元数据表”，gossip 负责让这张表在网络中不断扩散和收敛。

#### Gossip 不传播什么

Gossip 容易和 Turbine、TPU、TVU 混在一起，但它们负责的层不同。

Gossip 不负责：

- 把用户交易直接送进 leader 执行。
- 把 leader 产出的 shreds 高速广播给全网。
- replay ledger 或验证交易执行结果。
- 决定 MEV bundle 如何排序。

这些事情分别由其他组件负责：

```shell
Gossip  = 节点发现、地址和集群元数据传播
TPU     = leader 接收交易、执行交易、产出 block
Turbine = leader 把 shreds 扩散给 validators
TVU     = validator 接收 shreds、重组 block、replay 验证
```

#### Gossip 和 TPU / TVU 的关系

TPU 和 TVU 要正常工作，首先需要知道其他节点在哪里。

Gossip 提供的 ContactInfo 会告诉 validator：

- 某个 leader 的 TPU 地址在哪里，可以把交易转发过去。
- 某个 validator 的 TVU、repair 地址在哪里，可以接收或请求区块数据。
- 当前网络中的可见节点集合是什么。
- 节点是否属于同一个 shred version 的网络。

所以，gossip 更偏控制面，TPU、TVU、Turbine 更偏数据面。

```shell
控制面:
Gossip 传播节点身份、地址、版本和服务入口

数据面:
TPU 传交易
Turbine 传 shreds
TVU 收 shreds 并 replay
```

#### Gossip 的传播方式

Gossip 的设计目标不是让每条元信息都瞬间到达所有节点，而是通过节点之间不断交换信息，让集群视图逐渐收敛。

一个节点会周期性地选择一些 peer 交换自己知道的 CRDS 数据。其他节点收到后，会根据时间戳、签名和版本等信息判断是否接受、更新或丢弃。

这类传播方式的特点是：

- 扩散性强：不需要中心节点维护完整网络目录。
- 最终收敛：短时间内不同节点看到的集群视图可能不同，但会逐渐一致。
- 抗节点上下线：节点加入、退出、换地址时，可以通过 gossip 被其他节点感知。
- 适合元数据：适合传播节点地址和状态，不适合承载高吞吐 block 数据。

#### 为什么 gossip 对 Jito-Solana 也重要

Jito-Solana 主要增强的是交易进入 leader 的路径，例如 bundle、tip、Block Engine、BAM 等。但它仍然是一个 Solana validator，仍然需要依赖 gossip 参与集群网络。

对 Jito-Solana 来说，gossip 至少有几个作用：

- 发现其他 validators 和当前集群成员。
- 获取 leader、TPU、TVU、repair 等网络入口信息。
- 确认 shred version，避免接入错误网络。
- 支持节点同步、repair、replay 等基础流程。

但 gossip 本身不决定 Jito bundle 是否被打包，也不负责 BAM 的交易排序。Jito 和 BAM 优化的是出块前的交易入口和排序；gossip 维护的是 validator 在 Solana 网络中的基础连通性和集群视图。

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

#### BAM

BAM 通常指 Jito 的 Block Assembly Marketplace，是面向 Solana 的新一代交易调度和组块架构。

如果说传统 Jito Block Engine 更像是“把 bundle 和高价值交易更高效地送到 leader”，那么 BAM 更进一步：它把交易排序这件事从 validator 本地内部逻辑中抽出来，交给专门的 BAM Node 调度，然后再把已经排好序的交易流发送给即将出块的 BAM Validator 执行。

可以粗略理解成：

```shell
用户交易 / searcher bundle / Jito Block Engine 流
  ↓
BAM Node
  接收交易和 bundle
  验证交易有效性
  根据规则和插件排序
  在 TEE 中保护交易隐私
  生成可验证的排序结果
  ↓
BAM Validator / Leader
  接收排好序的交易流
  按顺序执行交易
  生成 entries
  切成 shreds 广播
  ↓
其他 validators
  TVU 接收 shreds
  replay 验证
```

BAM 的关键点有几个：

- 外部调度：BAM Node 负责接收、验证、排序交易和 bundle。
- Validator 仍然出块：BAM Validator 仍然负责执行交易、生产 block、广播 shreds 和参与共识。
- TEE：BAM Node 可以在可信执行环境中调度交易，降低交易在进入执行前被窥探的风险。
- 可验证排序：BAM 通过签名和证明让交易排序过程更容易审计。
- 插件：开发者可以通过插件表达更定制化的排序规则，例如特定应用的公平排序、优先级逻辑或 MEV 保护策略。

所以，BAM 不是一条新链，也不是替代 Solana 共识。它改变的是 leader 生产 block 之前“交易如何被调度和排序”的市场结构。

#### BAM Node

BAM Node 是 BAM 架构里的外部调度节点。

它会接收普通交易、bundle、Solana cluster state 和当前 leader 信息，然后做交易有效性检查、排序、调度，并把交易流发送给连接的 BAM Validator。

一个 BAM Node 可以服务多个 BAM Validators。它更像是一个专业化的 transaction scheduler，而不是负责投票和共识的 validator。

#### BAM Validator

BAM Validator 是连接到 BAM Node 的 Jito-Solana validator。

它的核心职责仍然是 validator 的职责：执行交易、生成 block、广播 shreds、投票和维护共识。区别在于，当它接入 BAM 后，leader slot 中的交易流会从 BAM Node 过来，并按 BAM Node 给出的顺序执行。

需要注意：

- BAM Validator 一次通常只连接一个 BAM Node。
- Validator 仍然拥有最终的 block production 职责。
- BAM 不直接影响 TVU 验证别人 block 的基本流程。
- 如果交易最终被执行并产出 block，后续仍然要进入 shred 传播、Blockstore 重组和 ReplayStage 验证。

#### BAM 和 TPU 的关系

前面说过，TPU 是 leader 接收交易并出块的入口侧。BAM 可以理解为对 TPU 入口侧交易调度能力的进一步外置和增强。

普通 Jito-Solana 的路径更像是：

```shell
交易 / bundle
  ↓
Jito Block Engine / relayer
  ↓
Leader TPU
  ↓
本地排序和执行
```

接入 BAM 后，路径更像是：

```shell
交易 / bundle
  ↓
BAM Node 外部调度
  ↓
BAM Validator / Leader
  ↓
按调度结果执行
```

官方文档里也强调：当 validator 接入 BAM 且即将成为 leader 时，通过 RPC 或直接发到 TPU 的交易也会经过 BAM 处理后再执行。

因此，BAM 关注的仍然是出块前的交易入口和排序，不改变 block 产出后的传播与验证路径。

#### BAM、Block Engine、Bundle 的关系

BAM 不等于 bundle，也不只是 Block Engine 的另一个名字。

可以这样区分：

```shell
bundle       = 一组有顺序要求的交易
Block Engine = Jito 现有的交易 / bundle 转发和预模拟基础设施
BAM Node     = 更通用的外部交易调度和排序节点
BAM Validator = 接收 BAM Node 交易流并负责出块的 validator
```

BAM 可以继续兼容 Jito bundles。也就是说，bundle 仍然是 searcher 表达复杂交易意图的一种方式，而 BAM 是更上层的排序和组块市场架构。

#### BAM 的意义

BAM 想解决的是 Solana block building 中更深一层的问题：谁来排序、排序是否可验证、应用能不能表达自己的排序规则、交易在执行前是否能减少被窥探。

它带来的变化可以概括为：

- 从单纯的低延迟交易转发，走向可编程交易调度。
- 从 validator 本地排序，走向外部专业调度节点和 validator 协作。
- 从黑盒排序，走向带签名和证明的可审计排序。
- 从通用排序规则，走向应用可以通过插件定制执行逻辑。

但它也有边界：

- BAM 不改变 Solana 的 slot、PoH、shred、replay 和投票模型。
- BAM 不保证某笔交易一定成功，只是改变交易进入 leader 和被排序的方式。
- TEE 降低交易供应链中的窥探风险，但不等于消除所有 MEV。
- 交易执行后的状态仍然由 Solana validator replay 和共识流程决定。

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

Jito-Solana + BAM:
用户交易 / Searcher bundle → BAM Node 调度排序 → BAM Validator Leader → Block
```

BAM 出现后，可以把 Jito-Solana 的演进理解成两层：

- Jito-Solana 增强 validator 的交易入口和 MEV 交易流能力。
- BAM 把交易调度和排序进一步模块化，让外部 BAM Node、TEE、插件和 validator 形成协作。

### 总结

理解 Jito-Solana 可以按这个顺序：

1. Solana 用 slot 安排谁来出块。
2. leader 用 PoH 记录时间推进和交易顺序。
3. tick 是 PoH 时间线里的时间刻度，用来证明 slot 正在推进。
4. 执行结果和 tick 会形成 entries。
5. entries 被切成 shreds 在网络中传播。
6. validators 收集 shreds，重组 ledger，并 replay 验证状态。
7. Jito-Solana 在 leader 接收交易的路径上增加 bundle、tip 和 MEV 相关能力。
8. BAM 在这个基础上把交易调度和排序进一步外置到 BAM Node，并通过 TEE、可验证排序和插件增强 block building。
9. Gossip 负责节点发现和集群元数据传播，为 TPU、TVU、Turbine 等数据路径提供基础网络视图。

所以，Jito-Solana 的核心不是改变 Solana 的共识模型，而是改进高价值交易如何进入 leader、如何排序、以及收益如何分配。

BAM 的核心也不是改变共识，而是让出块前的交易调度更隐私、更可验证、更可编程。

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
