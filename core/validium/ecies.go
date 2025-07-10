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

	// Slots for storing the pubkey commitments.
	// The commitment is the keccak256 hash of the compressed public key.
	CurrentPubkeySlot = common.BigToHash(big.NewInt(1))
	PrevPubkeySlot    = common.BigToHash(big.NewInt(2))
)

type stateDB interface {
	GetState(common.Address, common.Hash) common.Hash
}

// CurrentPubkey retrieves the current public key commitment from the Validium Bridge state.
func CurrentPubkey(state stateDB) []byte {
	val := state.GetState(ValidiumBridgeAddress, CurrentPubkeySlot)
	return val[:]
}

// PrevPubkey retrieves the previous public key commitment from the Validium Bridge state.
func PrevPubkey(state stateDB) []byte {
	val := state.GetState(ValidiumBridgeAddress, PrevPubkeySlot)
	return val[:]
}

// DerivePubkey derives the public key from a given private key.
func DerivePubkey(privateKey []byte) *ecies.PublicKey {
	k := ecies.NewPrivateKeyFromBytes(privateKey)
	return k.PublicKey
}

// DecryptEcies decrypts the given ciphertext using the provided private key.
// It returns the plaintext bytes or an error if decryption fails.
func DecryptEcies(ciphertext []byte, k *ecies.PrivateKey) ([]byte, error) {
	plaintext, err := ecies.Decrypt(k, ciphertext)
	if err != nil {
		log.Warn("failed to decrypt ECIES ciphertext", "ciphertext", common.Bytes2Hex(ciphertext), "error", err)
		return nil, errors.New("decryption failed")
	}

	return plaintext, nil
}
