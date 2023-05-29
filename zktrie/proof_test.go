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

package zktrie

import (
	"bytes"
	crand "crypto/rand"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

// makeProvers creates Merkle trie provers based on different implementations to
// test all variations.
func makeTrieProvers(tr *Trie) []func(key []byte) *memorydb.Database {
	var provers []func(key []byte) *memorydb.Database

	// Create a direct trie based Merkle prover
	provers = append(provers, func(key []byte) *memorydb.Database {
		proof := memorydb.New()
		err := tr.Prove(key, 0, proof)
		if err != nil {
			panic(err)
		}

		return proof
	})
	return provers
}

func makeSecureTrieProvers(tr *SecureTrie) []func(key []byte) *memorydb.Database {
	var provers []func(key []byte) *memorydb.Database

	// Create a direct trie based Merkle prover
	provers = append(provers, func(key []byte) *memorydb.Database {
		proof := memorydb.New()
		err := tr.Prove(key, 0, proof)
		if err != nil {
			panic(err)
		}

		return proof
	})
	return provers
}

func verifyValue(proveVal []byte, vPreimage []byte) bool {
	return bytes.Equal(proveVal, vPreimage)
}

func TestTrieOneElementProof(t *testing.T) {
	tr, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	key := zkt.NewByte32FromBytesPaddingZero(append(bytes.Repeat([]byte("k"), 10), bytes.Repeat([]byte("l"), 21)...)).Bytes()
	err := tr.TryUpdate(key, bytes.Repeat([]byte("v"), 32))
	assert.Nil(t, err)
	for i, prover := range makeTrieProvers(tr) {
		proof := prover(key)
		if proof == nil {
			t.Fatalf("prover %d: nil proof", i)
		}
		if proof.Len() != 2 {
			t.Errorf("prover %d: proof should have 1+1 element (including the magic kv)", i)
		}
		val, err := VerifyProof(tr.Hash(), key, proof)
		if err != nil {
			t.Fatalf("prover %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if !verifyValue(val, bytes.Repeat([]byte("v"), 32)) {
			t.Fatalf("prover %d: verified value mismatch: want 'v' get %x", i, val)
		}
	}
}

func TestSecureTrieOneElementProof(t *testing.T) {
	tr, _ := NewSecure(common.Hash{}, NewDatabase(memorydb.New()))
	key := zkt.NewByte32FromBytesPaddingZero(append(bytes.Repeat([]byte("k"), 10), bytes.Repeat([]byte("l"), 21)...)).Bytes()
	err := tr.TryUpdate(key, bytes.Repeat([]byte("v"), 32))
	assert.Nil(t, err)
	for i, prover := range makeSecureTrieProvers(tr) {
		secureKey := toSecureKey(key)
		proof := prover(secureKey)
		if proof == nil {
			t.Fatalf("prover %d: nil proof", i)
		}
		if proof.Len() != 2 {
			t.Errorf("prover %d: proof should have 1+1 element (including the magic kv)", i)
		}
		val, err := VerifyProof(tr.Hash(), secureKey, proof)
		if err != nil {
			t.Fatalf("prover %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if !verifyValue(val, bytes.Repeat([]byte("v"), 32)) {
			t.Fatalf("prover %d: verified value mismatch: want 'v' get %x", i, val)
		}
	}
}

func TestTrieProof(t *testing.T) {
	tr, vals := randomTrie(t, 500)
	root := tr.Hash()
	for i, prover := range makeTrieProvers(tr) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: missing key %x while constructing proof", i, kv.k)
			}
			val, err := VerifyProof(common.BytesToHash(root.Bytes()), kv.k, proof)
			if err != nil {
				t.Fatalf("prover %d: failed to verify proof for key %x: %v\nraw proof: %x\n", i, kv.k, err, proof)
			}
			if !verifyValue(val, zkt.NewByte32FromBytesPaddingZero(kv.v)[:]) {
				t.Fatalf("prover %d: verified value mismatch for key %x, want %x, get %x", i, kv.k, kv.v, val)
			}
		}
	}
}

func TestSecureTrieProof(t *testing.T) {
	tr, vals := randomSecureTrie(t, 500)
	root := tr.Hash()
	for i, prover := range makeSecureTrieProvers(tr) {
		for _, kv := range vals {
			secureKey := toSecureKey(kv.k)
			proof := prover(secureKey)
			if proof == nil {
				t.Fatalf("prover %d: missing key %x while constructing proof", i, secureKey)
			}
			val, err := VerifyProof(common.BytesToHash(root.Bytes()), secureKey, proof)
			if err != nil {
				t.Fatalf("prover %d: failed to verify proof for key %x: %v\nraw proof: %x\n", i, secureKey, err, proof)
			}
			if !verifyValue(val, zkt.NewByte32FromBytesPaddingZero(kv.v)[:]) {
				t.Fatalf("prover %d: verified value mismatch for key %x, want %x, get %x", i, secureKey, kv.v, val)
			}
		}
	}
}

