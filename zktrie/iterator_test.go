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
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

func TestIterator(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do000000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxverb"},
		{"ether000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxwookiedoo"},
		{"horse000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxstallion"},
		{"shaman00000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxhorse"},
		{"doge0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxcoin"},
		{"dog00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxpuppy"},
		{"somethingveryoddindeedthis is000", "xxxxxxxxxxxxxxxxxmyothernodedata"},
	}
	all := make(map[string]string)
	for _, val := range vals {
		all[val.k] = val.v
		trie.Update([]byte(val.k), []byte(val.v))
	}
	trie.Commit(nil)

	found := make(map[string]string)
	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	for k, v := range all {
		if found[k] != v {
			t.Errorf("iterator value mismatch for %s: got %q want %q", k, found[k], v)
		}
	}
}

type kv struct {
	k, v []byte
	t    bool
}

func TestIteratorLargeData(t *testing.T) {
	trie := newEmpty()
	vals := make(map[string]*kv)

	for i := byte(1); i < 255; i++ {
		value := &kv{common.RightPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.RightPadBytes([]byte{10, i}, 32), []byte{i}, false}
		trie.Update(value.k, value.v)
		trie.Update(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}

	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		vals[string(it.Key)].t = true
	}

	var untouched []*kv
	for _, value := range vals {
		if !value.t {
			untouched = append(untouched, value)
		}
	}

	if len(untouched) > 0 {
		t.Errorf("Missed %d nodes", len(untouched))
		for _, value := range untouched {
			t.Error(value)
		}
	}
}

func TestIteratorLargeDataSecureTrie(t *testing.T) {
	secureTrie, _ := NewSecure(
		common.Hash{},
		NewDatabaseWithConfig(memorydb.New(), &Config{Preimages: true}))
	vals := make(map[string]*kv)

	for i := byte(1); i < 255; i++ {
		value1 := &kv{common.RightPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.RightPadBytes([]byte{10, i}, 32), []byte{i}, false}
		err := secureTrie.TryUpdate(value1.k, value1.v)
		if err != nil {
			t.Fatal(err)
		}

		err = secureTrie.TryUpdate(value2.k, value2.v)
		if err != nil {
			t.Fatal(err)
		}

		secureKey1 := toSecureKey(value1.k)
		secureKey2 := toSecureKey(value2.k)
		vals[string(secureKey1)] = value1
		vals[string(secureKey2)] = value2
	}

	it := NewIterator(secureTrie.NodeIterator(nil))
	cnt := 0
	for it.Next() {
		cnt += 1
		vals[string(it.Key)].t = true
	}

	var untouched []*kv
	for _, value := range vals {
		if !value.t {
			untouched = append(untouched, value)
		}
	}

	if len(untouched) > 0 {
		t.Errorf("Missed %d nodes", len(untouched))
		for _, value := range untouched {
			t.Error(value)
		}
	}
}

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorCoverage(t *testing.T) {
	// Create some arbitrary test trie to iterate
	db, trie, _ := makeTestTrie(t)

	// Gather all the node hashes found by the iterator
	hashes := make(map[common.Hash]struct{})
	for it := trie.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hashes[it.Hash()] = struct{}{}
		}
	}
	// Cross-check the hashes and the database itself
	for hash := range hashes {
		if _, err := db.Get(StoreHashFromNodeHash(hash)[:]); err != nil {
			t.Errorf("failed to retrieve reported node %x: %v", hash, err)
		}
	}

	// Skip the other side of cross-check, as zkTrie will write DB on each update thus having more DB elements
	//for hash, obj := range db.dirties {
	//	if obj != nil && hash != (common.Hash{}) {
	//		if _, ok := hashes[hash]; !ok {
	//			t.Errorf("state entry not reported %x", hash)
	//		}
	//	}
	//}
	//it := db.diskdb.NewIterator(nil, nil)
	//for it.Next() {
	//	key := it.Key()
	//	if _, ok := hashes[common.BytesToHash(key)]; !ok {
	//		t.Errorf("state entry not reported %x", key)
	//	}
	//}
	//it.Release()
}

// Tests that the node iterator indeed walks over the entire database contents.
func TestNodeIteratorCoverageSecureTrie(t *testing.T) {
	// Create some arbitrary test trie to iterate
	db, tr, _ := makeTestSecureTrie()

	// Gather all the node hashes found by the iterator
	hashes := make(map[common.Hash]struct{})
	for it := tr.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hashes[it.Hash()] = struct{}{}
		}
	}
	// Cross-check the hashes and the database itself
	for hash := range hashes {
		if _, err := db.Get(StoreHashFromNodeHash(hash)[:]); err != nil {
			t.Errorf("failed to retrieve reported node %x: %v", hash, err)
		}
	}

	// Skip the other side of cross-check, as zkTrie will write DB on each update thus having more DB elements
	//for hash, obj := range db.dirties {
	//	if obj != nil && hash != (common.Hash{}) {
	//		if _, ok := hashes[hash]; !ok {
	//			t.Errorf("state entry not reported %x", hash)
	//		}
	//	}
	//}
	//it := db.diskdb.NewIterator(nil, nil)
	//for it.Next() {
	//	key := it.Key()
	//	if _, ok := hashes[common.BytesToHash(key)]; !ok {
	//		t.Errorf("state entry not reported %x", key)
	//	}
	//}
	//it.Release()
}

