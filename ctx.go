package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Ctx struct {
	ethCli          *ethclient.Client
	pk              *ecdsa.PrivateKey
	chainId         big.Int
	contractAddress *common.Address
}

// NewCtx creates a new context with the given config.
// If the config is invalid, it will prompt the user to input manually.
func NewCtx(conf *Config) *Ctx {
	ctx := &Ctx{}
	var err error

	errMsg := "failed to apply config file, please input manually when you see the prompt"
	if ctx.pk, err = crypto.HexToECDSA(conf.PrivateKey); err != nil {
		fmt.Printf("%s (reason: invalid private key, err: %v)\n", errMsg, err)
		ctx.pk = nil
	}
	// check url validity
	if err = ValidateRpcURL(conf.RpcURL); err != nil {
		fmt.Printf("%s (reason: invalid rpc url, err: %v)\n", errMsg, err)
	} else {
		rpcURL := conf.RpcURL
		if ctx.ethCli, err = ethclient.Dial(rpcURL); err != nil {
			fmt.Printf("%s (reason: failed to connect to given rpc url, err: %v)\n", errMsg, err)
			ctx.ethCli = nil
		}
	}
	ctx.chainId.SetUint64(conf.ChainId)
	if ctx.ethCli != nil {
		// query chain id from the client
		chainID, err := ctx.ethCli.ChainID(context.TODO())
		if err == nil {
			// if conf.ChainId and chainId are different, print a warning
			if chainID.Cmp(&ctx.chainId) != 0 {
				fmt.Printf("WARNING: chain id from config file (%d) and chain id from the client (%d) are different\n", ctx.chainId.Uint64(), chainID.Uint64())
			}
		}
	}
	return ctx
}

// setters
func (c *Ctx) SetEthClient(cli *ethclient.Client) {
	c.ethCli = cli
}

func (c *Ctx) SetPrivateKey(pk *ecdsa.PrivateKey) {
	c.pk = pk
}

func (c *Ctx) SetChainId(chainId *big.Int) {
	c.chainId.Set(chainId)
}

func (c *Ctx) SetContractAddress(addr *common.Address) {
	c.contractAddress = addr
}

// getters
func (c *Ctx) EthClient() *ethclient.Client {
	return c.ethCli
}

func (c *Ctx) PrivateKey() *ecdsa.PrivateKey {
	return c.pk
}

func (c *Ctx) ChainId() *big.Int {
	return &c.chainId
}

func (c *Ctx) ContractAddress() *common.Address {
	return c.contractAddress
}
