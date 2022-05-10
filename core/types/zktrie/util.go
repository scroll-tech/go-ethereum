package zktrie

import (
	"math/big"

	"github.com/iden3/go-iden3-crypto/poseidon"
)

// HashElems performs a poseidon hash over the array of ElemBytes, currently we
// are using 2 elements.  Uses poseidon.Hash to be compatible with the circom
// circuits implementations.
func HashElems(elems ...*big.Int) (*Hash, error) {
	poseidonHash, err := poseidon.Hash(elems)
	if err != nil {
		return nil, err
	}
	return NewHashFromBigInt(poseidonHash), nil
}

// HashElemsKey performs a poseidon hash over the array of ElemBytes, currently
// we are using 2 elements.
func HashElemsKey(key *big.Int, elems ...*big.Int) (*Hash, error) {
	if key == nil {
		key = new(big.Int).SetInt64(0)
	}
	bi := make([]*big.Int, 3)
	copy(bi[:], elems)
	bi[2] = key
	poseidonHash, err := poseidon.Hash(bi)
	if err != nil {
		return nil, err
	}
	return NewHashFromBigInt(poseidonHash), nil
}

// SetBitBigEndian sets the bit n in the bitmap to 1, in Big Endian.
func SetBitBigEndian(bitmap []byte, n uint) {
	bitmap[uint(len(bitmap))-n/8-1] |= 1 << (n % 8)
}

// TestBit tests whether the bit n in bitmap is 1.
func TestBit(bitmap []byte, n uint) bool {
	return bitmap[n/8]&(1<<(n%8)) != 0
}

// TestBitBigEndian tests whether the bit n in bitmap is 1, in Big Endian.
func TestBitBigEndian(bitmap []byte, n uint) bool {
	return bitmap[uint(len(bitmap))-n/8-1]&(1<<(n%8)) != 0
}

var BigOne = big.NewInt(1)
var BigZero = big.NewInt(0)

func BigEndianBitsToBigInt(bits []bool) *big.Int {
	result := big.NewInt(0)
	for _, bit := range bits {
		result.Mul(result, big.NewInt(2))
		if bit {
			result.Add(result, BigOne)
		}
	}
	return result
}
