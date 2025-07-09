package validium

import (
	"errors"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"

	ecies "github.com/scroll-tech/ecies-go/v2"
)

// State constants for Validium Bridge.
var (
	ValidiumBridgeAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
	CurrentPubkeySlot     = common.BigToHash(big.NewInt(1))
	PreviousPubkeySlot    = common.BigToHash(big.NewInt(2))
)

type stateDB interface {
	GetState(common.Address, common.Hash) common.Hash
}

func parsePubkey(raw []byte) *ecies.PublicKey {
	pk, err := ecies.NewPublicKeyFromBytes(raw)
	if err != nil {
		log.Error("invalid pubkey", "raw", common.Bytes2Hex(raw), "error", err)
		return nil
	}
	return pk
}

// CurrentPubkey retrieves the current public key from the Validium Bridge state.
// It returns a pointer to the public key or nil if the key is invalid.
func CurrentPubkey(state stateDB) *ecies.PublicKey {
	val := state.GetState(ValidiumBridgeAddress, CurrentPubkeySlot)
	return parsePubkey(val[:])
}

// PreviousPubkey retrieves the previous public key from the Validium Bridge state.
// It returns a pointer to the public key or nil if the key is invalid.
func PreviousPubkey(state stateDB) *ecies.PublicKey {
	val := state.GetState(ValidiumBridgeAddress, PreviousPubkeySlot)
	return parsePubkey(val[:])
}

// DerivePubkey derives the public key from a given private key.
func DerivePubkey(privateKey []byte) *ecies.PublicKey {
	k := ecies.NewPrivateKeyFromBytes(privateKey)
	return k.PublicKey
}

// DecryptEcies decrypts the given ciphertext using the provided private key.
// It returns the plaintext bytes or an error if decryption fails.
func DecryptEcies(ciphertext, privateKey []byte) ([]byte, error) {
	k := ecies.NewPrivateKeyFromBytes(privateKey)

	plaintext, err := ecies.Decrypt(k, ciphertext)
	if err != nil {
		log.Warn("failed to decrypt ECIES ciphertext", "ciphertext", common.Bytes2Hex(ciphertext), "error", err)
		return nil, errors.New("decryption failed")
	}

	return plaintext, nil
}
