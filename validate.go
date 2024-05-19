package main

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

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
