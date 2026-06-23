package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	client *ethclient.Client
}

func NewClient(address string) (*Client, error) {
	client, err := ethclient.Dial(address)
	if err != nil {
		return nil, err
	}
	return &Client{client}, nil
}

func (c *Client) GetBalance(address string) (*big.Int, error) {
	balance, err := c.client.BalanceAt(context.Background(), common.HexToAddress(address), nil)
	if err != nil {
		return nil, err
	}
	return balance, nil
}

func main() {
	client, err := NewClient("https://jaronnie.eth.blockrazor.xyz")
	if err != nil {
		log.Fatal(err)
	}
	balance, err := client.GetBalance("0xd454F98C9a0fEC19d653492A7a140054Ff03a250")
	if err != nil {
		log.Fatal(err)
	}

	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	fmt.Println(ethValue)

	//wsClient, err := NewClient("ws://aion-tokyo.blockrazor.me/ws")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//wsClient.Subscribe()
}

func (c *Client) Subscribe() {
	headers := make(chan *types.Header)
	sub, err := c.client.SubscribeNewHead(context.Background(), headers)
	if err != nil {
		panic(err)
	}

	for {
		select {
		case err := <-sub.Err():
			panic(err)
		case header := <-headers:
			fmt.Println(header.Number, header.Hash())
		}
	}
}
