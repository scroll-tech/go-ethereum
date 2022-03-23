package smt

import (
	"bytes"
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

func NewByte32FromBytesPadding(b []byte) *Byte32 {
	if len(b) != 32 && len(b) != 20 {
		panic("do not support length except for 120bit and 256bit now")
	}
	return pkcs7PadByte32(b)
}

func pkcs7PadByte32(b []byte) *Byte32 {
	if b == nil || len(b) == 0 || len(b) > 32 {
		panic("invalid input data")
	}
	byte32 := new(Byte32)
	copy(byte32[:], b)
	if len(b) == 32 {
		return byte32
	}
	n := 32 - len(b)
	copy(byte32[len(b):], bytes.Repeat([]byte{byte(n)}, n))
	return byte32
}

func UnPadBytes32(b []byte) []byte {
	if b == nil || len(b) != 32 {
		panic("invalid input data")
	}
	n := int(b[31])
	if n == 0 || n > 32 {
		panic("invalid PKCS#7 padding")
	}
	for i := 0; i < n; i++ {
		if int(b[32-n+i]) != n {
			panic("invalid PKCS#7 padding")
		}
	}
	return b[:len(b)-n]
}
