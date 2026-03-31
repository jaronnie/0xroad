package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// JSONRPCRequest 表示 JSON-RPC 请求
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// JSONRPCResponse 表示 JSON-RPC 响应
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// RPCError 表示 JSON-RPC 错误
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// tokenURI 方法的函数签名和参数编码
// 函数签名: tokenURI(uint256)
// 方法 ID: 0xc87b56dd (tokenURI(uint256) 的前 4 字节)
func encodeTokenURICall(tokenID int) string {
	// tokenURI 方法的签名
	methodID := "c87b56dd"

	// 将 tokenID 编码为 64 字符的十六进制字符串（32 字节）
	tokenIDHex := fmt.Sprintf("%064x", tokenID)

	return "0x" + methodID + tokenIDHex
}

func callContract(rpcURL, contractAddress, data string) (string, error) {
	// 构造 JSON-RPC 请求
	reqBody := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "eth_call",
		Params: []interface{}{
			map[string]string{
				"to":   contractAddress,
				"data": data,
			},
			"latest",
		},
		ID: 1,
	}

	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// 发送 HTTP POST 请求
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	var rpcResp JSONRPCResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	// 解析 JSON 字符串以获取实际的十六进制值
	var hexString string
	if err := json.Unmarshal(rpcResp.Result, &hexString); err != nil {
		return "", fmt.Errorf("failed to unmarshal hex string: %v", err)
	}

	return hexString, nil
}

func decodeString(result string) (string, error) {
	// 移除 0x 前缀
	if len(result) < 2 || result[:2] != "0x" {
		return "", fmt.Errorf("invalid hex string")
	}

	result = result[2:]

	// 跳过前 64 个字符（偏移量）
	if len(result) < 64 {
		return "", fmt.Errorf("result too short for offset")
	}

	// 解码字符串长度（接下来的 64 个字符）
	if len(result) < 128 {
		return "", fmt.Errorf("result too short for length")
	}
	length, err := strconv.ParseInt(result[64:128], 16, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse length: %v", err)
	}

	// 计算字符串内容的位置（从偏移量开始）
	offset, err := strconv.ParseInt(result[0:64], 16, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse offset: %v", err)
	}

	stringPos := 64 + int(offset)*2
	if stringPos+int(length)*2 > len(result) {
		return "", fmt.Errorf("string data too short: need %d, have %d", stringPos+int(length)*2, len(result))
	}

	// 解码字符串内容
	stringData := result[stringPos : stringPos+int(length)*2]

	// 将十六进制转换为 ASCII 字符串
	var str string
	for i := 0; i < len(stringData); i += 2 {
		hexChar := stringData[i : i+2]
		charCode, err := strconv.ParseInt(hexChar, 16, 32)
		if err != nil {
			return "", fmt.Errorf("failed to parse character: %v", err)
		}
		str += string(rune(charCode))
	}

	return str, nil
}

// decodeBase64DataURI 解码 base64 编码的 data URI
// 格式: data:application/json;base64,eyJ...
func decodeBase64DataURI(dataURI string) (string, bool) {
	// 检查是否是 data URI 格式
	if !strings.HasPrefix(dataURI, "data:") {
		return dataURI, false
	}

	// 提取 MIME 类型和编码部分
	parts := strings.SplitN(dataURI, ";", 2)
	if len(parts) != 2 {
		return dataURI, false
	}

	// 检查是否是 base64 编码
	encodingAndData := parts[1]
	if !strings.HasPrefix(encodingAndData, "base64,") {
		return dataURI, false
	}

	// 提取 base64 数据
	base64Data := strings.TrimPrefix(encodingAndData, "base64,")

	// 解码 base64
	decoded, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return dataURI, false
	}

	return string(decoded), true
}

// saveBase64Image 保存 base64 编码的图片到本地文件
func saveBase64Image(dataURI, tokenID string) error {
	// 检查是否是 data URI 格式
	if !strings.HasPrefix(dataURI, "data:image/") {
		return fmt.Errorf("不是图片 data URI")
	}

	// 提取 MIME 类型和编码
	// 格式: data:image/svg+xml;base64,PHN2Zy...
	// 或: data:image/png;base64,iVBORw...

	parts := strings.SplitN(dataURI, ";", 2)
	if len(parts) != 2 {
		return fmt.Errorf("无效的 data URI 格式")
	}

	// 提取图片类型 (svg+xml, png, jpeg, gif 等)
	mimeParts := strings.SplitN(parts[0], ":", 2)
	if len(mimeParts) != 2 {
		return fmt.Errorf("无效的 MIME 类型")
	}
	mimeType := mimeParts[1]

	// 检查编码方式
	encodingAndData := parts[1]
	if !strings.HasPrefix(encodingAndData, "base64,") {
		return fmt.Errorf("仅支持 base64 编码")
	}

	// 提取 base64 数据
	base64Data := strings.TrimPrefix(encodingAndData, "base64,")

	// 解码 base64
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("base64 解码失败: %v", err)
	}

	// 根据 MIME 类型确定文件扩展名
	var ext string
	switch {
	case strings.HasPrefix(mimeType, "image/svg"):
		ext = ".svg"
	case strings.HasPrefix(mimeType, "image/png"):
		ext = ".png"
	case strings.HasPrefix(mimeType, "image/jpeg"):
		ext = ".jpg"
	case strings.HasPrefix(mimeType, "image/gif"):
		ext = ".gif"
	case strings.HasPrefix(mimeType, "image/webp"):
		ext = ".webp"
	default:
		ext = ".bin"
	}

	// 生成文件名
	filename := fmt.Sprintf("token_%s%s", tokenID, ext)

	// 保存到文件
	if err := os.WriteFile(filename, imageData, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Printf("✓ 图片已保存到: %s (%d bytes)\n", filename, len(imageData))
	return nil
}

