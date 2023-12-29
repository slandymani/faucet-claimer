package main

import (
	"gitlab.com/distributed_lab/figure/v3"
	"gitlab.com/distributed_lab/kit/kv"
)

type EthConfig struct {
	Mnemonic        string `fig:"mnemonic"`
	AddressesNumber int    `fig:"addresses_number"`
	StartNumber     int    `fig:"start_number"`
	Faucet          string `fig:"faucet"`
}

func GetConfig() (EthConfig, error) {
	var result EthConfig

	err := figure.
		Out(&result).
		With(figure.BaseHooks, figure.EthereumHooks).
		From(kv.MustGetStringMap(kv.MustFromEnv(), "ethereum")).
		Please()

	return result, err
}
