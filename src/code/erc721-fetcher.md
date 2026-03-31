---
title: ERC721 Token URI Fetcher
icon: proicons:code-square
order: 1
---

一个用 Go 语言编写的 ERC721 合约调用工具，用于获取 NFT 的 Token URI 和元数据，支持自动保存链上 NFT 图片。

源码: https://github.com/jaronnie/0xroad/tree/main/code/erc721-fetcher

## 功能

- 通过以太坊 JSON-RPC API 调用 ERC721 合约的 `tokenURI` 方法
- 解码合约返回的 ABI 编码数据
- 支持从 IPFS 获取 NFT 元数据
- **支持链上 NFT（Base64 编码的元数据）**
- **自动下载并保存 NFT 图片到本地文件**（支持 IPFS 和 HTTP URL）
- 无需依赖 go-ethereum 等第三方库，仅使用 Go 标准库

## 使用方法

```bash
go run main.go -contract <合约地址> -tokenid <Token ID> [-rpc-url <RPC URL>]
```

### 参数说明

- `-contract`: NFT 合约地址（必需）
- `-tokenid`: Token ID（必需）
- `-rpc-url`: 以太坊 RPC 节点 URL（可选，默认：https://ethereum.publicnode.com）
- `-help`: 显示帮助信息

### 使用示例

#### 1. 查询 IPFS 托管的 NFT（如 Pudgy Penguins）

```bash
go run main.go -contract 0xbD3531da5cf5857E7cFAa92426877b022E612cf8 -tokenid 7641
```

#### 2. 查询链上 NFT（如 OnChain Punks / ChainlinkNN）

```bash
go run main.go -contract 0xe6313d1776e4043d906d5b7221be70cf470f5e87 -tokenid 812
```

#### 3. 使用自定义 RPC 节点

```bash
go run main.go -contract 0xe6313d1776e4043d906d5b7221be70cf470f5e87 -tokenid 812 -rpc-url https://cloudflare-eth.com
```

## 技术原理

### 1. JSON-RPC 调用

程序通过 HTTP POST 请求调用以太坊节点的 JSON-RPC API：

```json
{
  "jsonrpc": "2.0",
  "method": "eth_call",
  "params": [
    {
      "to": "0x合约地址",
      "data": "0x编码的函数调用数据"
    },
    "latest"
  ],
  "id": 1
}
```

### 2. 函数签名编码

`tokenURI(uint256)` 的函数选择器是 `0xc87b56dd`，这是通过对函数签名进行 Keccak-256 哈希并取前 4 字节得到的。

完整的调用数据格式：
- 前 4 字节：函数选择器
- 后 32 字节：Token ID（padding 到 32 字节）

### 3. ABI 编码解码

合约返回的数据是 ABI 编码的动态类型（string）：
- 前 32 字节：数据的偏移量（相对于参数块的起始位置）
- 接下来 32 字节：字符串长度
- 剩余字节：字符串内容（ASCII 编码）

## 支持的 NFT 类型

### 1. IPFS 托管的 NFT
元数据和图片存储在 IPFS 等去中心化存储网络

### 2. 链上 NFT（On-Chain NFT）
元数据和图片完全存储在区块链上，使用 Base64 编码