func main() {
	// 定义命令行参数
	rpcURL := flag.String("rpc-url", "https://ethereum.publicnode.com", "以太坊 RPC 节点 URL")
	contractAddr := flag.String("contract", "", "NFT 合约地址（必需）")
	tokenIDStr := flag.String("tokenid", "", "Token ID（必需）")
	showHelp := flag.Bool("help", false, "显示帮助信息")

	flag.Parse()

	// 显示帮助信息
	if *showHelp || flag.NArg() > 0 || *contractAddr == "" || *tokenIDStr == "" {
		fmt.Println("ERC721 Token URI Fetcher - 获取 NFT 元数据并保存图片")
		fmt.Println("\n使用方法:")
		fmt.Println("  go run main.go -contract <合约地址> -tokenid <Token ID> [-rpc-url <RPC URL>]")
		fmt.Println("\n示例:")
		fmt.Println("  go run main.go -contract 0xe6313d1776e4043d906d5b7221be70cf470f5e87 -tokenid 812")
		fmt.Println("  go run main.go -contract 0xe6313d1776e4043d906d5b7221be70cf470f5e87 -tokenid 812 -rpc-url https://cloudflare-eth.com")
		fmt.Println("\n参数:")
		flag.PrintDefaults()
		fmt.Println("\n公共 RPC 节点:")
		fmt.Println("  - https://ethereum.publicnode.com")
		fmt.Println("  - https://cloudflare-eth.com")
		fmt.Println("  - https://rpc.ankr.com/eth")
		os.Exit(0)
	}

	// 处理合约地址
	contract := *contractAddr
	if !strings.HasPrefix(contract, "0x") {
		contract = "0x" + contract
	}

	// 解析 Token ID
	tokenIDInt64, err := strconv.ParseInt(*tokenIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid Token ID: %v", err)
	}
	tokenID := int(tokenIDInt64)

	fmt.Println("==========================================")
	fmt.Println("ERC721 Token URI Fetcher")
	fmt.Println("==========================================")
	fmt.Printf("Contract Address: %s\n", contract)
	fmt.Printf("Token ID: %d\n", tokenID)
	fmt.Printf("RPC Endpoint: %s\n\n", *rpcURL)

	// 编码函数调用
	callData := encodeTokenURICall(tokenID)

	// 调用合约
	result, err := callContract(*rpcURL, contract, callData)
	if err != nil {
		log.Fatalf("Failed to call contract: %v", err)
	}

	// 解码结果
	tokenURI, err := decodeString(result)
	if err != nil {
		log.Fatalf("Failed to decode result: %v", err)
	}

	// 打印结果
	fmt.Println("==========================================")
	fmt.Printf("Token URI: %s\n", tokenURI)
	fmt.Println("==========================================")

	// 检查是否是 base64 编码的 data URI（链上 NFT）
	if decodedData, isBase64 := decodeBase64DataURI(tokenURI); isBase64 {
		fmt.Println("\n✓ 检测到 Base64 编码的链上元数据！")
		fmt.Println("------------------------------------------")

		// 尝试解析 JSON
		var metadata map[string]any
		if err := json.Unmarshal([]byte(decodedData), &metadata); err == nil {
			fmt.Println("元数据内容:")
			prettyJSON, _ := json.MarshalIndent(metadata, "", "  ")
			fmt.Println(string(prettyJSON))

			// 尝试保存图片
			fmt.Println("\n------------------------------------------")
			fmt.Println("检查是否包含图片数据...")

			// 优先查找 image_data 字段（完全链上）
			if imageData, ok := metadata["image_data"].(string); ok {
				fmt.Println("✓ 发现 image_data 字段（链上图片）")
				if err := saveBase64Image(imageData, fmt.Sprintf("%d", tokenID)); err != nil {
					fmt.Printf("✗ 保存图片失败: %v\n", err)
				}
			} else if imageURL, ok := metadata["image"].(string); ok {
				// 检查 image 字段是否也是 base64 data URI
				if strings.HasPrefix(imageURL, "data:image/") {
					fmt.Println("✓ 发现 image 字段（base64 编码）")
					if err := saveBase64Image(imageURL, fmt.Sprintf("%d", tokenID)); err != nil {
						fmt.Printf("✗ 保存图片失败: %v\n", err)
					}
				} else {
					fmt.Printf("ℹ image 字段指向外部 URL: %s\n", imageURL)
				}
			} else {
				fmt.Println("ℹ 元数据中没有找到图片数据")
			}

		} else {
			fmt.Println("解码后的数据:")
			fmt.Println(decodedData)
		}

		return
	}

	// 尝试获取元数据
	// 将 ipfs:// 转换为 https://ipfs.io/ipfs/
	var metadataURL string
	if len(tokenURI) > 7 && tokenURI[:7] == "ipfs://" {
		metadataURL = "https://ipfs.io/ipfs/" + tokenURI[7:]
	} else {
		metadataURL = tokenURI
	}

	fmt.Printf("\nFetching metadata from: %s\n", metadataURL)
	resp, err := http.Get(metadataURL)
	if err != nil {
		fmt.Printf("Failed to fetch metadata: %v\n", err)
		fmt.Println("\n提示：IPFS 网关可能无法访问，您可以手动访问该 URL 或使用其他 IPFS 网关")
		return
	}
	defer resp.Body.Close()

	fmt.Printf("HTTP Status Code: %d\n", resp.StatusCode)

	if resp.StatusCode == 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Failed to read response body: %v\n", err)
			return
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal(body, &metadata); err == nil {
			fmt.Println("\nMetadata:")
			prettyJSON, _ := json.MarshalIndent(metadata, "", "  ")
			fmt.Println(string(prettyJSON))

			// 尝试下载 IPFS/HTTP 图片
			fmt.Println("\n------------------------------------------")
			fmt.Println("检查是否包含图片数据...")

			if imageURL, ok := metadata["image"].(string); ok && imageURL != "" {
				fmt.Printf("✓ 发现 image 字段: %s\n", imageURL)

				// 处理不同类型的图片 URL
				var imageDownloadURL string
				var filename string

				if strings.HasPrefix(imageURL, "ipfs://") {
					// IPFS URL
					imageDownloadURL = "https://ipfs.io/ipfs/" + imageURL[7:]
					// 从 URL 提取文件名
					parts := strings.Split(imageURL[7:], "/")
					lastPart := parts[len(parts)-1]
					filename = fmt.Sprintf("token_%d_%s", tokenID, lastPart)
					// 如果没有扩展名，根据内容类型判断
					if !strings.Contains(lastPart, ".") {
						filename = fmt.Sprintf("token_%d.png", tokenID)
					}
				} else if strings.HasPrefix(imageURL, "http://") || strings.HasPrefix(imageURL, "https://") {
					// HTTP/HTTPS URL
					imageDownloadURL = imageURL
					// 从 URL 提取文件名
					parts := strings.Split(imageURL, "/")
					lastPart := parts[len(parts)-1]
					if strings.Contains(lastPart, ".") {
						filename = fmt.Sprintf("token_%d_%s", tokenID, lastPart)
					} else {
						filename = fmt.Sprintf("token_%d.png", tokenID)
					}
				} else if strings.HasPrefix(imageURL, "data:image/") {
					// Base64 编码的图片
					if err := saveBase64Image(imageURL, fmt.Sprintf("%d", tokenID)); err != nil {
						fmt.Printf("✗ 保存图片失败: %v\n", err)
					}
					return
				} else {
					fmt.Printf("ℹ 不支持的图片 URL 格式: %s\n", imageURL)
					return
				}

				// 下载图片
				fmt.Printf("正在下载图片: %s\n", imageDownloadURL)
				resp, err := http.Get(imageDownloadURL)
				if err != nil {
					fmt.Printf("✗ 下载图片失败: %v\n", err)
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != 200 {
					fmt.Printf("✗ 下载图片失败，HTTP 状态码: %d\n", resp.StatusCode)
					return
				}

				// 读取图片数据
				imageData, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("✗ 读取图片数据失败: %v\n", err)
					return
				}

				// 根据内容类型确定文件扩展名
				contentType := resp.Header.Get("Content-Type")
				if contentType != "" {
					switch {
					case strings.Contains(contentType, "image/svg"):
						filename = fmt.Sprintf("token_%d.svg", tokenID)
					case strings.Contains(contentType, "image/png"):
						filename = fmt.Sprintf("token_%d.png", tokenID)
					case strings.Contains(contentType, "image/jpeg"):
						filename = fmt.Sprintf("token_%d.jpg", tokenID)
					case strings.Contains(contentType, "image/gif"):
						filename = fmt.Sprintf("token_%d.gif", tokenID)
					case strings.Contains(contentType, "image/webp"):
						filename = fmt.Sprintf("token_%d.webp", tokenID)
					}
				}

				// 保存图片
				if err := os.WriteFile(filename, imageData, 0644); err != nil {
					fmt.Printf("✗ 保存图片失败: %v\n", err)
					return
				}

				fmt.Printf("✓ 图片已保存到: %s (%d bytes)\n", filename, len(imageData))
			} else {
				fmt.Println("ℹ 元数据中没有找到 image 字段")
			}
		} else {
			fmt.Println("\nRaw Response:")
			fmt.Println(string(body))
		}
	}
}
