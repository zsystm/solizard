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

// WriteContractInfos writes the contract information to a file
func WriteContractInfos(path string, infos []ContractInfo) error {
	// Validate all contract infos before writing
	if err := ValidateContractInfos(infos); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}

	// Marshal the contract infos to JSON with indentation for readability
	data, err := json.MarshalIndent(infos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contract infos: %v", err)
	}

	// Write the data to file with appropriate permissions (0644)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write contract infos to %s: %v", path, err)
	}

	return nil
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
