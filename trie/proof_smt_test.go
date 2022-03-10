// Copyright 2015 The go-ethereum Authors
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

package trie

import (
	"bytes"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types/smt"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/trie/db"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

// makeProvers creates Merkle trie provers based on different implementations to
// test all variations.
func makeSMTProvers(mt *MerkleTree) []func(key []byte) *memorydb.Database {
	var provers []func(key []byte) *memorydb.Database

	// Create a direct trie based Merkle prover
	provers = append(provers, func(key []byte) *memorydb.Database {
		word := smt.NewByte32FromBytesPaddingZero(key)
		k, err := word.Hash()
		if err != nil {
			panic(err)
		}
		proof := memorydb.New()
		mt.Prove(k, 0, proof)
		return proof
	})
	return provers
}

func TestSMTProof(t *testing.T) {
	mt, vals := randomSMT(500)
	root := mt.Root()
	for i, prover := range makeSMTProvers(mt) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: missing key %x while constructing proof", i, kv.k)
			}
			val, err := VerifyProof(common.BytesToHash(root.Bytes()), kv.k, proof)
			if err != nil {
				t.Fatalf("prover %d: failed to verify proof for key %x: %v\nraw proof: %x\n", i, kv.k, err, proof)
			}
			hv, err := smt.NewByte32FromBytesPaddingZero(kv.v).Hash()
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(val, hv.Bytes()) {
				t.Fatalf("prover %d: verified value mismatch for key %x: have %x, want %x", i, kv.k, val, hv.Bytes())
			}
		}
	}
}

func randomSMT(n int) (*MerkleTree, map[string]*kv) {
	mt, err := NewMerkleTree(db.NewEthKVStorage(memorydb.New()), 64)
	if err != nil {
		panic(err)
	}
	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {

		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), []byte{i}, false}

		mt.UpdateWord(smt.NewByte32FromBytesPaddingZero(value.k), smt.NewByte32FromBytesPaddingZero(value.v))
		mt.UpdateWord(smt.NewByte32FromBytesPaddingZero(value2.k), smt.NewByte32FromBytesPaddingZero(value2.v))
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		mt.UpdateWord(smt.NewByte32FromBytesPaddingZero(value.k), smt.NewByte32FromBytesPaddingZero(value.v))
		vals[string(value.k)] = value
	}

	return mt, vals
}

func TestKeyHash(t *testing.T) {

	vals := make(map[string]int)
	for i := 0; i < 110; i++ {

		k := common.LeftPadBytes([]byte{byte(i)}, 32)
		h, err := smt.NewByte32FromBytesPaddingZero(k).Hash()
		if err != nil {
			t.Fatal(err)
		}
		kHash := smt.NewHashFromBigInt(h)
		ks := kHash.Hex()[60:]
		if v, existed := vals[ks]; existed {
			t.Fatalf("duplicated of hash %s (%v with %v)", ks, v, i)
		}
		vals[ks] = i
	}
	t.Fatalf("always fail %v", vals)
}
