package zktrie

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
	assert.Equal(t, "3998087801436302712617435196225481036627874106324392591598072448097460358227", secureKey.String())
}

func TestToSecureKeyBytes(t *testing.T) {
	secureKeyBytes, err := ToSecureKeyBytes([]byte("testKey"))
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x8, 0xd6, 0xd6, 0x66, 0xa4, 0x8, 0xc5, 0x72, 0xa0, 0xc3, 0x71, 0x50, 0x89, 0xa0, 0x2b, 0xe7, 0x59, 0x97, 0x39, 0x5d, 0x2c, 0x37, 0x38, 0x5d, 0x67, 0x22, 0x84, 0xe5, 0xc8, 0xbf, 0xc, 0x53}, secureKeyBytes.Bytes())
}

func TestReverseByteOrder(t *testing.T) {
	assert.Equal(t, []byte{5, 4, 3, 2, 1}, ReverseByteOrder([]byte{1, 2, 3, 4, 5}))
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
	assert.Equal(t, "1613b67f0a90f864bafa14df215f89e0c5a1c128e54561f0d730d112678e981d", result.Hex())
}

func TestPreHandlingElems(t *testing.T) {
	flagArray := uint32(0b10101010101010101010101010101010)
	elems := make([]Byte32, 32)
	for i := range elems {
		elems[i] = *NewByte32FromBytes([]byte("test" + strconv.Itoa(i+1)))
	}

	result, err := HandlingElemsAndByte32(flagArray, elems)
	assert.NoError(t, err)
	assert.Equal(t, "259503a5495e5e7e83d7e8e3f22b214092f921b7cadba00526aea7485c1997e7", result.Hex())

	elems = elems[:1]
	result, err = HandlingElemsAndByte32(flagArray, elems)
	assert.NoError(t, err)
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000007465737431", result.Hex())
}