func TestTrieBadProof(t *testing.T) {
	tr, vals := randomTrie(t, 500)
	for i, prover := range makeTrieProvers(tr) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: nil proof", i)
			}
			it := proof.NewIterator(nil, nil)
			for i, d := 0, mrand.Intn(proof.Len()-1); i <= d; i++ {
				it.Next()
			}

			// Need to randomly mutate two keys, as magic kv in Proof is not used in verifyProof
			for i := 0; i <= 2; i++ {
				key := it.Key()
				proof.Delete(key)
				it.Next()
			}
			it.Release()

			if _, err := VerifyProof(tr.Hash(), kv.k, proof); err == nil {
				t.Fatalf("prover %d: expected proof to fail for key %x", i, kv.k)
			}
		}
	}
}

func TestSecureTrieBadProof(t *testing.T) {
	tr, vals := randomSecureTrie(t, 500)
	for i, prover := range makeSecureTrieProvers(tr) {
		for _, kv := range vals {
			secureKey := toSecureKey(kv.k)
			proof := prover(secureKey)
			if proof == nil {
				t.Fatalf("prover %d: nil proof", i)
			}
			it := proof.NewIterator(nil, nil)
			for i, d := 0, mrand.Intn(proof.Len()-1); i <= d; i++ {
				it.Next()
			}

			// Need to randomly mutate two keys, as magic kv in Proof is not used in verifyProof
			for i := 0; i <= 2; i++ {
				key := it.Key()
				proof.Delete(key)
				it.Next()
			}
			it.Release()

			if _, err := VerifyProof(tr.Hash(), secureKey, proof); err == nil {
				t.Fatalf("prover %d: expected proof to fail for key %x", i, secureKey)
			}
		}
	}
}

// Tests that missing keys can also be proven. The test explicitly uses a single
// entry trie and checks for missing keys both before and after the single entry.
func TestTrieMissingKeyProof(t *testing.T) {
	tr, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	key := zkt.NewByte32FromBytesPaddingZero(append(bytes.Repeat([]byte("k"), 10), bytes.Repeat([]byte("l"), 21)...)).Bytes()
	err := tr.TryUpdate(key, bytes.Repeat([]byte("v"), 32))
	assert.Nil(t, err)

	prover := makeTrieProvers(tr)[0]

	for i, key := range []string{"a", "j", "l", "z"} {
		keyBytes := bytes.Repeat([]byte(key), 32)
		proof := prover(keyBytes)

		if proof.Len() != 2 {
			t.Errorf("test %d: proof should have 2 element (with magic kv)", i)
		}
		val, err := VerifyProof(tr.Hash(), keyBytes, proof)
		if err != nil {
			t.Fatalf("test %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if val != nil {
			t.Fatalf("test %d: verified value mismatch: have %x, want nil", i, val)
		}
	}
}

func TestSecureTrieMissingKeyProof(t *testing.T) {
	tr, _ := NewSecure(common.Hash{}, NewDatabase(memorydb.New()))
	key := zkt.NewByte32FromBytesPaddingZero(append(bytes.Repeat([]byte("k"), 10), bytes.Repeat([]byte("l"), 21)...)).Bytes()
	err := tr.TryUpdate(key, bytes.Repeat([]byte("v"), 32))
	assert.Nil(t, err)

	prover := makeSecureTrieProvers(tr)[0]

	for i, key := range []string{"a", "j", "l", "z"} {
		keyBytes := bytes.Repeat([]byte(key), 32)
		secureKey := toSecureKey(keyBytes)
		proof := prover(secureKey)

		if proof.Len() != 2 {
			t.Errorf("test %d: proof should have 2 element (with magic kv)", i)
		}
		val, err := VerifyProof(tr.Hash(), secureKey, proof)
		if err != nil {
			t.Fatalf("test %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if val != nil {
			t.Fatalf("test %d: verified value mismatch: have %x, want nil", i, val)
		}
	}
}

// Tests that new "proof trace" feature
func TestProofWithDeletion(t *testing.T) {
	secure, _ := NewSecure(common.Hash{}, NewDatabase((memorydb.New())))
	//mt := &zkTrieImplTestWrapper{tr.Tree()}
	key1 := bytes.Repeat([]byte("k"), 32)
	key2 := bytes.Repeat([]byte("m"), 32)

	err := secure.TryUpdate(key1, bytes.Repeat([]byte("v"), 32))
	assert.NoError(t, err)
	err = secure.TryUpdate(key2, bytes.Repeat([]byte("v"), 32))
	assert.NoError(t, err)

	proof := memorydb.New()
	s_key1, err := zkt.ToSecureKeyBytes(key1)
	assert.NoError(t, err)

	proofTracer := secure.NewProofTracer()

	err = proofTracer.Prove(s_key1.Bytes(), 0, proof)
	assert.NoError(t, err)
	nd, err := secure.TryGet(key2)
	assert.NoError(t, err)

	s_key2, err := zkt.ToSecureKeyBytes(bytes.Repeat([]byte("x"), 32))
	assert.NoError(t, err)

	err = proofTracer.Prove(s_key2.Bytes(), 0, proof)
	assert.NoError(t, err)
	// assert.Equal(t, len(sibling1), len(delTracer.GetProofs()))

	siblings, err := proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(siblings))

	proofTracer.MarkDeletion(s_key1.Bytes())
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))
	l := len(siblings[0])
	// a hacking to grep the value part directly from the encoded leaf node,
	// notice the sibling of key `k*32`` is just the leaf of key `m*32`
	assert.Equal(t, siblings[0][l-33:l-1], nd)

	// Marking a key that is currently not hit (but terminated by an empty node)
	// also causes it to be added to the deletion proof
	proofTracer.MarkDeletion(s_key2.Bytes())
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(siblings))

	key3 := bytes.Repeat([]byte("x"), 32)
	err = secure.TryUpdate(key3, bytes.Repeat([]byte("z"), 32))
	assert.NoError(t, err)

	proofTracer = secure.NewProofTracer()
	err = proofTracer.Prove(s_key1.Bytes(), 0, proof)
	assert.NoError(t, err)
	err = proofTracer.Prove(s_key2.Bytes(), 0, proof)
	assert.NoError(t, err)

	proofTracer.MarkDeletion(s_key1.Bytes())
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))

	proofTracer.MarkDeletion(s_key2.Bytes())
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

