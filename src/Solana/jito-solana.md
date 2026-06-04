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

## Solana 的时间模型

### slot

slot 是 Solana 的出块时间窗口。

每个 slot 会指定一个 leader。这个 leader 在自己的 slot 里负责接收交易、执行交易、生成 PoH entries，并把结果传播给其他 validators。

slot 不等于一定有 block。leader 可能因为网络、性能或其他原因没有成功出块。

### block

block 是某个 slot 中实际成功产生并被验证的区块数据。

可以这样区分：

```shell
slot  = 出块机会 / 时间窗口
block = 这个窗口里真正产出的数据
```

所以，一个 slot 可能有 block，也可能没有 block。

### epoch

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
        └── optional block
```

## PoH：Solana 的可验证时钟

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
- validator 后续可以重算 hash 链，验证时间推进和事件顺序是否可信

更直观地说，PoH 是 Solana 出块过程里的“时钟”和“排序骨架”。

## 从交易到 ledger

### entry

leader 执行交易后，会把交易结果和 PoH hash 组织成 entries。

entry 可以理解为 ledger 里的连续记录单元：它记录了某段 PoH 时间内发生了哪些交易。

### ledger

Solana ledger 是 validator 持久化保存的链上历史数据。

它包含 slot、block、entry、transaction、shred 以及相关元数据，用来：

- 恢复历史
- 验证 PoH
- 重放交易
- 推导账户状态
- 支持 validator 重启后继续同步

validator 并不是只保存一个最终状态，它还需要保存足够的历史数据来证明和重放这个状态是怎么来的。

## shred：Solana 的区块传播单位

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

### Data shred

Data shred 包含真实的 block 数据，也就是 entries 序列的一部分。

validator 最终需要收集连续的 data shreds，才能重组出完整 entries，并继续 replay。

### Coding shred

Coding shred 不包含新的 ledger 内容。

它是基于一组 data shreds 生成的纠删码数据，用于在部分 data shred 丢失时恢复缺失内容。

可以简单理解为：

```shell
Data Shred   = 原始区块数据
Coding Shred = 用来恢复丢失数据的冗余数据
```

## Jito 相关概念

### MEV

MEV 是 Maximal Extractable Value，指通过交易排序、插入、组合或选择性打包可以提取的额外价值。

在 Solana 里，常见场景包括套利、清算、抢先成交、跨市场价格差捕获等。

Jito 的核心目标不是消除 MEV，而是让 MEV 的提交、排序和收益分配更透明、更高效，并减少无序抢跑对网络造成的垃圾流量。

### searcher

Searcher 是发现链上机会的人或程序。

Searcher 会监听市场、链上状态和交易流，构造可以获利的交易或 bundle，然后提交给 Jito。

### bundle

bundle 是一组有顺序要求的交易包。

很多策略不是单笔交易能完成的，而是需要多笔交易配合。例如：

```shell
交易 A：在市场 1 买入
交易 B：在市场 2 卖出
交易 C：返还借款或完成结算
```

这些交易必须按顺序执行，并且通常希望要么全部成功，要么全部失败。如果只执行其中一部分，策略可能亏损或留下风险。

### tip

tip 是 searcher 为了让 bundle 被 leader 优先选择而支付的额外激励。

在 Jito 体系里，leader 可以根据 bundle 的价值和 tip 做排序选择。tip 越高，不代表一定会被打包，但它会影响 bundle 的竞争力。

### builder

Builder 负责把普通交易、bundle 和其他交易流组合成更有价值的区块候选。

在以太坊语境中，builder 通常是独立角色；在 Solana/Jito 语境下，可以先把它理解成“负责优化区块内容和排序的组件或角色”。

### validator

Validator 最终负责执行交易、投票、出块并维护网络共识。

当某个 Jito-Solana validator 成为当前 slot 的 leader 时，它可以接收来自 Jito 交易流的 bundle，并把符合条件的交易打进自己生产的 block。

## 角色关系

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

## Jito-Solana 改了什么

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

## 总结

理解 Jito-Solana 可以按这个顺序：

1. Solana 用 slot 安排谁来出块。
2. leader 用 PoH 记录时间推进和交易顺序。
3. 执行结果会形成 entries。
4. entries 被切成 shreds 在网络中传播。
5. validators 收集 shreds，重组 ledger，并 replay 验证状态。
6. Jito-Solana 在 leader 接收交易的路径上增加 bundle、tip 和 MEV 相关能力。

所以，Jito-Solana 的核心不是改变 Solana 的共识模型，而是改进高价值交易如何进入 leader、如何排序、以及收益如何分配。
