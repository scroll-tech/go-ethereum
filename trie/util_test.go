package trie

import (
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetBitBigEndian(t *testing.T) {
	bitmap := make([]byte, 8)

	SetBitBigEndian(bitmap, 3)
	SetBitBigEndian(bitmap, 15)
	SetBitBigEndian(bitmap, 27)
	SetBitBigEndian(bitmap, 63)

	expected := []byte{0x80, 0x0, 0x0, 0x0, 0x8, 0x0, 0x80, 0x8}
	assert.Equal(t, expected, bitmap)
}

func TestBitManipulations(t *testing.T) {
	bitmap := []byte{0b10101010, 0b01010101}

	bitResults := make([]bool, 16)
	for i := uint(0); i < 16; i++ {
		bitResults[i] = TestBit(bitmap, i)
	}

	expectedBitResults := []bool{
		false, true, false, true, false, true, false, true,
		true, false, true, false, true, false, true, false,
	}
	assert.Equal(t, expectedBitResults, bitResults)

	bitResultsBigEndian := make([]bool, 16)
	for i := uint(0); i < 16; i++ {
		bitResultsBigEndian[i] = TestBitBigEndian(bitmap, i)
	}

	expectedBitResultsBigEndian := []bool{
		true, false, true, false, true, false, true, false,
		false, true, false, true, false, true, false, true,
	}
	assert.Equal(t, expectedBitResultsBigEndian, bitResultsBigEndian)
}

func TestBigEndianBitsToBigInt(t *testing.T) {
	bits := []bool{true, false, true, false, true, false, true, false}
	result := BigEndianBitsToBigInt(bits)
	expected := big.NewInt(170)
	assert.Equal(t, expected, result)
}

func TestToSecureKey(t *testing.T) {
	secureKey, err := ToSecureKey([]byte("testKey"))
	assert.NoError(t, err)
	assert.Equal(t, "10380846131134096261855654117842104248915214759620570252072028416245925344412", secureKey.String())
}

func TestToSecureKeyBytes(t *testing.T) {
	secureKeyBytes, err := ToSecureKeyBytes([]byte("testKey"))
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x16, 0xf3, 0x59, 0xc7, 0x30, 0x7e, 0x8, 0x97, 0xdc, 0x7c, 0x6b, 0x99, 0x53, 0xd1, 0xe1, 0xd8, 0x3, 0x6d, 0xc3, 0x83, 0xd4, 0xa, 0x0, 0x19, 0x9e, 0xda, 0xf0, 0x65, 0x27, 0xda, 0xf4, 0x9c}, secureKeyBytes.Bytes())
}

func TestHashElems(t *testing.T) {
	fst := big.NewInt(5)
	snd := big.NewInt(3)
	elems := make([]*big.Int, 32)
	for i := range elems {
		elems[i] = big.NewInt(int64(i + 1))
	}

	result, err := HashElems(fst, snd, elems...)
	assert.NoError(t, err)
	assert.Equal(t, "0746c424799ef6ad7916511016a5b8e30688fa6d62664eeb97d9f2ba07685ed8", result.Hex())
}

func TestPreHandlingElems(t *testing.T) {
	flagArray := uint32(0b10101010101010101010101010101010)
	elems := make([]Byte32, 32)
	for i := range elems {
		elems[i] = *NewByte32FromBytes([]byte("test" + strconv.Itoa(i+1)))
	}

	result, err := HandlingElemsAndByte32(flagArray, elems)
	assert.NoError(t, err)
	assert.Equal(t, "0c36a6406e35e1fec2a5602fcd80ab04ec24d727676a953673c0850d205f9378", result.Hex())

	elems = elems[:1]
	result, err = HandlingElemsAndByte32(flagArray, elems)
	assert.NoError(t, err)
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000007465737431", result.Hex())
}
