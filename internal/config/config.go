package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/pelletier/go-toml"
)

const DefaultRpcURL = "http://localhost:8545"

type Config struct {
	RpcURL     string `toml:"rpc_url"`
	PrivateKey string `toml:"private_key"`
	ChainId    uint64 `toml:"chain_id"`
	WaitTime   string `toml:"wait_time"`
}

func DefaultConfig() *Config {
	pk, _ := crypto.GenerateKey()
	hexPriv := common.Bytes2Hex(crypto.FromECDSA(pk))
	return &Config{
		RpcURL:     DefaultRpcURL,
		PrivateKey: hexPriv,
		ChainId:    1,
		WaitTime:   "5s",
	}
}

func ReadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	// Other fields are validated in NewCtx
	failMsg := "config file is invalid"
	if _, err := time.ParseDuration(c.WaitTime); err != nil {
		return fmt.Errorf("%s:: invalid wait time: %v", failMsg, err)
	}
	return nil
}
