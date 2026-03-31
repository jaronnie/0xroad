# ERC721 Token URI Fetcher

一个用 Go 语言编写的 ERC721 合约调用工具，用于获取 NFT 的 Token URI 和元数据，支持自动保存链上 NFT 图片。

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

### 输出示例

#### 链上 NFT（自动保存图片）

```
==========================================
ERC721 Token URI Fetcher
==========================================
Contract Address: 0xe6313d1776e4043d906d5b7221be70cf470f5e87
Token ID: 812
RPC Endpoint: https://ethereum.publicnode.com

==========================================
Token URI: data:application/json;base64,eyJuYW1lIjoiQ2hhaW...
==========================================

✓ 检测到 Base64 编码的链上元数据！
------------------------------------------
元数据内容:
{
  "name": "ChainlinkNN #812",
  "description": "...",
  "image_data": "data:image/svg+xml;base64,PHN2Zy..."
}

------------------------------------------
检查是否包含图片数据...
✓ 发现 image_data 字段（链上图片）
✓ 图片已保存到: token_812.svg (1234 bytes)
```

#### IPFS 托管的 NFT

```
==========================================
ERC721 Token URI Fetcher
==========================================
Contract Address: 0xbD3531da5cf5857E7cFAa92426877b022E612cf8
Token ID: 7641
RPC Endpoint: https://ethereum.publicnode.com

==========================================
Token URI: ipfs://bafybeibc5sgo2plmjkq2tzmhrn54bk3crhnc23zd2msg4ea7a4pxrkgfna/7641
==========================================

Fetching metadata from: https://ipfs.io/ipfs/bafybeibc5sgo2plmjkq2tzmhrn54bk3crhnc23zd2msg4ea7a4pxrkgfna/7641
HTTP Status Code: 200

Metadata:
{
  "name": "Pudgy Penguin #7641",
  "description": "...",
  "image": "ipfs://bafybe..."
}

------------------------------------------
检查是否包含图片数据...
✓ 发现 image 字段: ipfs://bafybe...
正在下载图片: https://ipfs.io/ipfs/bafybe...
✓ 图片已保存到: token_7641.png (12345 bytes)
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

### 4. IPFS 网关

程序会自动将 `ipfs://` 协议转换为 HTTP 网关 URL：
- `ipfs://CID/path` → `https://ipfs.io/ipfs/CID/path`

### 5. 链上 NFT 支持（Base64 Data URI）

对于完全在链上的 NFT，`tokenURI` 返回的是 Base64 编码的 data URI：

```
data:application/json;base64,eyJnameIjoiT2ZmL...
```

程序会：
1. 自动检测 Base64 编码的元数据
2. 解码并解析 JSON
3. 检查 `image_data` 或 `image` 字段
4. 如果是 Base64 编码的图片，自动保存到本地

支持保存的图片格式：
- SVG (`image/svg+xml`)
- PNG (`image/png`)
- JPEG (`image/jpeg`)
- GIF (`image/gif`)
- WebP (`image/webp`)

## 公共 RPC 端点

如果当前使用的 RPC 端点无法访问，可以尝试以下公共节点：

- Cloudflare: `https://cloudflare-eth.com`
- Infura (需要注册): `https://mainnet.infura.io/v3/YOUR_PROJECT_ID`
- Alchemy (需要注册): `https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY`
- Ankr: `https://rpc.ankr.com/eth`
- PublicNode: `https://ethereum.publicnode.com`

## 依赖

仅使用 Go 标准库：
- `bytes` - 字节缓冲操作
- `encoding/base64` - Base64 编解码
- `encoding/json` - JSON 编解码
- `flag` - 命令行参数解析
- `fmt` - 格式化输出
- `io` - I/O 操作
- `log` - 日志输出
- `net/http` - HTTP 客户端
- `os` - 文件系统操作
- `strconv` - 字符串转换
- `strings` - 字符串操作

## 支持的 NFT 类型

### 1. IPFS 托管的 NFT
元数据和图片存储在 IPFS 等去中心化存储网络，例如：
- **Pudgy Penguins**: 合约 `0xbD3531da5cf5857E7cFAa92426877b022E612cf8`
- **Bored Ape Yacht Club**: 合约 `0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D`

### 2. 链上 NFT（On-Chain NFT）
元数据和图片完全存储在区块链上，使用 Base64 编码：
- **ChainlinkNN**: 合约 `0xe6313d1776e4043d906d5b7221be70cf470f5e87`
- **OnChain Punks / 其他完全链上 NFT 项目**

程序会自动检测 NFT 类型：
- **链上 NFT**：自动解码元数据并保存图片到本地（`token_<id>.<ext>`）
- **IPFS NFT**：通过 IPFS 网关获取元数据，自动下载并保存图片（`token_<id>.<ext>` 或 `token_<id>_<filename>.<ext>`）
- **HTTP NFT**：获取元数据并下载图片（`token_<id>_<filename>.<ext>`）

支持的图片下载源：
- IPFS URL（`ipfs://...`）
- HTTP/HTTPS URL
- Base64 编码的 data URI

## 故障排查

### 参数错误或需要帮助

运行以下命令查看完整的使用说明：
```bash
go run main.go -help
```

### RPC 连接失败

- 更换其他公共 RPC 端点（使用 `-rpc-url` 参数）
- 检查网络连接
- 尝试使用 VPN

常见错误信息：
- `Failed to send request`: RPC 节点无法访问
- `RPC error`: 合约调用失败（可能是合约地址或 Token ID 错误）

### IPFS 网关超时

程序会提示网络超时，但已经成功获取到 Token URI。您可以：
1. 手动访问提示的 IPFS 网关 URL
2. 使用其他 IPFS 网关（如 `https://gateway.pinata.cloud/ipfs/CID`）
3. 使用 IPFS 桌面客户端或浏览器插件

### 图片保存失败

如果图片保存失败，可能的原因：
- 元数据中的图片不是 Base64 编码（程序会显示外部 URL）
- Base64 数据格式错误
- 磁盘空间不足或权限问题

## 项目结构

```
erc721-fetcher/
├── main.go       # 主程序文件
├── go.mod        # Go 模块文件
├── README.md     # 项目文档
└── token_*.svg   # 保存的 NFT 图片（运行后生成）
```

## 许可证

MIT License
