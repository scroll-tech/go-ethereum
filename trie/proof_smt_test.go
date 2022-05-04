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

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types/smt"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
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
		word := smt.NewByte32FromBytesPadding(key)
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

func verifyValue(vHash []byte, vPreimage []byte) bool {
	hv, err := smt.NewByte32FromBytesPadding(vPreimage).Hash()
	if err != nil {
		panic(err)
	}
	return bytes.Equal(vHash, hv.Bytes())
}

func TestSMTOneElementProof(t *testing.T) {
	mt, _ := NewMerkleTree(NewEthKVStorage(NewDatabase(memorydb.New())), 64)
	_, err := mt.UpdateWord(
		smt.NewByte32FromBytesPadding(bytes.Repeat([]byte("k"), 32)),
		smt.NewByte32FromBytesPadding(bytes.Repeat([]byte("v"), 32)),
	)
	assert.Nil(t, err)
	for i, prover := range makeSMTProvers(mt) {
		keyBytes := bytes.Repeat([]byte("k"), 32)
		proof := prover(keyBytes)
		if proof == nil {
			t.Fatalf("prover %d: nil proof", i)
		}
		if proof.Len() != 2 {
			t.Errorf("prover %d: proof should have 1+1 element (including the magic kv)", i)
		}
		val, err := VerifyProof(common.BytesToHash(mt.Root().Bytes()), keyBytes, proof)
		if err != nil {
			t.Fatalf("prover %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if !verifyValue(val, bytes.Repeat([]byte("v"), 32)) {
			t.Fatalf("prover %d: verified value mismatch: want 'k'", i)
		}
	}
}

func TestSMTProof(t *testing.T) {
	mt, vals := randomSMT(t, 500)
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
			if !verifyValue(val, kv.v) {
				t.Fatalf("prover %d: verified value mismatch for key %x, want %x", i, kv.k, kv.v)
			}
		}
	}
}

func TestSMTBadProof(t *testing.T) {
	mt, vals := randomSMT(t, 500)
	root := mt.Root()
	for i, prover := range makeSMTProvers(mt) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: nil proof", i)
			}
			it := proof.NewIterator(nil, nil)
			for i, d := 0, mrand.Intn(proof.Len()); i <= d; i++ {
				it.Next()
			}
			key := it.Key()
			val, _ := proof.Get(key)
			proof.Delete(key)
			it.Release()

			mutateByte(val)
			proof.Put(crypto.Keccak256(val), val)

			if _, err := VerifyProof(common.BytesToHash(root.Bytes()), kv.k, proof); err == nil {
				t.Fatalf("prover %d: expected proof to fail for key %x", i, kv.k)
			}
		}
	}
}

// Tests that missing keys can also be proven. The test explicitly uses a single
// entry trie and checks for missing keys both before and after the single entry.
func TestSMTMissingKeyProof(t *testing.T) {
	mt, _ := NewMerkleTree(NewEthKVStorage(NewDatabase(memorydb.New())), 64)
	_, err := mt.UpdateWord(
		smt.NewByte32FromBytesPadding(bytes.Repeat([]byte("k"), 20)),
		smt.NewByte32FromBytesPadding(bytes.Repeat([]byte("v"), 20)),
	)
	assert.Nil(t, err)

	prover := makeSMTProvers(mt)[0]

	for i, key := range []string{"a", "j", "l", "z"} {
		keyBytes := bytes.Repeat([]byte(key), 32)
		proof := prover(keyBytes)

		if proof.Len() != 2 {
			t.Errorf("test %d: proof should have 2 element (with magic kv)", i)
		}
		val, err := VerifyProof(common.BytesToHash(mt.Root().Bytes()), keyBytes, proof)
		if err != nil {
			t.Fatalf("test %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if val != nil {
			t.Fatalf("test %d: verified value mismatch: have %x, want nil", i, val)
		}
	}
}

func randomSMT(t *testing.T, n int) (*MerkleTree, map[string]*kv) {
	mt, err := NewMerkleTree(NewEthKVStorage(NewDatabase(memorydb.New())), 64)
	if err != nil {
		panic(err)
	}
	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {

		value := &kv{common.LeftPadBytes([]byte{i}, 32), bytes.Repeat([]byte{i}, 32), false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), bytes.Repeat([]byte{i}, 32), false}

		_, err = mt.UpdateWord(smt.NewByte32FromBytesPadding(value.k), smt.NewByte32FromBytesPadding(value.v))
		assert.Nil(t, err)
		_, err = mt.UpdateWord(smt.NewByte32FromBytesPadding(value2.k), smt.NewByte32FromBytesPadding(value2.v))
		assert.Nil(t, err)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		_, err = mt.UpdateWord(smt.NewByte32FromBytesPadding(value.k), smt.NewByte32FromBytesPadding(value.v))
		assert.Nil(t, err)
		vals[string(value.k)] = value
	}

	return mt, vals
}
