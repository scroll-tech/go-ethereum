package zktrie

import (
	"math/big"
)

// HashElemsWithDomain performs a recursive poseidon hash over the array of ElemBytes, each hash
// reduce 2 fieds into one, with a specified domain field which would be used in
// every recursiving call
func HashElemsWithDomain(domain, fst, snd *big.Int, elems ...*big.Int) (*Hash, error) {

	l := len(elems)
	baseH, err := hashScheme([]*big.Int{fst, snd}, domain)
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return NewHashFromBigInt(baseH), nil
	} else if l == 1 {
		return HashElemsWithDomain(domain, baseH, elems[0])
	}

	tmp := make([]*big.Int, (l+1)/2)
	for i := range tmp {
		if (i+1)*2 > l {
			tmp[i] = elems[i*2]
		} else {
			h, err := hashScheme(elems[i*2:(i+1)*2], domain)
			if err != nil {
				return nil, err
			}
			tmp[i] = h
		}
	}

	return HashElemsWithDomain(domain, baseH, tmp[0], tmp[1:]...)
}

// HashElems call HashElemsWithDomain with a domain of HASH_DOMAIN_ELEMS_BASE(256)*<element counts>
func HashElems(fst, snd *big.Int, elems ...*big.Int) (*Hash, error) {

	return HashElemsWithDomain(big.NewInt(int64(len(elems)*HASH_DOMAIN_ELEMS_BASE)+HASH_DOMAIN_BYTE32),
		fst, snd, elems...)
}

// HandlingElemsAndByte32 hash an arry mixed with field and byte32 elements, turn each byte32 into
// field elements first then calculate the hash with HashElems
func HandlingElemsAndByte32(flagArray uint32, elems []Byte32) (*Hash, error) {

	ret := make([]*big.Int, len(elems))
	var err error

	for i, elem := range elems {
		if flagArray&(1<<i) != 0 {
			ret[i], err = elem.Hash()
			if err != nil {
				return nil, err
			}
		} else {
			ret[i] = new(big.Int).SetBytes(elem[:])
		}
	}

	if len(ret) < 2 {
		return NewHashFromBigInt(ret[0]), nil
	}

	return HashElems(ret[0], ret[1], ret[2:]...)

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

// ToSecureKey turn the byte key into the integer represent of "secured" key
func ToSecureKey(key []byte) (*big.Int, error) {
	word := NewByte32FromBytesPaddingZero(key)
	return word.Hash()
}

// ToSecureKeyBytes turn the byte key into a 32-byte "secured" key, which represented a big-endian integer
func ToSecureKeyBytes(key []byte) (*Byte32, error) {
	k, err := ToSecureKey(key)
	if err != nil {
		return nil, err
	}

	return NewByte32FromBytes(k.Bytes()), nil
}

// ReverseByteOrder swaps the order of the bytes in the slice.
func ReverseByteOrder(b []byte) []byte {
	o := make([]byte, len(b))
	for i := range b {
		o[len(b)-1-i] = b[i]
	}
	return o
}
