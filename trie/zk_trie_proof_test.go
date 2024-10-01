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
	"crypto/rand"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

// makeProvers creates Merkle trie provers based on different implementations to
// test all variations.
func makeSMTProvers(mt *ZkTrie) []func(key []byte) *memorydb.Database {
	var provers []func(key []byte) *memorydb.Database

	// Create a direct trie based Merkle prover
	provers = append(provers, func(key []byte) *memorydb.Database {
		proofDB := memorydb.New()
		err := mt.Prove(key, proofDB)
		if err != nil {
			panic(err)
		}

		return proofDB
	})
	return provers
}

func verifyValue(proveVal []byte, vPreimage []byte) bool {
	return bytes.Equal(proveVal, vPreimage)
}

func TestSMTOneElementProof(t *testing.T) {
	mt, _ := newTestingMerkle(t)
	err := mt.TryUpdate(
		NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("k"), 32)).Bytes(),
		1,
		[]Byte32{*NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("v"), 32))},
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

		root, err := mt.Root()
		assert.NoError(t, err)

		val, err := VerifyProofSMT(common.BytesToHash(root.Bytes()), keyBytes, proof)
		if err != nil {
			t.Fatalf("prover %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if !verifyValue(val, bytes.Repeat([]byte("v"), 32)) {
			t.Fatalf("prover %d: verified value mismatch: want 'v' get %x", i, val)
		}
	}
}

func TestSMTProof(t *testing.T) {
	mt, vals := randomZktrie(t, 500)
	root, err := mt.Root()
	assert.NoError(t, err)

	for i, prover := range makeSMTProvers(mt) {
		for kStr, v := range vals {
			k := []byte(kStr)
			proof := prover(k)
			if proof == nil {
				t.Fatalf("prover %d: missing key %x while constructing proof", i, k)
			}
			val, err := VerifyProofSMT(common.BytesToHash(root.Bytes()), k, proof)
			if err != nil {
				t.Fatalf("prover %d: failed to verify proof for key %x: %v\nraw proof: %x\n", i, k, err, proof)
			}
			if !verifyValue(val, NewByte32FromBytesPaddingZero(v)[:]) {
				t.Fatalf("prover %d: verified value mismatch for key %x, want %x, get %x", i, k, v, val)
			}
		}
	}
}

func TestSMTBadProof(t *testing.T) {
	mt, vals := randomZktrie(t, 500)
	root, err := mt.Root()
	assert.NoError(t, err)

	for i, prover := range makeSMTProvers(mt) {
		for kStr, _ := range vals {
			k := []byte(kStr)
			proof := prover(k)
			if proof == nil {
				t.Fatalf("prover %d: nil proof", i)
			}
			it := proof.NewIterator(nil, nil)
			for i, d := 0, mrand.Intn(proof.Len()-1); i <= d; i++ {
				it.Next()
			}
			if bytes.Equal(it.Key(), magicHash) {
				it.Next()
			}

			key := it.Key()
			proof.Delete(key)
			it.Release()

			if value, err := VerifyProof(common.BytesToHash(root.Bytes()), k, proof); err == nil && value != nil {
				t.Fatalf("prover %d: expected proof to fail for key %x", i, k)
			}
		}
	}
}

// Tests that missing keys can also be proven. The test explicitly uses a single
// entry trie and checks for missing keys both before and after the single entry.
func TestSMTMissingKeyProof(t *testing.T) {
	mt, _ := newTestingMerkle(t)
	err := mt.TryUpdate(
		NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("k"), 32)).Bytes(),
		1,
		[]Byte32{*NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("v"), 32))},
	)
	assert.Nil(t, err)

	prover := makeSMTProvers(mt)[0]

	for i, key := range []string{"a", "j", "l", "z"} {
		keyBytes := bytes.Repeat([]byte(key), 32)
		proof := prover(keyBytes)

		if proof.Len() != 2 {
			t.Errorf("test %d: proof should have 2 element (with magic kv)", i)
		}

		root, err := mt.Root()
		assert.NoError(t, err)

		val, err := VerifyProofSMT(common.BytesToHash(root.Bytes()), keyBytes, proof)
		if err != nil {
			t.Fatalf("test %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if val != nil {
			t.Fatalf("test %d: verified value mismatch: have %x, want nil", i, val)
		}
	}
}

