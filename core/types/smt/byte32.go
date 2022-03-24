package smt

import (
	"bytes"
	"fmt"
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

func NewByte32FromBytesPadding(b []byte) *Byte32 {
	if len(b) != 0 && len(b) != 32 && len(b) != 20 {
		panic(fmt.Errorf("do not support length except for 120bit and 256bit now. data: %v len: %v", b, len(b)))
	}
	return pkcs7PadByte32(b)
}

func pkcs7PadByte32(b []byte) *Byte32 {

	if b == nil || len(b) == 0 {
		//panic("invalid input data")
	}
	if len(b) > 32 {
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
	isPad := true
	if n == 0 || n > 32 {
		isPad = false
	} else {
		for i := 0; i < n; i++ {
			if int(b[32-n+i]) != n {
				isPad = false
				break
			}
		}
	}
	if isPad {
		return b[:len(b)-n]
	} else {
		return b
	}
}
