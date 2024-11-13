package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

type ContractInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func ReadContractInfos(path string) ([]ContractInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ci []ContractInfo
	if err = json.Unmarshal(data, &ci); err != nil {
		return nil, err
	}
	return ci, nil
}

func (ci ContractInfo) Validate() error {
	if ci.Name == "" {
		return fmt.Errorf("contract name is empty")
	}
	if ci.Address == "" {
		return fmt.Errorf("contract address is empty")
	}
	if !common.IsHexAddress(ci.Address) {
		return fmt.Errorf("contract address is invalid")
	}
	return nil
}

func ValidateContractInfos(cis []ContractInfo) error {
	for _, c := range cis {
		if err := c.Validate(); err != nil {
			return err
		}
	}
	return nil
}
