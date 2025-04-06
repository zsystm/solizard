package lib

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zsystm/solizard/internal/log"
)

// NativeCurrency represents the native currency information
type NativeCurrency struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// ChainInfo represents the structure of each chain in the JSON
type ChainInfo struct {
	Name           string         `json:"name"`
	ChainID        uint64         `json:"chainId"`
	ShortName      string         `json:"shortName"`
	NetworkID      uint64         `json:"networkId"`
	NativeCurrency NativeCurrency `json:"nativeCurrency"`
	RPC            []string       `json:"rpc"`
	Faucets        []string       `json:"faucets"`
	InfoURL        string         `json:"infoURL"`
}

func ParseChainsJSON(path string) ([]*ChainInfo, error) {
	// Read the JSON file
	jsonData, err := os.ReadFile(path)
	if err != nil {
		log.Error(fmt.Sprintf("failed to read chain infos (reason: %v)\n", err))
		return nil, err
	}

	// Create a slice to hold the chain information
	var chainInfos []*ChainInfo

	// Parse JSON into the slice
	err = json.Unmarshal(jsonData, &chainInfos)
	if err != nil {
		log.Error(fmt.Sprintf("failed to parse JSON file: %v\n", err))
		return nil, err
	}

	return chainInfos, nil
}

func GetChainInfoByID(chainInfos []*ChainInfo, chainID uint64) (*ChainInfo, error) {
	for _, chainInfo := range chainInfos {
		if chainInfo.ChainID == chainID {
			return chainInfo, nil
		}
	}
	return nil, fmt.Errorf("chain ID %d not found", chainID)
}
