package solizard

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zsystm/solizard/internal/abi"
	"github.com/zsystm/solizard/internal/config"
	"github.com/zsystm/solizard/internal/ctx"
	"github.com/zsystm/solizard/internal/prompt"
	"github.com/zsystm/solizard/internal/step"
	"github.com/zsystm/solizard/internal/validation"
)

//go:embed embeds/*
var embeddedFiles embed.FS

const SolizardDir = ".solizard"

var (
	// AbiDir is the directory where all abi files are stored
	// default is $HOME/solizard/abis
	AbiDir      = "abis"
	ZeroAddr    = common.Address{}
	ConfigExist = false
	conf        *config.Config
)

func init() {
	// get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("failed to get user's home directory (reason: %v)\n", err)
		os.Exit(1)
	}
	dir := homeDir + "/" + SolizardDir
	AbiDir = dir + "/" + AbiDir

	// initialize .solizard default if not exists
	if _, err := os.Stat(AbiDir); os.IsNotExist(err) {
		// create solizard and abi directory if not exists
		if err := os.MkdirAll(AbiDir, 0755); err != nil {
			fmt.Printf("failed to create abi directory (reason: %v)\n", err)
			os.Exit(1)
		}
		// copy embeds to directory
		if err = fs.WalkDir(embeddedFiles, "embeds", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			data, err := embeddedFiles.ReadFile(path)
			if err != nil {
				return err
			}
			// if it is a yaml file, write it to abi directory
			if d.Type().IsRegular() && d.Name()[len(d.Name())-4:] == ".abi" {
				if err := os.WriteFile(AbiDir+"/"+d.Name(), data, 0644); err != nil {
					return err
				}
			}
			if d.Type().IsRegular() && d.Name() == "config.toml" {
				if err := os.WriteFile(homeDir+"/"+SolizardDir+"/config.toml", data, 0644); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			fmt.Printf("failed to walk embedded files (reason: %v)\n", err)
		}
		// copy embeds to abi directory
	}

	// check if abi directory contains files
	if err := validation.DirContainsFiles(AbiDir); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	ConfigPath := dir + "/" + "config.toml"
	conf, err = config.ReadConfig(ConfigPath)
	if err != nil {
		fmt.Printf("failed to read config file (reason: %v)\n", err)
		ConfigExist = false
	}
	ConfigExist = true
}

func Run() error {
	mAbi, err := abi.LoadABIs(AbiDir)
	if err != nil {
		return err
	}

	sctx := new(ctx.Context)
	if ConfigExist {
		if prompt.MustSelectApplyConfig() {
			sctx = ctx.NewCtx(conf)
		}
	}

	// start the main loop
	for {
	STEP_SELECT_CONTRACT:
		selectedAbi := prompt.MustSelectContractABI(mAbi)
	INPUT_RPC_URL:
		if sctx.EthClient() == nil {
			rpcURL := prompt.MustInputRpcUrl()
			client, err := ethclient.Dial(rpcURL)
			if err != nil {
				fmt.Printf("failed to connect to given rpc url: %v, please input valid one\n", err)
				goto INPUT_RPC_URL
			}
			sctx.SetEthClient(client)
		}
	INPUT_CONTRACT_ADDRESS:
		contractAddress := prompt.MustInputContractAddress()
		if err := validation.ValidateContractAddress(sctx, contractAddress); err != nil {
			fmt.Printf("Invalid contract address (reason: %v)\n", err)
			goto INPUT_CONTRACT_ADDRESS
		}
	SELECT_METHOD:
		rw := prompt.MustSelectReadOrWrite()
		if rw == abi.WriteMethod {
			// input private key
			if sctx.PrivateKey() == nil {
				pk := prompt.MustInputPrivateKey()
				sctx.SetPrivateKey(pk)
			}
			// input chainId
			if sctx.ChainId().Sign() == 0 {
				chainID := prompt.MustInputChainID()
				sctx.SetChainId(&chainID)
			}
		}
		methodName, method := prompt.MustSelectMethod(selectedAbi, rw)
		input := prompt.MustCreateInputDataForMethod(method)
		if rw == abi.ReadMethod {
			callMsg := ethereum.CallMsg{From: ZeroAddr, To: sctx.ContractAddress(), Data: input}
			output, err := sctx.EthClient().CallContract(context.TODO(), callMsg, nil)
			if err != nil {
				return err
			}
			res, err := selectedAbi.Unpack(methodName, output)
			if err != nil {
				return err
			}
			fmt.Printf("output: %v\n", res)
		} else {
			nonce, err := sctx.EthClient().NonceAt(context.TODO(), crypto.PubkeyToAddress(sctx.PrivateKey().PublicKey), nil)
			if err != nil {
				fmt.Printf("failed to get nonce (reason: %v), maybe rpc is not working.\n", err)
				goto INPUT_RPC_URL
			}
			gasPrice, err := sctx.EthClient().SuggestGasPrice(context.TODO())
			if err != nil {
				fmt.Printf("failed to get gas price (reason: %v), maybe rpc is not working.\n", err)
				goto INPUT_RPC_URL
			}
			// TODO: Change to EthClient().EstimateGas() call.
			sufficientGasLimit := uint64(3000000)
			unsignedTx := types.NewTx(&types.LegacyTx{
				To:       sctx.ContractAddress(),
				Nonce:    nonce,
				Value:    common.Big0,
				Gas:      sufficientGasLimit,
				GasPrice: gasPrice,
				Data:     input,
			})
			signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(sctx.ChainId()), sctx.PrivateKey())
			if err != nil {
				fmt.Printf("failed to SignTx (reason: %v)\n", err)
				return err
			}
			if err = sctx.EthClient().SendTransaction(context.TODO(), signedTx); err != nil {
				fmt.Printf("failed to send transaction (reason: %v), maybe rpc is not working.\n", err)
				return err
			}
			fmt.Printf("transaction sent (txHash %v).\n", signedTx.Hash().Hex())
			// sleep for x seconds to wait for transaction to be mined
			waitTime, err := time.ParseDuration(conf.WaitTime)
			fmt.Printf("waiting for transaction to be mined... (sleep %s\n", waitTime.String())
			time.Sleep(waitTime)
			receipt, err := sctx.EthClient().TransactionReceipt(context.TODO(), signedTx.Hash())
			if err != nil {
				fmt.Printf("failed to get transaction receipt (reason: %v).\n", err)
			} else {
				jsonReceipt, _ := receipt.MarshalJSON()
				fmt.Printf("transaction receipt: %s\n", string(jsonReceipt))
			}
		}
		st := prompt.MustSelectStep()
		switch st {
		case step.StepSelectMethod:
			goto SELECT_METHOD
		case step.StepChangeContract:
			goto STEP_SELECT_CONTRACT
		case step.StepChangeContractAddress:
			goto INPUT_CONTRACT_ADDRESS
		case step.StepExit:
			panic("exit")
		}
	}
}
