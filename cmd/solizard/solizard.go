package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	internalabi "github.com/zsystm/solizard/internal/abi"
	"github.com/zsystm/solizard/internal/config"
	"github.com/zsystm/solizard/internal/ctx"
	"github.com/zsystm/solizard/internal/log"
	"github.com/zsystm/solizard/internal/prompt"
	"github.com/zsystm/solizard/internal/step"
	"github.com/zsystm/solizard/internal/validation"
	"github.com/zsystm/solizard/lib"
)

//go:embed embeds/*
var embeddedFiles embed.FS

const SolizardDir = ".solizard"

var (
	// AbiDir is the directory where all abi files are stored
	// default is $HOME/.solizard/abis
	AbiDir            = "abis"
	ZeroAddr          = common.Address{}
	ConfigPath        = ""
	ConfigExist       = false
	ContractInfosPath = ""
	ContractInfoExist = false
	Conf              *config.Config
	ContractInfos     []config.ContractInfo
	ChainInfos        []*lib.ChainInfo
)

func init() {
	// get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Error(fmt.Sprintf("failed to get user's home directory (reason: %v)\n", err))
		os.Exit(1)
	}
	dir := homeDir + "/" + SolizardDir
	AbiDir = dir + "/" + AbiDir
	ConfigPath = filepath.Join(homeDir, SolizardDir, "config.toml")

	if _, err = os.Stat(AbiDir); os.IsNotExist(err) {
		if err = os.MkdirAll(AbiDir, 0755); err != nil {
			log.Error(fmt.Sprintf("failed to create abi directory (reason: %v)\n", err))
			os.Exit(1)
		}
	}
	// copy embeds to directory and files if not exists
	if err = fs.WalkDir(embeddedFiles, "embeds", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := embeddedFiles.ReadFile(path)
		if err != nil {
			log.Error(err.Error())
			return err
		}
		if d.Type().IsRegular() && d.Name()[len(d.Name())-4:] == ".abi" {
			abiFile := filepath.Join(AbiDir, d.Name())
			// If abi file does not exist, create it
			if _, err = os.Stat(abiFile); os.IsNotExist(err) {
				if err = os.WriteFile(abiFile, data, 0644); err != nil {
					log.Error(err.Error())
					return err
				}
			}
		}
		if d.Type().IsRegular() && d.Name() == "config.toml" {
			// If config file does not exist, create it
			if _, err = os.Stat(ConfigPath); os.IsNotExist(err) {
				if err = os.WriteFile(ConfigPath, data, 0644); err != nil {
					log.Error(err.Error())
					return err
				}
			}
			ConfigExist = true
		}
		if d.Type().IsRegular() && d.Name() == "contract_infos.json" {
			// If contract_infos.json file does not exist, create it
			ContractInfosPath = filepath.Join(homeDir, SolizardDir, "contract_infos.json")
			if _, err = os.Stat(ContractInfosPath); os.IsNotExist(err) {
				// create contract_infos.json if not exists
				if err = os.WriteFile(ContractInfosPath, data, 0644); err != nil {
					log.Error(fmt.Sprintf("failed to create contract_infos.json (reason: %v)\n", err))
					os.Exit(1)
				}
			}
			ContractInfoExist = true
		}
		if d.Type().IsRegular() && d.Name() == "chains_mini.json" {
			// If chains_mini.json file does not exist, create it
			ChainInfosPath := filepath.Join(homeDir, SolizardDir, "chains_mini.json")
			if _, err = os.Stat(ChainInfosPath); os.IsNotExist(err) {
				// create chains_mini.json if not exists
				if err = os.WriteFile(ChainInfosPath, data, 0644); err != nil {
					log.Error(fmt.Sprintf("failed to create chains_mini.json (reason: %v)\n", err))
					os.Exit(1)
				}
			}
			ChainInfos, err = lib.ParseChainsJSON(ChainInfosPath)
			if err != nil {
				log.Error(fmt.Sprintf("failed to read chain infos (reason: %v)\n", err))
				os.Exit(1)
			}
		}
		return nil
	}); err != nil {
		// If error occurs, print error message and exit
		log.Error(fmt.Sprintf("failed to walk embedded files (reason: %v)\n", err))
		os.Exit(1)
	}

	if Conf, err = config.ReadConfig(ConfigPath); err != nil {
		log.Error(fmt.Sprintf("failed to read config file (reason: %v)\n", err))
		ConfigExist = false
	} else {
		ConfigExist = true
	}

	if ContractInfos, err = config.ReadContractInfos(ContractInfosPath); err != nil {
		log.Error(fmt.Sprintf("failed to read contract infos (reason: %v)\n", err))
		os.Exit(1)
	}
	if err = config.ValidateContractInfos(ContractInfos); err != nil {
		log.Error(fmt.Sprintf("address book is invalid (reason: %v)\n", err))
		panic(err)
	}
}

