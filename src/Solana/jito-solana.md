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