func TestSecureTrieProofWithDeletion(t *testing.T) {
	tr, _ := NewSecure(common.Hash{}, NewDatabase((memorydb.New())))
	key1 := zkt.NewByte32FromBytesPaddingZero(append(bytes.Repeat([]byte("k"), 10), bytes.Repeat([]byte("l"), 21)...)).Bytes()
	key2 := zkt.NewByte32FromBytesPaddingZero(append(bytes.Repeat([]byte("m"), 10), bytes.Repeat([]byte("n"), 21)...)).Bytes()
	secureKey1 := toSecureKey(key1)

	err := tr.TryUpdate(key1, bytes.Repeat([]byte("v"), 32))
	assert.NoError(t, err)
	err = tr.TryUpdate(key2, bytes.Repeat([]byte("v"), 32))
	assert.NoError(t, err)

	proof := memorydb.New()
	assert.NoError(t, err)

	sibling1, err := tr.ProveWithDeletion(secureKey1, 0, proof)
	assert.NoError(t, err)
	nd, err := tr.TryGet(key2)
	assert.NoError(t, err)
	l := len(sibling1)
	// a hacking to grep the value part directly from the encoded leaf node,
	// notice the sibling of key1 is just the leaf of key2
	assert.Equal(t, sibling1[l-33:l-1], nd)

	notKey := zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte{'x'}, 31)).Bytes()
	sibling2, err := tr.ProveWithDeletion(notKey, 0, proof)
	assert.NoError(t, err)
	assert.Nil(t, sibling2)
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	crand.Read(r)
	return r
}

func randomTrie(t *testing.T, n int) (*Trie, map[string]*kv) {
	tr, err := New(common.Hash{}, NewDatabase((memorydb.New())))
	if err != nil {
		panic(err)
	}
	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {

		value := &kv{zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte{i}, 31)).Bytes(), bytes.Repeat([]byte{i}, 32), false}
		value2 := &kv{zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte{i + 10}, 31)).Bytes(), bytes.Repeat([]byte{i + 5}, 32), false}

		err = tr.TryUpdate(value.k, value.v)
		assert.Nil(t, err)
		err = tr.TryUpdate(value2.k, value2.v)
		assert.Nil(t, err)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{zkt.NewByte32FromBytesPaddingZero(randBytes(31)).Bytes(), randBytes(32), false}
		err = tr.TryUpdate(value.k, value.v)
		assert.Nil(t, err)
		vals[string(value.k)] = value
	}

	return tr, vals
}

func randomSecureTrie(t *testing.T, n int) (*SecureTrie, map[string]*kv) {
	tr, err := NewSecure(common.Hash{}, NewDatabase((memorydb.New())))
	if err != nil {
		panic(err)
	}
	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {

		value := &kv{zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte{i}, 31)).Bytes(), bytes.Repeat([]byte{i}, 32), false}
		value2 := &kv{zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte{i + 10}, 31)).Bytes(), bytes.Repeat([]byte{i + 5}, 32), false}

		err = tr.TryUpdate(value.k, value.v)
		assert.Nil(t, err)
		err = tr.TryUpdate(value2.k, value2.v)
		assert.Nil(t, err)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{zkt.NewByte32FromBytesPaddingZero(randBytes(31)).Bytes(), randBytes(32), false}
		err = tr.TryUpdate(value.k, value.v)
		assert.Nil(t, err)
		vals[string(value.k)] = value
	}

	return tr, vals
}

func BenchmarkProveSecureTrie(b *testing.B) {
	t := new(testing.T)
	trie, vals := randomSecureTrie(t, 4096)
	var keys []string
	for k := range vals {
		keys = append(keys, k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kv := vals[keys[i%len(keys)]]
		proofs := memorydb.New()
		if err := trie.Prove(kv.k, 0, proofs); err != nil || proofs.Len() == 0 {
			b.Fatalf("zero length proof for %x", kv.k)
		}
	}
}
