// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package zktrie

import (
	"bytes"
	"testing"
)

func TestBinaryCompact(t *testing.T) {
	tests := []struct{ binary, compact []byte }{
		{binary: []byte{}, compact: []byte{0x00}},
		{binary: []byte{0}, compact: []byte{0x01, 0x00}},
		{binary: []byte{0, 1}, compact: []byte{0x02, 0x40}},
		{binary: []byte{0, 1, 1}, compact: []byte{0x03, 0x60}},
		{binary: []byte{0, 1, 1, 0}, compact: []byte{0x04, 0x60}},
		{binary: []byte{0, 1, 1, 0, 1}, compact: []byte{0x05, 0x68}},
		{binary: []byte{0, 1, 1, 0, 1, 0}, compact: []byte{0x06, 0x68}},
		{binary: []byte{0, 1, 1, 0, 1, 0, 1}, compact: []byte{0x07, 0x6a}},
		{binary: []byte{0, 1, 1, 0, 1, 0, 1, 0}, compact: []byte{0x00, 0x6a}},
		{binary: []byte{0, 1, 0, 1, 0, 1, 0, 1 /* 8 bit */, 0, 1, 1, 0}, compact: []byte{0x04, 0x55, 0x60}},
	}
	for _, test := range tests {
		if c := binaryToCompact(test.binary); !bytes.Equal(c, test.compact) {
			t.Errorf("binaryToCompact(%x) -> %x, want %x", test.binary, c, test.compact)
		}
		if h := compactToBinary(test.compact); !bytes.Equal(h, test.binary) {
			t.Errorf("compactToBinary(%x) -> %x, want %x", test.compact, h, test.binary)
		}
	}
}

func TestBinaryKeybytes(t *testing.T) {
	tests := []struct{ key, binary []byte }{
		{key: []byte{}, binary: []byte{}},
		{
			key:    []byte{0x5f},
			binary: []byte{0, 1, 0, 1 /**/, 1, 1, 1, 1},
		},
		{
			key:    []byte{0x12, 0x34},
			binary: []byte{0, 0, 0, 1 /**/, 0, 0, 1, 0 /**/, 0, 0, 1, 1 /**/, 0, 1, 0, 0 /**/},
		},
	}
	for _, test := range tests {
		if h := keybytesToBinary(test.key); !bytes.Equal(h, test.binary) {
			t.Errorf("keybytesToBinary(%x) -> %x, want %x", test.key, h, test.binary)
		}
		if k := binaryToKeybytes(test.binary); !bytes.Equal(k, test.key) {
			t.Errorf("binaryToKeybytes(%x) -> %x, want %x", test.binary, k, test.key)
		}
	}
}
