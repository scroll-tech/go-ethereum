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
	"encoding/binary"
	"fmt"
	"math/big"
	mrand "math/rand"
	"sort"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rlp"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

type entrySlice []*kv

func (p entrySlice) Len() int           { return len(p) }
func (p entrySlice) Less(i, j int) bool { return bytes.Compare(p[i].k, p[j].k) < 0 }
func (p entrySlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Basic case to test the functionality of the main workflow.
func TestSimpleProofEntireTrie(t *testing.T) {
	trie, kvs := nonRandomTrie(3)
	var entries entrySlice
	for _, kv := range kvs {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	proof := memorydb.New()
	if err := trie.Prove(entries[0].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[2].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}

	var keys [][]byte
	var vals [][]byte
	for i := 0; i <= 2; i++ {
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", keys[0], keys[len(keys)-1], keys, vals, proof)
	if err != nil {
		t.Fatalf("Verification of range proof failed!\n%v\n", err)
	}
}

func TestSimpleProofEntireTrieAccountTrie(t *testing.T) {
	trie, kvs := nonRandomAccountTrie(3)
	var entries entrySlice
	for _, kv := range kvs {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	proof := memorydb.New()
	if err := trie.Prove(entries[0].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[2].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}

	var keys [][]byte
	var vals [][]byte
	for i := 0; i <= 2; i++ {
		keys = append(keys, entries[i].k)
		account, _ := types.UnmarshalStateAccount(entries[i].v)
		account_rlp, _ := rlp.EncodeToBytes(account)
		vals = append(vals, account_rlp)
	}
	_, err := VerifyRangeProof(trie.Hash(), "account", keys[0], keys[len(keys)-1], keys, vals, proof)
	if err != nil {
		t.Fatalf("Verification of range proof failed!\n%v\n", err)
	}
}

// Basic case to test the functionality of the main workflow.
func TestSimpleProofValidRange(t *testing.T) {
	trie, kvs := nonRandomTrie(7)
	var entries entrySlice
	for _, kv := range kvs {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	proof := memorydb.New()
	if err := trie.Prove(entries[2].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[5].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}

	var keys [][]byte
	var vals [][]byte
	for i := 2; i <= 5; i++ {
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", keys[0], keys[len(keys)-1], keys, vals, proof)
	if err != nil {
		t.Fatalf("Verification of range proof failed!\n%v\n", err)
	}
}

// Basic case to test the functionality of the main workflow.
func TestSimpleProofValidRangeAccountTrie(t *testing.T) {
	trie, kvs := nonRandomAccountTrie(7)
	var entries entrySlice
	for _, kv := range kvs {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	proof := memorydb.New()
	if err := trie.Prove(entries[2].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[5].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}

	var keys [][]byte
	var vals [][]byte
	for i := 2; i <= 5; i++ {
		keys = append(keys, entries[i].k)
		account, _ := types.UnmarshalStateAccount(entries[i].v)
		account_rlp, _ := rlp.EncodeToBytes(account)
		vals = append(vals, account_rlp)
	}
	_, err := VerifyRangeProof(trie.Hash(), "account", keys[0], keys[len(keys)-1], keys, vals, proof)
	if err != nil {
		t.Fatalf("Verification of range proof failed!\n%v\n", err)
	}
}

// TestRangeProof tests normal range proof with both edge proofs
// as the existent proof. The test cases are generated randomly.
func TestRangeProof(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)
	for i := 0; i < 2; i++ {
		startTime := time.Now()
		start := mrand.Intn(len(entries))
		end := mrand.Intn(len(entries)-start) + start + 1

		proof := memorydb.New()
		if err := trie.Prove(entries[start].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(entries[end-1].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		var keys [][]byte
		var vals [][]byte
		for i := start; i < end; i++ {
			keys = append(keys, entries[i].k)
			vals = append(vals, entries[i].v)
		}

		_, err := VerifyRangeProof(trie.Hash(), "storage", keys[0], keys[len(keys)-1], keys, vals, proof)
		if err != nil {
			t.Fatalf("Case %d(%d->%d) expect no error, got %v", i, start, end-1, err)
		}
		elapsed := time.Since(startTime)
		fmt.Printf("Case #%d: Elapsed time: %.2f seconds\n", i, elapsed.Seconds())
	}
}

// TestRangeProof tests normal range proof with two non-existent proofs.
// The test cases are generated randomly.
func TestRangeProofWithNonExistentProof(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)
	for i := 0; i < 2; i++ {
		start := mrand.Intn(len(entries))
		end := mrand.Intn(len(entries)-start) + start + 1
		proof := memorydb.New()

		// Short circuit if the decreased key is same with the previous key
		first := decreseKey(common.CopyBytes(entries[start].k))
		if start != 0 && bytes.Equal(first, entries[start-1].k) {
			continue
		}
		// Short circuit if the decreased key is underflow
		if bytes.Compare(first, entries[start].k) > 0 {
			continue
		}
		// Short circuit if the increased key is same with the next key
		last := increseKey(common.CopyBytes(entries[end-1].k))
		if end != len(entries) && bytes.Equal(last, entries[end].k) {
			continue
		}
		// Short circuit if the increased key is overflow
		if bytes.Compare(last, entries[end-1].k) < 0 {
			continue
		}
		if err := trie.Prove(first, 0, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(last, 0, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		var keys [][]byte
		var vals [][]byte
		for i := start; i < end; i++ {
			keys = append(keys, entries[i].k)
			vals = append(vals, entries[i].v)
		}
		_, err := VerifyRangeProof(trie.Hash(), "storage", first, last, keys, vals, proof)
		if err != nil {
			t.Fatalf("Case %d(%d->%d) expect no error, got %v", i, start, end-1, err)
		}
	}
	// Special case, two edge proofs for two edge key.
	proof := memorydb.New()
	first := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000").Bytes()
	last := common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff").Bytes()
	if err := trie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	var k [][]byte
	var v [][]byte
	for i := 0; i < len(entries); i++ {
		k = append(k, entries[i].k)
		v = append(v, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", first, last, k, v, proof)
	if err != nil {
		t.Fatal("Failed to verify whole rang with non-existent edges")
	}
}

// TestRangeProofWithInvalidNonExistentProof tests such scenarios:
// - There exists a gap between the first element and the left edge proof
// - There exists a gap between the last element and the right edge proof
func TestRangeProofWithInvalidNonExistentProof(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	// Case 1
	start, end := 100, 200
	first := decreseKey(common.CopyBytes(entries[start].k))

	proof := memorydb.New()
	if err := trie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[end-1].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	start = 105 // Gap created
	k := make([][]byte, 0)
	v := make([][]byte, 0)
	for i := start; i < end; i++ {
		k = append(k, entries[i].k)
		v = append(v, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", first, k[len(k)-1], k, v, proof)
	if err == nil {
		t.Fatalf("Expected to detect the error, got nil")
	}

	// Case 2
	start, end = 100, 200
	last := increseKey(common.CopyBytes(entries[end-1].k))
	proof = memorydb.New()
	if err := trie.Prove(entries[start].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	end = 195 // Capped slice
	k = make([][]byte, 0)
	v = make([][]byte, 0)
	for i := start; i < end; i++ {
		k = append(k, entries[i].k)
		v = append(v, entries[i].v)
	}
	_, err = VerifyRangeProof(trie.Hash(), "storage", k[0], last, k, v, proof)
	if err == nil {
		t.Fatalf("Expected to detect the error, got nil")
	}
}

// TestOneElementRangeProof tests the proof with only one
// element. The first edge proof can be existent one or
// non-existent one.
func TestOneElementRangeProof(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	// One element with existent edge proof, both edge proofs
	// point to the SAME key.
	start := 1000
	proof := memorydb.New()
	if err := trie.Prove(entries[start].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", entries[start].k, entries[start].k, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// One element with left non-existent edge proof
	start = 1000
	first := decreseKey(common.CopyBytes(entries[start].k))
	proof = memorydb.New()
	if err := trie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[start].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), "storage", first, entries[start].k, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// One element with right non-existent edge proof
	start = 1000
	last := increseKey(common.CopyBytes(entries[start].k))
	proof = memorydb.New()
	if err := trie.Prove(entries[start].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), "storage", entries[start].k, last, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// One element with two non-existent edge proofs
	start = 1000
	first, last = decreseKey(common.CopyBytes(entries[start].k)), increseKey(common.CopyBytes(entries[start].k))
	proof = memorydb.New()
	if err := trie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), "storage", first, last, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test the mini trie with only a single element.
	tinyTrie, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	entry := &kv{common.RightPadBytes(randBytes(31), 32), randBytes(20), false}
	tinyTrie.Update(entry.k, entry.v)

	first = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000").Bytes()
	last = entry.k
	proof = memorydb.New()
	if err := tinyTrie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := tinyTrie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}

	_, err = VerifyRangeProof(tinyTrie.Hash(), "storage", first, last, [][]byte{entry.k}, [][]byte{entry.v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestAllElementsProof tests the range proof with all elements.
// The edge proofs can be nil.
func TestAllElementsProof(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	var k [][]byte
	var v [][]byte
	for i := 0; i < len(entries); i++ {
		k = append(k, entries[i].k)
		v = append(v, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", nil, nil, k, v, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// With edge proofs, it should still work.
	proof := memorydb.New()
	if err := trie.Prove(entries[0].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[len(entries)-1].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), "storage", k[0], k[len(k)-1], k, v, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Even with non-existent edge proofs, it should still work.
	proof = memorydb.New()
	first := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000").Bytes()
	last := common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff").Bytes()
	if err := trie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), "storage", first, last, k, v, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestSingleSideRangeProof tests the range starts from zero.
func TestSingleSideRangeProof(t *testing.T) {
	for i := 0; i < 2; i++ {
		trie, vals := randomTrie(t, 4096)
		var entries entrySlice
		for _, kv := range vals {
			entries = append(entries, kv)
		}
		sort.Sort(entries)

		var cases = []int{0, 1, 50, 100, 1000, 2000, len(entries) - 1}
		for _, pos := range cases {
			proof := memorydb.New()
			if err := trie.Prove(common.Hash{}.Bytes(), 0, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
			if err := trie.Prove(entries[pos].k, 0, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
			k := make([][]byte, 0)
			v := make([][]byte, 0)
			for i := 0; i <= pos; i++ {
				k = append(k, entries[i].k)
				v = append(v, entries[i].v)
			}
			_, err := VerifyRangeProof(trie.Hash(), "storage", common.Hash{}.Bytes(), k[len(k)-1], k, v, proof)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		}
	}
}

// TestReverseSingleSideRangeProof tests the range ends with 0xffff...fff.
func TestReverseSingleSideRangeProof(t *testing.T) {
	for i := 0; i < 2; i++ {
		trie, vals := randomTrie(t, 4096)
		var entries entrySlice
		for _, kv := range vals {
			entries = append(entries, kv)
		}
		sort.Sort(entries)

		var cases = []int{0, 1, 50, 100, 1000, 2000, len(entries) - 1}
		for _, pos := range cases {
			proof := memorydb.New()
			if err := trie.Prove(entries[pos].k, 0, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
			last := common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
			if err := trie.Prove(last.Bytes(), 0, proof); err != nil {
				t.Fatalf("Failed to prove the last node %v", err)
			}
			k := make([][]byte, 0)
			v := make([][]byte, 0)
			for i := pos; i < len(entries); i++ {
				k = append(k, entries[i].k)
				v = append(v, entries[i].v)
			}
			_, err := VerifyRangeProof(trie.Hash(), "storage", k[0], last.Bytes(), k, v, proof)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		}
	}
}

// TestBadRangeProof tests a few cases which the proof is wrong.
// The prover is expected to detect the error.
func TestBadRangeProof(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	for i := 0; i < 2; i++ {
		start := mrand.Intn(len(entries))
		end := mrand.Intn(len(entries)-start) + start + 1
		proof := memorydb.New()
		if err := trie.Prove(entries[start].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(entries[end-1].k, 0, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		var keys [][]byte
		var vals [][]byte
		for i := start; i < end; i++ {
			keys = append(keys, entries[i].k)
			vals = append(vals, entries[i].v)
		}
		var first, last = keys[0], keys[len(keys)-1]
		testcase := mrand.Intn(6)
		var index int
		switch testcase {
		case 0:
			// Modified key
			index = mrand.Intn(end - start)
			keys[index] = common.RightPadBytes(randBytes(31), 32) // In theory it can't be same
		case 1:
			// Modified val
			index = mrand.Intn(end - start)
			vals[index] = randBytes(20) // In theory it can't be same
		case 2:
			// Gapped entry slice
			index = mrand.Intn(end - start)
			if (index == 0 && start < 100) || (index == end-start-1 && end <= 100) {
				continue
			}
			keys = append(keys[:index], keys[index+1:]...)
			vals = append(vals[:index], vals[index+1:]...)
		case 3:
			// Out of order
			index1 := mrand.Intn(end - start)
			index2 := mrand.Intn(end - start)
			if index1 == index2 {
				continue
			}
			keys[index1], keys[index2] = keys[index2], keys[index1]
			vals[index1], vals[index2] = vals[index2], vals[index1]
		case 4:
			// Set random key to nil, do nothing
			index = mrand.Intn(end - start)
			keys[index] = nil
		case 5:
			// Set random value to nil, deletion
			index = mrand.Intn(end - start)
			vals[index] = nil
		}
		_, err := VerifyRangeProof(trie.Hash(), "storage", first, last, keys, vals, proof)
		if err == nil {
			t.Fatalf("%d Case %d index %d range: (%d->%d) expect error, got nil", i, testcase, index, start, end-1)
		}
	}
}

// TestGappedRangeProof focuses on the small trie with embedded nodes.
// If the gapped node is embedded in the trie, it should be detected too.
func TestGappedRangeProof(t *testing.T) {
	trie, values := randomTrie(t, 10)
	var entries entrySlice
	for _, kv := range values {
		entries = append(entries, kv)
	}
	sort.Sort(entries)
	first, last := 2, 8
	proof := memorydb.New()
	if err := trie.Prove(entries[first].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[last-1].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	var keys [][]byte
	var vals [][]byte
	for i := first; i < last; i++ {
		if i == (first+last)/2 {
			continue
		}
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", keys[0], keys[len(keys)-1], keys, vals, proof)
	if err == nil {
		t.Fatal("expect error, got nil")
	}
}

// TestSameSideProofs tests the element is not in the range covered by proofs
func TestSameSideProofs(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	pos := 1000
	first := decreseKey(common.CopyBytes(entries[pos].k))
	first = decreseKey(first)
	last := decreseKey(common.CopyBytes(entries[pos].k))

	proof := memorydb.New()
	if err := trie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", first, last, [][]byte{entries[pos].k}, [][]byte{entries[pos].v}, proof)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	first = increseKey(common.CopyBytes(entries[pos].k))
	last = increseKey(common.CopyBytes(entries[pos].k))
	last = increseKey(last)

	proof = memorydb.New()
	if err := trie.Prove(first, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), "storage", first, last, [][]byte{entries[pos].k}, [][]byte{entries[pos].v}, proof)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestHasRightElement(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	var cases = []struct {
		start   int
		end     int
		hasMore bool
	}{
		{-1, 1, true}, // single element with non-existent left proof
		{0, 1, true},  // single element with existent left proof
		{0, 10, true},
		{50, 100, true},
		{50, len(entries), false},               // No more element expected
		{len(entries) - 1, len(entries), false}, // Single last element with two existent proofs(point to same key)
		{len(entries) - 1, -1, false},           // Single last element with non-existent right proof
		{0, len(entries), false},                // The whole set with existent left proof
		{-1, len(entries), false},               // The whole set with non-existent left proof
		{-1, -1, false},                         // The whole set with non-existent left/right proof
	}
	for _, c := range cases {
		var (
			firstKey []byte
			lastKey  []byte
			start    = c.start
			end      = c.end
			proof    = memorydb.New()
		)
		if c.start == -1 {
			firstKey, start = common.Hash{}.Bytes(), 0
			if err := trie.Prove(firstKey, 0, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
		} else {
			firstKey = entries[c.start].k
			if err := trie.Prove(entries[c.start].k, 0, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
		}
		if c.end == -1 {
			lastKey, end = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff").Bytes(), len(entries)
			if err := trie.Prove(lastKey, 0, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
		} else {
			lastKey = entries[c.end-1].k
			if err := trie.Prove(entries[c.end-1].k, 0, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
		}
		k := make([][]byte, 0)
		v := make([][]byte, 0)
		for i := start; i < end; i++ {
			k = append(k, entries[i].k)
			v = append(v, entries[i].v)
		}
		hasMore, err := VerifyRangeProof(trie.Hash(), "storage", firstKey, lastKey, k, v, proof)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if hasMore != c.hasMore {
			t.Fatalf("Wrong hasMore indicator, want %t, got %t", c.hasMore, hasMore)
		}
	}
}

// TestEmptyRangeProof tests the range proof with "no" element.
// The first edge proof must be a non-existent proof.
func TestEmptyRangeProof(t *testing.T) {
	trie, vals := randomTrie(t, 4096)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	var cases = []struct {
		pos int
		err bool
	}{
		{len(entries) - 1, false},
		{500, true},
	}
	for _, c := range cases {
		proof := memorydb.New()
		first := increseKey(common.CopyBytes(entries[c.pos].k))
		if err := trie.Prove(first, 0, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		_, err := VerifyRangeProof(trie.Hash(), "storage", first, nil, nil, nil, proof)
		if c.err && err == nil {
			t.Fatalf("Expected error, got nil")
		}
		if !c.err && err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}
}

// TestBloatedProof tests a malicious proof, where the proof is more or less the
// whole trie. Previously we didn't accept such packets, but the new APIs do, so
// lets leave this test as a bit weird, but present.
func TestBloatedProof(t *testing.T) {
	// Use a small trie
	trie, kvs := nonRandomTrie(100)
	var entries entrySlice
	for _, kv := range kvs {
		entries = append(entries, kv)
	}
	sort.Sort(entries)
	var keys [][]byte
	var vals [][]byte

	proof := memorydb.New()
	// In the 'malicious' case, we add proofs for every single item
	// (but only one key/value pair used as leaf)
	for i, entry := range entries {
		trie.Prove(entry.k, 0, proof)
		if i == 50 {
			keys = append(keys, entry.k)
			vals = append(vals, entry.v)
		}
	}
	// For reference, we use the same function, but _only_ prove the first
	// and last element
	want := memorydb.New()
	trie.Prove(keys[0], 0, want)
	trie.Prove(keys[len(keys)-1], 0, want)

	if _, err := VerifyRangeProof(trie.Hash(), "storage", keys[0], keys[len(keys)-1], keys, vals, proof); err != nil {
		t.Fatalf("expected bloated proof to succeed, got %v", err)
	}
}

// TestEmptyValueRangeProof tests normal range proof with both edge proofs
// as the existent proof, but with an extra empty value included, which is a
// noop technically, but practically should be rejected.
func TestEmptyValueRangeProof(t *testing.T) {
	trie, values := randomTrie(t, 512)
	var entries entrySlice
	for _, kv := range values {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	// Create a new entry with a slightly modified key
	mid := len(entries) / 2
	key := common.CopyBytes(entries[mid-1].k)
	for n := len(key) - 1; n >= 0; n-- {
		if key[n] < 0xff {
			key[n]++
			break
		}
	}
	noop := &kv{key, []byte{}, false}
	entries = append(append(append([]*kv{}, entries[:mid]...), noop), entries[mid:]...)

	start, end := 1, len(entries)-1

	proof := memorydb.New()
	if err := trie.Prove(entries[start].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[end-1].k, 0, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	var keys [][]byte
	var vals [][]byte
	for i := start; i < end; i++ {
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", keys[0], keys[len(keys)-1], keys, vals, proof)
	if err == nil {
		t.Fatalf("Expected failure on noop entry")
	}
}

// TestAllElementsEmptyValueRangeProof tests the range proof with all elements,
// but with an extra empty value included, which is a noop technically, but
// practically should be rejected.
func TestAllElementsEmptyValueRangeProof(t *testing.T) {
	trie, values := randomTrie(t, 512)
	var entries entrySlice
	for _, kv := range values {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	// Create a new entry with a slightly modified key
	mid := len(entries) / 2
	key := common.CopyBytes(entries[mid-1].k)
	for n := len(key) - 1; n >= 0; n-- {
		if key[n] < 0xff {
			key[n]++
			break
		}
	}
	noop := &kv{key, []byte{}, false}
	entries = append(append(append([]*kv{}, entries[:mid]...), noop), entries[mid:]...)

	var keys [][]byte
	var vals [][]byte
	for i := 0; i < len(entries); i++ {
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), "storage", nil, nil, keys, vals, nil)
	if err == nil {
		t.Fatalf("Expected failure on noop entry")
	}
}

func increseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			break
		}
	}
	return key
}

func decreseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]--
		if key[i] != 0xff {
			break
		}
	}
	return key
}

func nonRandomTrie(n int) (*Trie, map[string]*kv) {
	trie, err := New(common.Hash{}, NewDatabase((memorydb.New())))
	if err != nil {
		panic(err)
	}
	vals := make(map[string]*kv)
	max := uint64(0xffffffffffffffff)
	for i := uint64(0); i < uint64(n); i++ {
		value := make([]byte, 32)
		key := make([]byte, 32)
		binary.LittleEndian.PutUint64(key, i)
		binary.LittleEndian.PutUint64(value, i-max)
		//value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		elem := &kv{key, value, false}
		trie.Update(elem.k, elem.v)
		vals[string(elem.k)] = elem
	}
	return trie, vals
}

func nonRandomAccountTrie(n int) (*Trie, map[string]*kv) {
	trie, err := New(common.Hash{}, NewDatabase((memorydb.New())))
	if err != nil {
		panic(err)
	}
	vals := make(map[string]*kv)

	for i := uint64(1); i <= uint64(n); i++ {
		account := new(types.StateAccount)
		account.Nonce = i
		account.Balance = big.NewInt(int64(i))
		account.Root = common.Hash{}
		account.KeccakCodeHash = common.FromHex("678910")
		account.PoseidonCodeHash = common.FromHex("1112131415")

		key := make([]byte, 32)
		binary.LittleEndian.PutUint64(key, i)
		trie.UpdateAccount(key, account)
		elem := &kv{key, trie.Get(key), false}
		vals[string(elem.k)] = elem
	}
	return trie, vals
}

func BenchmarkVerifyRangeProof10(b *testing.B)   { benchmarkVerifyRangeProof(b, 10) }
func BenchmarkVerifyRangeProof100(b *testing.B)  { benchmarkVerifyRangeProof(b, 100) }
func BenchmarkVerifyRangeProof1000(b *testing.B) { benchmarkVerifyRangeProof(b, 1000) }
func BenchmarkVerifyRangeProof5000(b *testing.B) { benchmarkVerifyRangeProof(b, 5000) }

func benchmarkVerifyRangeProof(b *testing.B, size int) {
	t := new(testing.T)
	trie, vals := randomTrie(t, 8192)
	var entries entrySlice
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	sort.Sort(entries)

	start := 2
	end := start + size
	proof := memorydb.New()
	if err := trie.Prove(entries[start].k, 0, proof); err != nil {
		b.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[end-1].k, 0, proof); err != nil {
		b.Fatalf("Failed to prove the last node %v", err)
	}
	var keys [][]byte
	var values [][]byte
	for i := start; i < end; i++ {
		keys = append(keys, entries[i].k)
		values = append(values, entries[i].v)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := VerifyRangeProof(trie.Hash(), "storage", keys[0], keys[len(keys)-1], keys, values, proof)
		if err != nil {
			b.Fatalf("Case %d(%d->%d) expect no error, got %v", i, start, end-1, err)
		}
	}
}
