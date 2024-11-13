package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

type AddressBook map[string]struct {
	Address string `json:"address"`
}

// ReadAddressBook reads the address book from the given path
func ReadAddressBook(path string) (AddressBook, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ab AddressBook
	if err = json.Unmarshal(data, &ab); err != nil {
		return nil, err
	}
	return ab, nil
}

func (ab AddressBook) Validate() error {
	// check if the address book is empty
	if len(ab) == 0 {
		return fmt.Errorf("address book is empty")
	}
	for name, entry := range ab {
		if entry.Address == "" {
			return fmt.Errorf("address book entry %s has empty address", name)
		}
		if !common.IsHexAddress(entry.Address) {
			return fmt.Errorf("address book entry %s has invalid address", name)
		}
	}

	return nil
}
