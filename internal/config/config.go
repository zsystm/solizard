package config

import (
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/pelletier/go-toml"
)

const DefaultRpcURL = "http://localhost:8545"

type Config struct {
	RpcURL     string        `toml:"rpc_url"`
	PrivateKey string        `toml:"private_key"`
	ChainId    uint64        `toml:"chain_id"`
	WateTime   time.Duration `toml:"waiting_time_for_transaction"`
}

func DefaultConfig() *Config {
	pk, _ := crypto.GenerateKey()
	hexPriv := common.Bytes2Hex(crypto.FromECDSA(pk))
	return &Config{
		RpcURL:     DefaultRpcURL,
		PrivateKey: hexPriv,
		ChainId:    1,
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
