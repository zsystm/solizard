package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const Binary = "solizard"

var (
	// AbiDir is the directory where all abi files are stored
	// default is $HOME/solizard/abis
	AbiDir        = "abis"
	DefaultRPCURL = "http://localhost:8545"
	ZeroAddr      = common.Address{}
)

func init() {
	// get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	AbiDir = homeDir + "/" + Binary + "/" + AbiDir
}

func main() {
	// create signal channel for handling program termination
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)

	// catch signals and handle program termination
	go func() {
		sig := <-sigs
		fmt.Printf("Terminating program... (reason: %v)\n", sig)
		done <- true
	}()
	// run the main program
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("Terminating program... (reason: %v)\n", r)
				done <- true
			}
		}()
		run()
	}()

	<-done
	fmt.Println("Bye")
}

func run() {
	// create eth client
	var client *ethclient.Client
	var err error

	// read all abi files in ABI_DIR
	files, err := os.ReadDir(AbiDir)
	if err != nil {
		panic(err)
	}
	// parse all abi files
	mAbi := make(map[string]abi.ABI)
	for _, f := range files {
		abiFilepath := AbiDir + "/" + f.Name()
		contractABI, err := readABIFile(abiFilepath)
		if err != nil {
			panic(err)
		}
		mAbi[f.Name()] = contractABI
	}

	var pk *ecdsa.PrivateKey
	var chainID big.Int
	var rpcURL string
	// start the main loop
	for {
	STEP_SELECT_CONTRACT:
		selectedAbi := MustSelectContractABI(mAbi)
	INPUT_RPC_URL:
		if rpcURL == "" {
			rpcURL = MustInputRpcUrl()
			client, err = ethclient.Dial(rpcURL)
			if err != nil {
				fmt.Printf("Failed to connect to given rpc url: %v, please input valid one\n", err)
				goto INPUT_RPC_URL
			}
		}
	INPUT_CONTRACT_ADDRESS:
		contractAddress := MustInputContractAddress()
		cAddr := common.HexToAddress(contractAddress)
		code, err := client.CodeAt(context.TODO(), cAddr, nil)
		if err != nil {
			fmt.Printf("Failed to get code at given contract address (reason: %v), maybe non existing contract\n", err)
		}
		if len(code) == 0 {
			fmt.Printf("Given contract address is not a contract address, no bytecode in chain\n")
			goto INPUT_CONTRACT_ADDRESS
		}
	SELECT_METHOD:
		rw := MustSelectReadOrWrite()
	INPUT_PRIVATE_KEY:
		if rw == WriteMethod {
			// input private key
			if pk == nil {
				pk, err = InputPrivateKey()
				if err != nil {
					fmt.Printf("Invalid private key (reason: %v), please input valid one\n", err)
					goto INPUT_PRIVATE_KEY
				}
			}
			// input chainID
			if chainID.Sign() == 0 {
				chainID = MustInputChainID()
			}
		}
		methodName, method := MustSelectMethod(selectedAbi, rw)
		var input []byte
		input = MustCreateInputDataForMethod(method)
		if rw == ReadMethod {
			callMsg := ethereum.CallMsg{From: ZeroAddr, To: &cAddr, Data: input}
			output, err := client.CallContract(context.TODO(), callMsg, nil)
			if err != nil {
				panic(err)
			}
			res, err := selectedAbi.Unpack(methodName, output)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Output: %v\n", res)
		} else {
			nonce, err := client.PendingNonceAt(context.TODO(), cAddr)
			if err != nil {
				fmt.Printf("Failed to get nonce (reason: %v), maybe rpc is not working.\n", err)
				goto INPUT_RPC_URL
			}
			gasPrice, err := client.SuggestGasPrice(context.TODO())
			if err != nil {
				fmt.Printf("Failed to get gas price (reason: %v), maybe rpc is not working.\n", err)
				goto INPUT_RPC_URL
			}
			sufficientGasLimit := uint64(3000000)
			unsignedTx := types.NewTx(&types.LegacyTx{
				To:       &cAddr,
				Nonce:    nonce,
				Value:    common.Big0,
				Gas:      sufficientGasLimit,
				GasPrice: gasPrice,
			})
			signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(&chainID), pk)
			err = client.SendTransaction(context.TODO(), signedTx)
			if err != nil {
				fmt.Printf("Failed to send transaction (reason: %v), maybe rpc is not working.\n", err)
				goto INPUT_RPC_URL
			}
			fmt.Printf("Transaction sent (txHash %v).\n", signedTx.Hash().Hex())
			// sleep for 5 seconds to wait for transaction to be mined
			time.Sleep(5 * time.Second)
			receipt, err := client.TransactionReceipt(context.TODO(), signedTx.Hash())
			if err != nil {
				fmt.Printf("Failed to get transaction receipt (reason: %v).\n", err)
			}
			fmt.Printf("Transaction receipt: %v\n", receipt)
		}
		step := MustSelectStep()
		switch step {
		case StepSelectContract:
			goto STEP_SELECT_CONTRACT
		case StepInputContractAddress:
			goto INPUT_CONTRACT_ADDRESS
		case StepSelectMethod:
			goto SELECT_METHOD
		case StepExit:
			panic("exit")
		}
	}
}
