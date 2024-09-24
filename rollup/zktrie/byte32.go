package zktrie

import (
	"math/big"
)

type Byte32 [32]byte

func (b *Byte32) Hash() (*big.Int, error) {
	first16 := new(big.Int).SetBytes(b[0:16])
	last16 := new(big.Int).SetBytes(b[16:32])
	hash, err := hashScheme([]*big.Int{first16, last16}, big.NewInt(HASH_DOMAIN_BYTE32))
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func (b *Byte32) Bytes() []byte { return b[:] }

// same action as common.Hash (truncate bytes longer than 32 bytes FROM beginning,
// and padding 0 at the beginning for shorter bytes)
func NewByte32FromBytes(b []byte) *Byte32 {

	byte32 := new(Byte32)

	if len(b) > 32 {
		b = b[len(b)-32:]
	}

	copy(byte32[32-len(b):], b)
	return byte32
}

// create bytes32 with zeropadding to shorter bytes, or truncate it
func NewByte32FromBytesPaddingZero(b []byte) *Byte32 {
	byte32 := new(Byte32)
	copy(byte32[:], b)
	return byte32
}
