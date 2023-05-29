package zktrie

import (
	itrie "github.com/scroll-tech/zktrie/trie"
	itypes "github.com/scroll-tech/zktrie/types"
)

func binaryToCompact(b []byte) []byte {
	compact := make([]byte, 0, (len(b)+7)/8+1)
	compact = append(compact, byte(len(b)%8))
	var v byte
	for i := 0; i < len(b); i += 8 {
		v = 0
		for j := 0; j < 8 && i+j < len(b); j++ {
			v |= b[i+j] << (7 - j)
		}
		compact = append(compact, v)
	}
	return compact
}

func compactToBinary(c []byte) []byte {
	remainder := int(c[0])
	b := make([]byte, 0, (len(c)-1)*8+remainder)
	for i, cc := range c {
		if i == 0 {
			continue
		}
		num := 8
		if i+1 == len(c) && remainder > 0 {
			num = remainder
		}
		for j := 0; j < num; j++ {
			b = append(b, (cc>>(7-j))&1)
		}
	}
	return b
}

func keybytesToBinary(b []byte) []byte {
	d := make([]byte, 0, 8*len(b))
	for i := 0; i < len(b); i++ {
		for j := 0; j < 8; j++ {
			if b[i]&(1<<(7-j)) == 0 {
				d = append(d, 0)
			} else {
				d = append(d, 1)
			}
		}
	}
	return d
}

func binaryToKeybytes(b []byte) []byte {
	if len(b)%8 != 0 {
		panic("can't convert binary key whose size is not multiple of 8")
	}
	d := make([]byte, len(b)/8)
	for i := 0; i < len(b)/8; i++ {
		d[i] = 0
		for j := 0; j < 8; j++ {
			d[i] |= b[i*8+j] << (7 - j)
		}
	}
	return d
}

func reverseBitInPlace(b []byte) {
	var v [8]uint8
	for i := 0; i < len(b); i++ {
		for j := 0; j < 8; j++ {
			v[j] = (b[i] >> j) & 1
		}
		var tmp uint8 = 0
		for j := 0; j < 8; j++ {
			tmp |= v[8-j-1] << j
		}
		b[i] = tmp
	}
}

// internal trie hash key related method

func keybytesToHashKey(b []byte) *itypes.Hash {
	var h itypes.Hash
	copy(h[:], b)
	reverseBitInPlace(h[:])
	return &h
}

func keybytesToHashKeyAndCheck(b []byte) (*itypes.Hash, error) {
	h := keybytesToHashKey(b)
	if !itypes.CheckBigIntInField(h.BigInt()) {
		return nil, itrie.ErrInvalidField
	}
	return h, nil
}

func hashKeyToKeybytes(h *itypes.Hash) []byte {
	b := make([]byte, itypes.HashByteLen)
	copy(b, h[:])
	reverseBitInPlace(b)
	return b
}

func hashKeyToBinary(h *itypes.Hash) []byte {
	kb := hashKeyToKeybytes(h)
	return keybytesToBinary(kb)
}
