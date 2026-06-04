---
title: jito-solana
icon: /Solana.svg
order: 2
---

## 简介

Jito-Solana 是 Jito 对 Solana validator 的一个 fork。它保留 Solana/Agave validator 的核心链功能，同时加入 Jito 生态需要的低延迟、MEV、bundle、tip 等相关能力。

## 相关概念

### slot && block

slot 是 Solana 的出块时间窗口；block 是这个 slot 中成功产生并被验证的实际区块数据。

### bundle

一组有顺序要求的交易包, 因为很多链上策略不是单笔交易能完成的，而是需要多笔交易配合，比如套利。

### Validator && Builder && Searcher

Searcher 发现机会并提交高价值交易或 bundle，Builder 负责把这些交易或 bundle 组合成更优区块，Validator 最终负责执行、投票、出块并维护网络共识。

### PoH

PoH, Proof of History, 是 Solana 的可验证时间序列机制。它不是共识本身，而是 leader 出块和 validator 验证时使用的“可验证时钟”。

Leader 持续计算一条串行 hash 链：
```
hash_0
hash_1 = sha256(hash_0)
hash_2 = sha256(hash_1)
hash_3 = sha256(hash_2)
```

SHA256 hash 链不能被并行跳算。某个 hash 出现在第 N 步，意味着前面确实经历了 N 次顺序计算。Validator 后续可以重算这段 hash 链，验证 leader 给出的时间推进和事件顺序。


### Solana ledger

Solana ledger 数据就是 validator 存下来的 slot/block/entry/transaction/shred 及其元数据，用来恢复历史、验证 PoH、重放交易并推导账户状态。

### Shred

shred 是 Solana 里区块数据的小切片，Leader 不会把一个完整 block 一次性发给所有 validator，而是把 PoH 产生的 entries 序列化后切成很多小片，每一片就是一个 shred。

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

shred 就是 Solana 为了快速传播区块，把 block/entries 切出来的网络数据片。