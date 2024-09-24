package zktrie

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDummyHash(t *testing.T) {
	result, err := dummyHash([]*big.Int{}, nil)
	assert.Equal(t, big.NewInt(0), result)
	assert.Equal(t, hashNotInitErr, err)
}

func TestCheckBigIntInField(t *testing.T) {
	bi := big.NewInt(0)
	assert.True(t, CheckBigIntInField(bi))

	bi = new(big.Int).Sub(Q, big.NewInt(1))
	assert.True(t, CheckBigIntInField(bi))

	bi = new(big.Int).Set(Q)
	assert.False(t, CheckBigIntInField(bi))
}

func TestNewHashAndBigIntFromBytes(t *testing.T) {
	b := bytes.Repeat([]byte{1, 2}, 16)
	h := NewHashFromBytes(b)
	assert.Equal(t, "0102010201020102010201020102010201020102010201020102010201020102", h.Hex())
	assert.Equal(t, "45585349...", h.String())

	h, err := NewHashFromCheckedBytes(b)
	assert.NoError(t, err)
	assert.Equal(t, "0102010201020102010201020102010201020102010201020102010201020102", h.Hex())

	bi, err := NewBigIntFromHashBytes(b)
	assert.NoError(t, err)
	assert.Equal(t, "455853498485199945361735166433836579326217380693297711485161465995904286978", bi.String())

	h1 := NewHashFromBytes(b)
	text, err := h1.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, "455853498485199945361735166433836579326217380693297711485161465995904286978", h1.BigInt().String())
	h2 := &Hash{}
	err = h2.UnmarshalText(text)
	assert.NoError(t, err)
	assert.Equal(t, h1, h2)

	short := []byte{1, 2, 3, 4, 5}
	_, err = NewHashFromCheckedBytes(short)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("expected %d bytes, but got %d bytes", HashByteLen, len(short)), err.Error())

	short = []byte{1, 2, 3, 4, 5}
	_, err = NewBigIntFromHashBytes(short)
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("expected %d bytes, but got %d bytes", HashByteLen, len(short)), err.Error())

	outOfField := bytes.Repeat([]byte{255}, 32)
	_, err = NewBigIntFromHashBytes(outOfField)
	assert.Error(t, err)
	assert.Equal(t, "NewBigIntFromHashBytes: Value not inside the Finite Field", err.Error())
}

func TestNewHashFromBigIntAndString(t *testing.T) {
	bi := big.NewInt(12345)
	h := NewHashFromBigInt(bi)
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000003039", h.Hex())
	assert.Equal(t, "12345", h.String())

	s := "454086624460063511464984254936031011189294057512315937409637584344757371137"
	h, err := NewHashFromString(s)
	assert.NoError(t, err)
	assert.Equal(t, "0101010101010101010101010101010101010101010101010101010101010101", h.Hex())
	assert.Equal(t, "45408662...", h.String())
}

func TestNewHashFromBytes(t *testing.T) {
	h := HashZero
	read, err := rand.Read(h[:])
	require.NoError(t, err)
	require.Equal(t, HashByteLen, read)
	require.Equal(t, h, *NewHashFromBytes(h.Bytes()))
}
