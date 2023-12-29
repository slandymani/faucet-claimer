package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/keys/hd"
	"github.com/tyler-smith/go-bip39"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type Account struct {
	Sk      *ecdsa.PrivateKey
	Pk      *ecdsa.PublicKey
	Address common.Address
	Balance *big.Int
	Nonce   *uint64
}

func FromMnemonicSeed(mnemonic string, index int) (*btcec.PrivateKey, *btcec.PublicKey) {
	seed := bip39.NewSeed(mnemonic, "")
	master, ch := hd.ComputeMastersFromSeed(seed, []byte("Bitcoin seed"))
	private, _ := hd.DerivePrivateKeyForPath(
		btcec.S256(),
		master,
		ch,
		fmt.Sprintf("44'/60'/0'/0/%d", index),
	)

	return btcec.PrivKeyFromBytes(private[:])
}

func main() {
	config, err := GetConfig()
	if err != nil {
		panic(errors.Wrap(err, "wrong config"))
	}

	accounts := make(map[int]Account)

	fmt.Println("Start generating addresses and getting balances")
	now := time.Now()

	for i := 0; i < config.AddressesNumber+config.StartNumber; i++ {
		sk, pk := FromMnemonicSeed(config.Mnemonic, i+config.StartNumber)
		address := crypto.PubkeyToAddress(*pk.ToECDSA())

		balance := big.NewInt(0)

		accounts[i] = Account{
			Sk:      sk.ToECDSA(),
			Pk:      pk.ToECDSA(),
			Address: address,
			Balance: balance,
		}
	}

	fmt.Println(time.Since(now))

	fmt.Println("Finish generating addresses and getting balances")

	sent := 0

	fmt.Println("Start sending txs")
	fmt.Println("Start time ", time.Now())
	now = time.Now()

	notSentAddr := make(map[int]Account)

	client := http.Client{Timeout: 20 * time.Second}

	for sent < config.AddressesNumber {
		req, err := http.NewRequest(http.MethodPost, config.Faucet, nil)
		if err != nil {
			fmt.Println(errors.Wrap(err, "failed to create request"))
			continue
		}

		q := req.URL.Query()
		q.Add("token", "Q")
		q.Add("address", accounts[sent].Address.String())
		req.URL.RawQuery = q.Encode()

		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(errors.Wrap(err, "failed to send request"))
			continue
		}

		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			fmt.Println(errors.Wrap(err, "failed to read notification service response"))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			if string(b) == "{\"message\":\"Another transaction is processing, please wait\"}" {
				time.Sleep(time.Second * 5)
				continue
			}
			fmt.Println(fmt.Sprintf("tokens not sent to %s: %s", accounts[sent].Address.String(), string(b)))
			notSentAddr[sent] = accounts[sent]
			sent++
			continue
		}

		fmt.Println(fmt.Sprintf("%s sent", accounts[sent].Address.String()))
		sent++

		randDelay := new(big.Int)
		randDelay, err = rand.Int(rand.Reader, big.NewInt(8500))
		if err != nil {
			randDelay = big.NewInt(3000)
		}
		randDelay.Add(randDelay, big.NewInt(3000))
		time.Sleep(time.Duration(randDelay.Int64() * int64(time.Millisecond)))
	}

	fmt.Println(now)
	fmt.Println(time.Since(now))

	fmt.Println("Finish sending txs")

	fmt.Println("Not sent:")
	for _, account := range notSentAddr {
		fmt.Println(account.Address.String())
	}
}
