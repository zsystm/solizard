package ctx

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net/url"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/zsystm/solizard/internal/config"
	"github.com/zsystm/solizard/lib"

	"github.com/fatih/color"
)

type Context struct {
	ethCli          *ethclient.Client
	rpcURL          string
	pk              *ecdsa.PrivateKey
	chainId         big.Int
	contractAddress *common.Address
}

// NewCtx creates a new ctx with the given config.
// If the config is invalid, it will prompt the user to input manually.
func NewCtx(conf *config.Config) *Context {
	ctx := &Context{}
	var err error

	errMsg := "failed to apply config file, please input manually when you see the prompt"
	if ctx.pk, err = crypto.HexToECDSA(conf.PrivateKey); err != nil {
		fmt.Printf("%s (reason: invalid private key, err: %v)\n", errMsg, err)
		ctx.pk = nil
	}
	// check url validity
	if _, err := url.ParseRequestURI(conf.RpcURL); err != nil {
		fmt.Printf("%s (reason: invalid rpc url, err: %v)\n", errMsg, err)
	} else {
		rpcURL := conf.RpcURL
		if ctx.ethCli, err = ethclient.Dial(rpcURL); err != nil {
			fmt.Printf("%s (reason: failed to connect to given rpc url, err: %v)\n", errMsg, err)
			ctx.ethCli = nil
		}
		ctx.rpcURL = rpcURL
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
func (c *Context) SetEthClient(cli *ethclient.Client) {
	c.ethCli = cli
}

func (c *Context) SetPrivateKey(pk *ecdsa.PrivateKey) {
	c.pk = pk
}

func (c *Context) SetChainId(chainId *big.Int) {
	c.chainId.Set(chainId)
}

func (c *Context) SetContractAddress(addr *common.Address) {
	c.contractAddress = addr
}

// getters
func (c *Context) EthClient() *ethclient.Client {
	return c.ethCli
}

func (c *Context) PrivateKey() *ecdsa.PrivateKey {
	return c.pk
}

func (c *Context) ChainId() *big.Int {
	return &c.chainId
}

func (c *Context) ContractAddress() *common.Address {
	return c.contractAddress
}

func PrintContext(ctx *Context, chainInfos []*lib.ChainInfo) {
	title := color.New(color.FgHiYellow, color.Bold).SprintFunc()
	key := color.New(color.FgCyan, color.Bold).SprintFunc()
	val := color.New(color.FgWhite).SprintFunc()

	chainId := ctx.ChainId().Uint64()
	chainInfo, _ := lib.GetChainInfoByID(chainInfos, chainId)

	const contentWidth = 35

	fmt.Println("╔═════════════════════════════════════╗")
	fmt.Printf("║         %s       ║\n", title("Current Configuration"))
	fmt.Println("╟─────────────────────────────────────╢")
	fmt.Printf("║ %s ║\n", lib.PadRightAnsiAware(fmt.Sprintf("%s: %s", key("RPC URL"), val(ctx.rpcURL)), contentWidth))
	fmt.Printf("║ %s ║\n", lib.PadRightAnsiAware(fmt.Sprintf("%s: %d", key("Chain ID"), ctx.ChainId()), contentWidth))
	if chainInfo != nil {
		fmt.Printf("║ %s ║\n", lib.PadRightAnsiAware(fmt.Sprintf("%s: %s", key("Chain Name"), val(chainInfo.Name)), contentWidth))
		fmt.Printf("║ %s ║\n", lib.PadRightAnsiAware(fmt.Sprintf("%s: %s (%s)", key("Native Currency"), val(chainInfo.NativeCurrency.Name), val(chainInfo.NativeCurrency.Symbol)), contentWidth))
		fmt.Printf("║ %s ║\n", lib.PadRightAnsiAware(fmt.Sprintf("%s: %d", key("Decimals"), chainInfo.NativeCurrency.Decimals), contentWidth))
	}
	fmt.Println("╚═════════════════════════════════════╝")
	fmt.Println("")
}
