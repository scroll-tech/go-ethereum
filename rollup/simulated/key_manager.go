package simulated

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

type KeyManager struct {
	chainID *big.Int
	keys    map[string]*ecdsa.PrivateKey
}

func NewKeyManager() *KeyManager {
	return &KeyManager{
		keys: make(map[string]*ecdsa.PrivateKey),
	}
}

func (k *KeyManager) Key(alias string) *ecdsa.PrivateKey {
	if key, ok := k.keys[alias]; ok {
		return key
	}

	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	fmt.Println("Generated key:", alias, "address:", crypto.PubkeyToAddress(key.PublicKey))
	k.keys[alias] = key
	return key
}

func (k *KeyManager) Address(alias string) common.Address {
	key := k.Key(alias)
	address := crypto.PubkeyToAddress(key.PublicKey)
	return address
}

func (k *KeyManager) SetChainID(chainID *big.Int) {
	k.chainID = chainID
}

func (k *KeyManager) Transactor(alias string) *bind.TransactOpts {
	key := k.Key(alias)

	transactor, err := bind.NewKeyedTransactorWithChainID(key, k.chainID)
	if err != nil {
		panic(err)
	}

	return transactor
}
