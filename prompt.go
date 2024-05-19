package main

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/zsystm/promptui"
)

const DefaultPromptListSize = 10

func MustSelectContractABI(abis map[string]abi.ABI) abi.ABI {
	contractNames := make([]string, 0, len(abis))
	for name := range abis {
		contractNames = append(contractNames, name)
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf("Select the contract to interact (total: %d)", len(abis)),
		Items: contractNames,
		Size:  DefaultPromptListSize,
		Searcher: func(input string, index int) bool {
			// if there is a method name that contains the input, return true
			return strings.Contains(contractNames[index], input)
		},
		StartInSearchMode: shouldSupportSearchMode(len(abis)),
	}

	_, selected, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return abis[selected]
}

func MustInputRpcUrl() string {
	prompt := promptui.Prompt{
		Label:     "Enter the RPC URL",
		Default:   DefaultRPCURL,
		AllowEdit: true,
	}
	rpcURL, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return rpcURL
}

func MustInputContractAddress() string {
	address, err := getUserInput("Enter the contract address")
	if err != nil {
		panic(err)
	}
	return address
}

func MustSelectReadOrWrite() MethodType {
	prompt := promptui.Select{
		Label: "Read or Write contract",
		Items: []MethodType{ReadMethod, WriteMethod},
	}

	_, selected, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return MethodType(selected)
}

func InputPrivateKey() (*ecdsa.PrivateKey, error) {
	prompt := promptui.Prompt{
		Label: "Enter your private key to execute contract (e.g. 1234..., no 0x prefix)",
		Mask:  '*',
	}
	privateKey, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, err
	}
	return pk, nil
}

func MustInputChainID() big.Int {
	chainIDStr, err := getUserInput("Enter the chain ID to execute contract method (e.g. 1 for mainnet, 3 for ropsten, 4 for rinkeby, 5 for goerli)")
	if err != nil {
		panic(err)
	}
	chainID := new(big.Int)
	chainID.SetString(chainIDStr, 10)
	return *chainID
}

func MustSelectMethod(contractABI abi.ABI, rw MethodType) (string, abi.Method) {
	var methodNames []string
	for name := range getMethodsByType(contractABI, rw) {
		methodNames = append(methodNames, name)
	}

	prompt := promptui.Select{
		Label: fmt.Sprintf("Select Method (total: %d)", len(methodNames)),
		Items: methodNames,
		Size:  DefaultPromptListSize,
		Searcher: func(input string, index int) bool {
			// if there is a method name that contains the input, return true
			return strings.Contains(methodNames[index], input)
		},
		StartInSearchMode: shouldSupportSearchMode(len(methodNames)),
	}

	_, selectedMethod, err := prompt.Run()
	if err != nil {
		panic(err)
	}

	return selectedMethod, contractABI.Methods[selectedMethod]
}

func getUserInput(promptText string) (string, error) {
	prompt := promptui.Prompt{
		Label: promptText,
	}
	return prompt.Run()
}

func MustSelectStep() Step {
	prompt := promptui.Select{
		Label: "Select the next step",
		Items: []Step{StepSelectContract, StepInputContractAddress, StepSelectMethod, StepExit},
	}

	_, selected, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return Step(selected)
}

func MustCreateInputDataForMethod(method abi.Method) []byte {
	lenInput := len(method.Inputs)
	if lenInput == 0 {
		// short circuit if no arguments
		return method.ID
	}

	arguments := make(abi.Arguments, 0, lenInput)
	for _, input := range method.Inputs {
		arguments = append(arguments, abi.Argument{
			Name:    input.Name,
			Type:    input.Type,
			Indexed: input.Indexed,
		})
	}

	// get user input for each argument
	var args []interface{}
	for _, arg := range arguments {
		strValue, err := getUserInput(fmt.Sprintf("Enter value for %s (type: %s)", arg.Name, arg.Type))
		if err != nil {
			panic(err)
		}
		var value interface{}
		switch arg.Type.T {
		case abi.IntTy:
			value, _ = new(big.Int).SetString(strValue, 10)
		case abi.UintTy:
			value, _ = new(big.Int).SetString(strValue, 10)
		case abi.BoolTy:
			value = strValue == "true"
		case abi.StringTy:
			value = strValue
		case abi.SliceTy, abi.ArrayTy:
			value = parseArrayOrSliceInput(strValue, arg.Type)
		case abi.TupleTy:
			value = parseTupleInput(strValue, arg.Type)
		case abi.AddressTy:
			value = common.HexToAddress(strValue)
		case abi.FixedBytesTy, abi.BytesTy:
			value = common.Hex2Bytes(strValue)
		case abi.HashTy:
			value = common.HexToHash(strValue)
		case abi.FixedPointTy, abi.FunctionTy:
			// TODO: implement
			panic("type not supported")
		default:
			panic("unsupported type")
		}
		// change value to the correct type
		args = append(args, value)
	}
	// pack the arguments
	data, err := method.Inputs.Pack(args...)
	if err != nil {
		panic(err)
	}
	return append(method.ID, data...)
}