func randomZktrie(t *testing.T, n int) (*ZkTrie, map[string][]byte) {
	randBytes := func(len int) []byte {
		buf := make([]byte, len)
		if n, err := rand.Read(buf); n != len || err != nil {
			panic(err)
		}
		return buf
	}

	mt, _ := newTestingMerkle(t)
	vals := make(map[string][]byte)
	for i := byte(0); i < 100; i++ {

		key, value := common.LeftPadBytes([]byte{i}, 32), NewByte32FromBytes(bytes.Repeat([]byte{i}, 32))
		key2, value2 := common.LeftPadBytes([]byte{i + 10}, 32), NewByte32FromBytes(bytes.Repeat([]byte{i}, 32))

		err := mt.TryUpdate(key, 1, []Byte32{*value})
		assert.Nil(t, err)
		err = mt.TryUpdate(key2, 1, []Byte32{*value2})
		assert.Nil(t, err)
		vals[string(key)] = value.Bytes()
		vals[string(key2)] = value2.Bytes()
	}
	for i := 0; i < n; i++ {
		key, value := randBytes(32), NewByte32FromBytes(randBytes(20))
		err := mt.TryUpdate(key, 1, []Byte32{*value})
		assert.Nil(t, err)
		vals[string(key)] = value.Bytes()
	}

	return mt, vals
}

// Tests that new "proof trace" feature
func TestProofWithDeletion(t *testing.T) {
	mt, _ := newTestingMerkle(t)
	key1 := bytes.Repeat([]byte("b"), 32)
	key2 := bytes.Repeat([]byte("c"), 32)
	err := mt.TryUpdate(
		key1,
		1,
		[]Byte32{*NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("v"), 32))},
	)
	assert.NoError(t, err)
	err = mt.TryUpdate(
		key2,
		1,
		[]Byte32{*NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("n"), 32))},
	)
	assert.NoError(t, err)

	proof := memorydb.New()
	proofTracer := mt.NewProofTracer()

	err = proofTracer.Prove(key1, proof)
	assert.NoError(t, err)
	nd, err := mt.TryGet(key2)
	assert.NoError(t, err)

	key4 := bytes.Repeat([]byte("x"), 32)
	err = proofTracer.Prove(key4, proof)
	assert.NoError(t, err)
	//assert.Equal(t, len(sibling1), len(delTracer.GetProofs()))

	siblings, err := proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(siblings))

	proofTracer.MarkDeletion(key1)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))
	l := len(siblings[0])
	// a hacking to grep the value part directly from the encoded leaf node,
	// notice the sibling of key `k*32`` is just the leaf of key `m*32`
	assert.Equal(t, siblings[0][l-33:l-1], nd)

	// Marking a key that is currently not hit (but terminated by an empty node)
	// also causes it to be added to the deletion proof
	proofTracer.MarkDeletion(key4)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(siblings))

	key3 := bytes.Repeat([]byte("x"), 32)
	err = mt.TryUpdate(
		key3,
		1,
		[]Byte32{*NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("z"), 32))},
	)
	assert.NoError(t, err)

	proofTracer = mt.NewProofTracer()
	err = proofTracer.Prove(key1, proof)
	assert.NoError(t, err)
	err = proofTracer.Prove(key4, proof)
	assert.NoError(t, err)

	proofTracer.MarkDeletion(key1)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))

	proofTracer.MarkDeletion(key4)
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(siblings))

	// one of the siblings is just leaf for key2, while
	// another one must be a middle node
	match1 := bytes.Equal(siblings[0][l-33:l-1], nd)
	match2 := bytes.Equal(siblings[1][l-33:l-1], nd)
	assert.True(t, match1 || match2)
	assert.False(t, match1 && match2)
}