type kvs struct{ k, v string }

var testdata1 = []kvs{
	{"bar00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxb"},
	{"barb0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxba"},
	{"bard0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbc"},
	{"bars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbb"},
	{"fab00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxz"},
	{"foo00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxa"},
	{"food0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxab"},
	{"foos0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxaa"},
}

var testdata2 = []kvs{
	{"aardvark000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxc"},
	{"bar00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxb"},
	{"barb0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbd"},
	{"bars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbe"},
	{"fab00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxz"},
	{"foo00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxa"},
	{"foos0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxaa"},
	{"food0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxab"},
	{"jars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxd"},
}

func TestIteratorSeek(t *testing.T) {
	trie := newEmpty()
	for _, val := range testdata1 {
		trie.Update([]byte(val.k), []byte(val.v))
	}

	// Seek to the middle.
	it := NewIterator(trie.NodeIterator([]byte("fab00000000000000000000000000000")))
	if err := checkIteratorOrder(testdata1[4:], it); err != nil {
		t.Fatal(err)
	}

	// Seek to a non-existent key.
	it = NewIterator(trie.NodeIterator([]byte("barc0000000000000000000000000000")))
	if err := checkIteratorOrder(testdata1[2:], it); err != nil {
		t.Fatal(err)
	}

	// Seek beyond the end.
	it = NewIterator(trie.NodeIterator([]byte("z")))
	if err := checkIteratorOrder(nil, it); err != nil {
		t.Fatal(err)
	}
}

func checkIteratorOrder(want []kvs, it *Iterator) error {
	for it.Next() {
		if len(want) == 0 {
			return fmt.Errorf("didn't expect any more values, got key %q", it.Key)
		}
		if !bytes.Equal(it.Key, []byte(want[0].k)) {
			return fmt.Errorf("wrong key: got %q, want %q", it.Key, want[0].k)
		}
		want = want[1:]
	}
	if len(want) > 0 {
		return fmt.Errorf("iterator ended early, want key %q", want[0])
	}
	return nil
}

func TestDifferenceIterator(t *testing.T) {
	triea := newEmpty()
	for _, val := range testdata1 {
		triea.Update([]byte(val.k), []byte(val.v))
	}
	triea.Commit(nil)

	trieb := newEmpty()
	for _, val := range testdata2 {
		trieb.Update([]byte(val.k), []byte(val.v))
	}
	trieb.Commit(nil)

	found := make(map[string]string)
	di, _ := NewDifferenceIterator(triea.NodeIterator(nil), trieb.NodeIterator(nil))
	it := NewIterator(di)
	for it.Next() {
		found[string(it.Key)] = string(it.Value)
	}

	all := []struct{ k, v string }{
		{"aardvark000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxc"},
		{"barb0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbd"},
		{"bars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbe"},
		{"jars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxd"},
	}
	for _, item := range all {
		if found[item.k] != item.v {
			t.Errorf("iterator value mismatch for %s: got %v want %v", item.k, found[item.k], item.v)
		}
	}
	if len(found) != len(all) {
		t.Errorf("iterator count mismatch: got %d values, want %d", len(found), len(all))
	}
}

func TestUnionIterator(t *testing.T) {
	triea := newEmpty()
	for _, val := range testdata1 {
		triea.Update([]byte(val.k), []byte(val.v))
	}
	triea.Commit(nil)

	trieb := newEmpty()
	for _, val := range testdata2 {
		trieb.Update([]byte(val.k), []byte(val.v))
	}
	trieb.Commit(nil)

	di, _ := NewUnionIterator([]NodeIterator{triea.NodeIterator(nil), trieb.NodeIterator(nil)})
	it := NewIterator(di)

	all := []struct{ k, v string }{
		{"aardvark000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxc"},
		{"bar00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxb"},
		{"barb0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbd"},
		{"barb0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxba"},
		{"bard0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbc"},
		{"bars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbb"},
		{"bars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxbe"},
		{"fab00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxz"},
		{"foo00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxa"},
		{"food0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxab"},
		{"foos0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxaa"},
		{"jars0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxd"},
	}

	for i, kv := range all {
		if !it.Next() {
			t.Errorf("Iterator ends prematurely at element %d", i)
		}
		if kv.k != string(it.Key) {
			t.Errorf("iterator value mismatch for element %d: got key %s want %s", i, it.Key, kv.k)
		}
		if kv.v != string(it.Value) {
			t.Errorf("iterator value mismatch for element %d: got value %s want %s", i, it.Value, kv.v)
		}
	}
	if it.Next() {
		t.Errorf("Iterator returned extra values.")
	}
}

