package smt

import (
	"math/big"

	"github.com/iden3/go-iden3-crypto/poseidon"
)

type Byte32 [32]byte

func (b *Byte32) Hash() (*big.Int, error) {
	first16 := new(big.Int).SetBytes(b[0:16])
	last16 := new(big.Int).SetBytes(b[16:32])
	hash, err := poseidon.Hash([]*big.Int{first16, last16})
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func NewByte32FromBytesPaddingZero(b []byte) *Byte32 {
	if len(b) > 32 {
		panic("bytes length larger than 32")
	}
	byte32 := new(Byte32)
	copy(byte32[:], b)
	return byte32
}
