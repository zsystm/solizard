package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const Binary = "solizard"

var (
	// AbiDir is the directory where all abi files are stored
	// default is $HOME/solizard/abis
	AbiDir      = "abis"
	ZeroAddr    = common.Address{}
	ConfigExist = false
	conf        *Config
)

func init() {
	// get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("failed to get user's home directory (reason: %v)\n", err)
		os.Exit(1)
	}
	AbiDir = homeDir + "/" + Binary + "/" + AbiDir
	if err := DirContainsFiles(AbiDir); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ConfigPath := homeDir + "/" + Binary + "/config.toml"
	conf, err = ReadConfig(ConfigPath)
	if err != nil {
		fmt.Printf("failed to read config file (reason: %v)\n", err)
		ConfigExist = false
	}
	ConfigExist = true
}

func main() {
	// create signal channel for handling program termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	// catch signals and handle program termination
	go func() {
		sig := <-sigs
		fmt.Printf("terminating program... (reason: %v)\n", sig)
		done <- true
	}()
	// run the main program
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("terminating program... (reason: %v)\n", r)
				done <- true
			}
		}()
		err := run()
		if err != nil {
			fmt.Printf("terminating program... (reason: %v)\n", err)
			done <- true
		}
	}()

	<-done
	fmt.Println("terminated")
}

func run() error {
	mAbi, err := loadABIs(AbiDir)
	if err != nil {
		return err
	}

	ctx := new(Ctx)
	if ConfigExist {
		if MustSelectApplyConfig() {
			ctx = NewCtx(conf)
		}
	}

	// start the main loop
	for {
	STEP_SELECT_CONTRACT:
		selectedAbi := MustSelectContractABI(mAbi)
	INPUT_RPC_URL:
		if ctx.EthClient() == nil {
			rpcURL := MustInputRpcUrl()
			client, err := ethclient.Dial(rpcURL)
			if err != nil {
				fmt.Printf("failed to connect to given rpc url: %v, please input valid one\n", err)
				goto INPUT_RPC_URL
			}
			ctx.SetEthClient(client)
		}
	INPUT_CONTRACT_ADDRESS:
		contractAddress := MustInputContractAddress()
		if err := ValidateContractAddress(ctx, contractAddress); err != nil {
			fmt.Printf("Invalid contract address (reason: %v)\n", err)
			goto INPUT_CONTRACT_ADDRESS
		}
	SELECT_METHOD:
		rw := MustSelectReadOrWrite()
		if rw == WriteMethod {
			// input private key
			if ctx.PrivateKey() == nil {
				pk := MustInputPrivateKey()
				ctx.SetPrivateKey(pk)
			}
			// input chainId
			if ctx.ChainId().Sign() == 0 {
				chainID := MustInputChainID()
				ctx.SetChainId(&chainID)
			}
		}
		methodName, method := MustSelectMethod(selectedAbi, rw)
		var input []byte
		input = MustCreateInputDataForMethod(method)
		if rw == ReadMethod {
			callMsg := ethereum.CallMsg{From: ZeroAddr, To: ctx.ContractAddress(), Data: input}
			output, err := ctx.EthClient().CallContract(context.TODO(), callMsg, nil)
			if err != nil {
				return err
			}
			res, err := selectedAbi.Unpack(methodName, output)
			if err != nil {
				return err
			}
			fmt.Printf("output: %v\n", res)
		} else {
			nonce, err := ctx.EthClient().NonceAt(context.TODO(), crypto.PubkeyToAddress(ctx.PrivateKey().PublicKey), nil)
			if err != nil {
				fmt.Printf("failed to get nonce (reason: %v), maybe rpc is not working.\n", err)
				goto INPUT_RPC_URL
			}
			gasPrice, err := ctx.EthClient().SuggestGasPrice(context.TODO())
			if err != nil {
				fmt.Printf("failed to get gas price (reason: %v), maybe rpc is not working.\n", err)
				goto INPUT_RPC_URL
			}
			sufficientGasLimit := uint64(3000000)
			unsignedTx := types.NewTx(&types.LegacyTx{
				To:       ctx.ContractAddress(),
				Nonce:    nonce,
				Value:    common.Big0,
				Gas:      sufficientGasLimit,
				GasPrice: gasPrice,
				Data:     input,
			})
			signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(ctx.ChainId()), ctx.PrivateKey())
			if err = ctx.EthClient().SendTransaction(context.TODO(), signedTx); err != nil {
				fmt.Printf("failed to send transaction (reason: %v), maybe rpc is not working.\n", err)
				return err
			}
			fmt.Printf("transaction sent (txHash %v).\n", signedTx.Hash().Hex())
			// sleep for 5 seconds to wait for transaction to be mined
			fmt.Println("waiting for transaction to be mined... (sleep 5 sec)")
			time.Sleep(5 * time.Second)
			receipt, err := ctx.EthClient().TransactionReceipt(context.TODO(), signedTx.Hash())
			if err != nil {
				fmt.Printf("failed to get transaction receipt (reason: %v).\n", err)
			} else {
				jsonReceipt, _ := receipt.MarshalJSON()
				fmt.Printf("transaction receipt: %s\n", string(jsonReceipt))
			}
		}
		step := MustSelectStep()
		switch step {
		case StepSelectMethod:
			goto SELECT_METHOD
		case StepChangeContract:
			goto STEP_SELECT_CONTRACT
		case StepChangeContractAddress:
			goto INPUT_CONTRACT_ADDRESS
		case StepExit:
			panic("exit")
		}
	}
}