func TestIteratorNoDups(t *testing.T) {
	tr := newEmpty()
	for _, val := range testdata1 {
		tr.Update([]byte(val.k), []byte(val.v))
	}
	checkIteratorNoDups(t, tr.NodeIterator(nil), nil)
}

func TestIteratorContinueAfterError(t *testing.T) {
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)

	tr, _ := New(common.Hash{}, triedb)
	for _, val := range testdata1 {
		tr.Update([]byte(val.k), []byte(val.v))
	}
	tr.Commit(nil)
	triedb.Commit(tr.Hash(), true, nil)
	wantNodeCount := checkIteratorNoDups(t, tr.NodeIterator(nil), nil)

	var diskKeys [][]byte
	for it := tr.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			diskKeys = append(diskKeys, StoreHashFromNodeHash(it.Hash())[:])
		}
	}

	for i := 0; i < 20; i++ {
		// Create trie that will load all nodes from DB.
		tr, _ := New(tr.Hash(), triedb)

		// Remove a random node from the database. It can't be the root node
		// because that one is already loaded.
		var (
			rkey common.Hash
			rval []byte
		)
		for {
			copy(rkey[:], diskKeys[rand.Intn(len(diskKeys))])
			if !bytes.Equal(rkey[:], StoreHashFromNodeHash(tr.Hash())[:]) {
				break
			}
		}

		rval, _ = diskdb.Get(rkey[:])
		diskdb.Delete(rkey[:])
		// Iterate until the error is hit.
		seen := make(map[string]bool)
		it := tr.NodeIterator(nil)
		checkIteratorNoDups(t, it, seen)
		if !strings.Contains(it.Error().Error(), "not found") {
			t.Errorf("Iterator returned wrong error: %v", it.Error())
		}

		// Add the node back and continue iteration.
		diskdb.Put(rkey[:], rval)
		checkIteratorNoDups(t, it, seen)
		if it.Error() != nil {
			t.Fatal("unexpected error", it.Error())
		}
		if len(seen) != wantNodeCount {
			t.Fatal("wrong node iteration count, got", len(seen), "want", wantNodeCount)
		}
	}
}

// Similar to the test above, this one checks that failure to create nodeIterator at a
// certain key prefix behaves correctly when Next is called. The expectation is that Next
// should retry seeking before returning true for the first time.
func TestIteratorContinueAfterSeekError(t *testing.T) {
	// Commit test trie to db, then remove the node containing "bars".
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)

	ctr, _ := New(common.Hash{}, triedb)
	for _, val := range testdata1 {
		ctr.Update([]byte(val.k), []byte(val.v))
	}
	root, _, _ := ctr.Commit(nil)
	triedb.Commit(root, true, nil)

	// Delete a random node
	barsNodeDiskKey := StoreHashFromNodeHash(common.HexToHash("0076cc317ac42e3fc2dea8bd3869583c74cb7107666c9dc0b57853ea6d80a2bc"))[:]
	barsNodeBlob, _ := diskdb.Get(barsNodeDiskKey)
	diskdb.Delete(barsNodeDiskKey)

	// Create a new iterator that seeks to "bars". Seeking can't proceed because
	// the node is missing.
	tr, _ := New(root, triedb)
	it := tr.NodeIterator([]byte("bars"))
	if !strings.Contains(it.Error().Error(), "not found") {
		t.Errorf("Iterator returned wrong error: %v", it.Error())
	}

	// Reinsert the missing node.
	diskdb.Put(barsNodeDiskKey, barsNodeBlob)

	// Check that iteration produces the right set of values.
	if err := checkIteratorOrder(testdata1[3:], NewIterator(it)); err != nil {
		t.Fatal(err)
	}
}

func checkIteratorNoDups(t *testing.T, it NodeIterator, seen map[string]bool) int {
	if seen == nil {
		seen = make(map[string]bool)
	}
	for it.Next(true) {
		if seen[string(it.Path())] {
			t.Fatalf("iterator visited node path %x twice", it.Path())
		}
		seen[string(it.Path())] = true
	}
	return len(seen)
}

type loggingDb struct {
	getCount uint64
	backend  ethdb.KeyValueStore
}

func (l *loggingDb) Has(key []byte) (bool, error) {
	return l.backend.Has(key)
}

func (l *loggingDb) Get(key []byte) ([]byte, error) {
	l.getCount++
	return l.backend.Get(key)
}

func (l *loggingDb) Put(key []byte, value []byte) error {
	return l.backend.Put(key, value)
}

func (l *loggingDb) Delete(key []byte) error {
	return l.backend.Delete(key)
}

func (l *loggingDb) NewBatch() ethdb.Batch {
	return l.backend.NewBatch()
}

func (l *loggingDb) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	fmt.Printf("NewIterator\n")
	return l.backend.NewIterator(prefix, start)
}
func (l *loggingDb) Stat(property string) (string, error) {
	return l.backend.Stat(property)
}

func (l *loggingDb) Compact(start []byte, limit []byte) error {
	return l.backend.Compact(start, limit)
}

func (l *loggingDb) Close() error {
	return l.backend.Close()
}