func Run() error {
	fmt.Println(`🦎 Welcome to Solizard v1.8.0 🦎`)
	mAbi, err := internalabi.LoadABIs(AbiDir)
	if err != nil {
		return err
	}

	sctx := new(ctx.Context)
	if ConfigExist {
		if prompt.MustSelectApplyConfig() {
			sctx = ctx.NewCtx(Conf)
			ctx.PrintContext(sctx, ChainInfos)
		}
	}

	var selectedContractName string
	var selectedAbi abi.ABI
	// start the main loop
	for {
	STEP_SELECT_CONTRACT:
		selectedContractName, selectedAbi = prompt.MustSelectContractABI(mAbi)
	INPUT_RPC_URL:
		if sctx.EthClient() == nil {
			rpcURL := prompt.MustInputRpcUrl()
			client, err := ethclient.Dial(rpcURL)
			if err != nil {
				log.Error(fmt.Sprintf("failed to connect to given rpc url: %v, please input valid one\n", err))
				goto INPUT_RPC_URL
			}
			sctx.SetEthClient(client)
			Conf.RpcURL = rpcURL
			if err = config.WriteConfig(ConfigPath, Conf); err != nil {
				log.Error(fmt.Sprintf("failed to write config file (reason: %v)\n", err))
			}
		}
	INPUT_CONTRACT_ADDRESS:
		// check if address book exists
		useContractInfo := false
		var contractAddress string
		if ContractInfoExist {
			for _, ci := range ContractInfos {
				if ci.Name == selectedContractName {
					if yes := prompt.MustSelectAddressBookUsage(ci.Address); yes {
						contractAddress = ci.Address
						useContractInfo = true
						break
					}
				}
			}
		}
		if !useContractInfo {
			contractAddress = prompt.MustInputContractAddress()
		}
		if err := validation.ValidateContractAddress(sctx, contractAddress); err != nil {
			log.Error(fmt.Sprintf("Invalid contract address (reason: %v)\n", err))
			goto INPUT_CONTRACT_ADDRESS
		}

	SELECT_METHOD:
		rw := prompt.MustSelectReadOrWrite()
		if rw == internalabi.WriteMethod {
			// input private key
			if sctx.PrivateKey() == nil {
				pk := prompt.MustInputPrivateKey()
				sctx.SetPrivateKey(pk)
				// Don't write private key to config file for security reasons
			}
			// input chainId
			if sctx.ChainId().Sign() == 0 {
				chainID := prompt.MustInputChainID()
				sctx.SetChainId(&chainID)
				Conf.ChainId = chainID.Uint64()
				if err = config.WriteConfig(ConfigPath, Conf); err != nil {
					log.Error(fmt.Sprintf("failed to write config file (reason: %v)\n", err))
				}
			}
		}
		methodName, method := prompt.MustSelectMethod(selectedAbi, rw)
		input := prompt.MustCreateInputDataForMethod(method)
		if rw == internalabi.ReadMethod {
			callMsg := ethereum.CallMsg{From: ZeroAddr, To: sctx.ContractAddress(), Data: input}
			output, err := sctx.EthClient().CallContract(context.TODO(), callMsg, nil)
			if err != nil {
				log.Error(fmt.Sprintf("failed to call contract (reason: %v)\n", err))
				return err
			}
			res, err := selectedAbi.Unpack(methodName, output)
			if err != nil {
				log.Error(fmt.Sprintf("failed to unpack output (reason: %v)\n", err))
				return err
			}
			fmt.Printf("output: %v\n", res)
		} else {
			value := common.Big0
			if method.IsPayable() {
				value = prompt.MustInputValue()
			}
			nonce, err := sctx.EthClient().NonceAt(context.TODO(), crypto.PubkeyToAddress(sctx.PrivateKey().PublicKey), nil)
			if err != nil {
				log.Error(fmt.Sprintf("failed to get nonce (reason: %v), maybe rpc is not working.\n", err))
				goto INPUT_RPC_URL
			}
			gasPrice, err := sctx.EthClient().SuggestGasPrice(context.TODO())
			if err != nil {
				log.Error(fmt.Sprintf("failed to get gas price (reason: %v), maybe rpc is not working.\n", err))
				goto INPUT_RPC_URL
			}
			// TODO: Change to EthClient().EstimateGas() call.
			sufficientGasLimit := uint64(3000000)
			unsignedTx := types.NewTx(&types.LegacyTx{
				To:       sctx.ContractAddress(),
				Nonce:    nonce,
				Value:    value,
				Gas:      sufficientGasLimit,
				GasPrice: gasPrice,
				Data:     input,
			})
			signedTx, err := types.SignTx(unsignedTx, types.NewEIP155Signer(sctx.ChainId()), sctx.PrivateKey())
			if err != nil {
				log.Error(fmt.Sprintf("failed to sign transaction (reason: %v)\n", err))
				return err
			}
			if err = sctx.EthClient().SendTransaction(context.TODO(), signedTx); err != nil {
				log.Error(fmt.Sprintf("failed to send transaction (reason: %v), maybe rpc is not working.\n", err))
				return err
			}
			log.Info(fmt.Sprintf("transaction sent (txHash %v).\n", signedTx.Hash().Hex()))
			// sleep for x seconds to wait for transaction to be mined
			waitTime, err := time.ParseDuration(Conf.WaitTime)
			log.Info(fmt.Sprintf("waiting for transaction to be mined... (sleep %s)\n", waitTime.String()))
			time.Sleep(waitTime)
			receipt, err := sctx.EthClient().TransactionReceipt(context.TODO(), signedTx.Hash())
			if err != nil {
				log.Error(fmt.Sprintf("failed to get transaction receipt (reason: %v)\n", err))
			} else {
				jsonReceipt, _ := receipt.MarshalJSON()
				log.Info(fmt.Sprintf("transaction receipt: %s\n", string(jsonReceipt)))
			}
		}
		if !useContractInfo {
			// Save interacted contract address to address book
			ContractInfos = append(
				ContractInfos,
				config.ContractInfo{
					Name:    selectedContractName,
					Address: contractAddress,
				},
			)
			if err = config.WriteContractInfos(ContractInfosPath, ContractInfos); err != nil {
				log.Error(fmt.Sprintf("failed to write contract infos (reason: %v)\n", err))
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
