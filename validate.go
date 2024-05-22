package main

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func ValidateRpcURL(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("input cannot be empty")
	}
	if _, err := url.ParseRequestURI(s); err != nil {
		return fmt.Errorf("invalid rpc url: %v", err)
	}
	return nil
}

func ValidateAddress(s string) error {
	if !common.IsHexAddress(s) {
		return fmt.Errorf("invalid address")
	}
	return nil
}

func ValidatePrivateKey(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("input cannot be empty")
	}
	if _, err := crypto.HexToECDSA(s); err != nil {
		return fmt.Errorf("invalid private key: %v", err)
	}
	return nil
}

func ValidateInt(s string) error {
	if _, ok := new(big.Int).SetString(s, 10); !ok {
		return fmt.Errorf("invalid int")
	}
	return nil
}

// DirContainsFiles returns nil if a directory exists and contains files
func DirContainsFiles(dir string) error {
	// check if abi directory exists and there are abi files
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read abi directory, you must create %s directory", AbiDir)
	}
	if len(files) == 0 {
		return fmt.Errorf("no abi files found in %s directory, you must put abi files in that directory", AbiDir)
	}
	return nil
}

// Stateful validation functions

// ValidateContractAddress validates the given string is a valid contract address
// and sets the contract address in the context
func ValidateContractAddress(ctx *Ctx, s string) error {
	if err := ValidateAddress(s); err != nil {
		return err
	}

	cAddr := common.HexToAddress(s)
	// check if the contract exists on the chain
	code, err := ctx.ethCli.CodeAt(context.TODO(), cAddr, nil)
	if err != nil {
		return fmt.Errorf("failed to get contract code: %v", err)
	}
	// check if the contract address is a contract address
	if len(code) == 0 {
		return fmt.Errorf("given contract address is not a contract address, no bytecode in chain")
	}
	ctx.SetContractAddress(&cAddr)
	return nil
}
