---
title: 简介
icon: /solidity.svg
order: 1
---

以太坊虚拟机（EVM）是以太坊区块链的核心，作为一个去中心化计算引擎来执行智能合约。

作为所有以太坊账户和智能合约的运行环境，EVM 使开发者能够部署可在区块链上运行的应用程序，而无需中央权威。

Solidity 是一种用于编写以太坊虚拟机（EVM）智能合约的编程语言。

## hello world

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.34;
contract HelloWeb3{
    string public _string = "Hello 0xroad!";
}
```

使用 [Remix](https://remix.ethereum.org) 运行 Solidity 合约。Remix 是以太坊官方推荐的智能合约集成开发环境（IDE），适合新手，可以在浏览器中快速开发和部署合约，无需在本地安装任何程序。

![](https://oss.jaronnie.com/image-20260328155807531.png)
