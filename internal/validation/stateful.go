package validation

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zsystm/solizard/internal/ctx"
)

// ValidateContractAddress validates the given string is a valid contract address
// and sets the contract address in the ctx
func ValidateContractAddress(ctx *ctx.Context, s string) error {
	if err := ValidateAddress(s); err != nil {
		return err
	}

	cAddr := common.HexToAddress(s)
	// check if the contract exists on the chain
	code, err := ctx.EthClient().CodeAt(context.TODO(), cAddr, nil)
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
