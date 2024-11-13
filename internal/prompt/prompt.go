package prompt

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/zsystm/promptui"

	internalabi "github.com/zsystm/solizard/internal/abi"
	"github.com/zsystm/solizard/internal/config"
	"github.com/zsystm/solizard/internal/step"
	"github.com/zsystm/solizard/internal/validation"
)

const DefaultPromptListSize = 10

// MustSelectApplyConfig returns true if the user don't want to setup manually
// which means the user wants to apply the config file
func MustSelectApplyConfig() bool {
	prompt := promptui.Prompt{
		Label: "found config file at ~/.solizard/config.toml, apply? [Y/n]",
	}
	ret, _ := prompt.Run()
	if NoSelected(ret) {
		return false
	}
	return true
}

// MustSelectContractABI prompts the user to select a contract ABI and returns the selected contract name and ABI
func MustSelectContractABI(abis map[string]abi.ABI) (string, abi.ABI) {
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
	return strings.TrimSuffix(selected, ".abi"), abis[selected]
}

func MustInputRpcUrl() string {
	prompt := promptui.Prompt{
		Label:     "Enter the RPC URL",
		Default:   config.DefaultRpcURL,
		AllowEdit: true,
		Validate:  validation.ValidateRpcURL,
	}
	rpcURL, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return rpcURL
}

func MustSelectAddressBookUsage(contractAddr string) bool {
	prompt := promptui.Prompt{
		Label: fmt.Sprintf("Use %s as contract address? [Y/n]", contractAddr),
	}
	ret, _ := prompt.Run()
	if NoSelected(ret) {
		return false
	}
	return true
}

func MustInputContractAddress() string {
	prompt := promptui.Prompt{
		Label:    "Enter the contract address",
		Validate: validation.ValidateAddress,
	}
	address, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return address
}

func MustSelectReadOrWrite() internalabi.MethodType {
	prompt := promptui.Select{
		Label: "Read or Write contract",
		Items: []internalabi.MethodType{internalabi.ReadMethod, internalabi.WriteMethod},
	}

	_, selected, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return internalabi.MethodType(selected)
}

func MustInputPrivateKey() *ecdsa.PrivateKey {
	prompt := promptui.Prompt{
		Label:    "Enter your private key to execute contract (e.g. 1234..., no 0x prefix)",
		Mask:     '*',
		Validate: validation.ValidatePrivateKey,
	}
	privateKey, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		panic(err)
	}
	return pk
}

func MustInputChainID() big.Int {
	prompt := promptui.Prompt{
		Label:    "Enter the chain ID to execute contract method (e.g. 1 for mainnet, 3 for ropsten, 4 for rinkeby, 5 for goerli)",
		Validate: validation.ValidateInt,
	}
	chainIDStr, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	chainID := new(big.Int)
	chainID.SetString(chainIDStr, 10)
	return *chainID
}

func MustSelectMethod(contractABI abi.ABI, rw internalabi.MethodType) (string, abi.Method) {
	var methodNames []string
	for name := range internalabi.GetMethodsByType(contractABI, rw) {
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

func MustSelectStep() step.Step {
	prompt := promptui.Select{
		Label: "Select the next step",
		Items: []step.Step{step.StepChangeContract, step.StepChangeContractAddress, step.StepSelectMethod, step.StepExit},
	}

	_, selected, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	return step.Step(selected)
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
		// TODO: Make validation for each type
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
			value = internalabi.ParseArrayOrSliceInput(strValue, arg.Type)
		case abi.TupleTy:
			value = internalabi.ParseTupleInput(strValue, arg.Type)
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

func MustInputValue() *big.Int {
	prompt := promptui.Prompt{
		Label:    "Enter the value to be sent with the contract call (in wei)",
		Validate: validation.ValidateInt,
	}
	valueStr, err := prompt.Run()
	if err != nil {
		panic(err)
	}
	value := new(big.Int)
	value.SetString(valueStr, 10)
	return value
}

const SelectableListSize = 4

func shouldSupportSearchMode(listLen int) bool {
	return listLen > SelectableListSize
}

func NoSelected(s string) bool {
	return strings.ToLower(s) == "n"
}
