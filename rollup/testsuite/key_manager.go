package testsuite

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	commonETH "github.com/ethereum/go-ethereum/common"
	cryptoETH "github.com/ethereum/go-ethereum/crypto"

	bindETH "github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

type KeyManager struct {
	keys map[string]*ecdsa.PrivateKey
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

func (k *KeyManager) Transactor(alias string, chainID *big.Int) *bind.TransactOpts {
	key := k.Key(alias)

	transactor, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		panic(err)
	}

	return transactor
}

func (k *KeyManager) L1Address(alias string) commonETH.Address {
	key := k.Key(alias)
	address := cryptoETH.PubkeyToAddress(key.PublicKey)
	return address
}

func (k *KeyManager) L1Transactor(alias string, chainID *big.Int) *bindETH.TransactOpts {
	key := k.Key(alias)

	transactor, err := bindETH.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		panic(err)
	}

	return transactor
}
