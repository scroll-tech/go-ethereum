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
	"io/ioutil"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"testing"

	itypes "github.com/scroll-tech/zktrie/types"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb/leveldb"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

// convert key representation from Trie to SecureTrie
func toSecureKey(b []byte) []byte {
	if k, err := itypes.ToSecureKey(b); err != nil {
		return nil
	} else {
		return hashKeyToKeybytes(itypes.NewHashFromBigInt(k))
	}
}

func newEmptySecureTrie() *SecureTrie {
	trie, _ := NewSecure(
		common.Hash{},
		NewDatabaseWithConfig(memorydb.New(), &Config{Preimages: true}),
	)
	return trie
}

// makeTestSecureTrie creates a large enough secure trie for testing.
func makeTestSecureTrie() (*Database, *SecureTrie, map[string][]byte) {
	// Create an empty trie
	triedb := NewDatabase(memorydb.New())
	trie, _ := NewSecure(common.Hash{}, triedb)

	// Fill it with some arbitrary data
	content := make(map[string][]byte)
	for i := byte(0); i < 255; i++ {
		// Map the same data under multiple keys
		key, val := common.LeftPadBytes([]byte{1, i}, 32), bytes.Repeat([]byte{i}, 32)
		content[string(key)] = val
		trie.Update(key, val)

		key, val = common.LeftPadBytes([]byte{2, i}, 32), bytes.Repeat([]byte{i}, 32)
		content[string(key)] = val
		trie.Update(key, val)

		// Add some other data to inflate the trie
		for j := byte(3); j < 13; j++ {
			key, val = common.LeftPadBytes([]byte{j, i}, 32), bytes.Repeat([]byte{j, i}, 16)
			content[string(key)] = val
			trie.Update(key, val)
		}
	}
	trie.Commit(nil)

	// Return the generated trie
	return triedb, trie, content
}

func TestTrieDelete(t *testing.T) {
	t.Skip("var-len kv not supported")
	trie := newEmptySecureTrie()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			trie.Update([]byte(val.k), []byte(val.v))
		} else {
			trie.Delete([]byte(val.k))
		}
	}
	hash := trie.Hash()
	exp := common.HexToHash("29b235a58c3c25ab83010c327d5932bcf05324b7d6b1185e650798034783ca9d")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestTrieGetKey(t *testing.T) {
	trie := newEmptySecureTrie()
	for i := byte(1); i < 255; i++ {
		key := common.RightPadBytes([]byte{i}, 32)
		value := common.LeftPadBytes([]byte{i}, 32)
		err := trie.TryUpdate(key, value)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(trie.Get(key), value) {
			t.Errorf("Get did not return bar")
		}
		if k := trie.GetKey(toSecureKey(key)); !bytes.Equal(k, key) {
			t.Errorf("GetKey returned %q, want %q", k, key)
		}
	}
}

func TestZkTrieConcurrency(t *testing.T) {
	// Create an initial trie and copy if for concurrent access
	_, trie, _ := makeTestSecureTrie()

	threads := runtime.NumCPU()
	tries := make([]*SecureTrie, threads)
	for i := 0; i < threads; i++ {
		cpy := *trie
		tries[i] = &cpy
	}
	// Start a batch of goroutines interactng with the trie
	pend := new(sync.WaitGroup)
	pend.Add(threads)
	for i := 0; i < threads; i++ {
		go func(index int) {
			defer pend.Done()

			for j := byte(0); j < 255; j++ {
				// Map the same data under multiple keys
				key, val := common.LeftPadBytes([]byte{byte(index), 1, j}, 32), bytes.Repeat([]byte{j}, 32)
				tries[index].Update(key, val)

				key, val = common.LeftPadBytes([]byte{byte(index), 2, j}, 32), bytes.Repeat([]byte{j}, 32)
				tries[index].Update(key, val)

				// Add some other data to inflate the trie
				for k := byte(3); k < 13; k++ {
					key, val = common.LeftPadBytes([]byte{byte(index), k, j}, 32), bytes.Repeat([]byte{k, j}, 16)
					tries[index].Update(key, val)
				}
			}
			tries[index].Commit(nil)
		}(i)
	}
	// Wait for all threads to finish
	pend.Wait()
}

func tempDBZK(b *testing.B) (string, *Database) {
	dir, err := ioutil.TempDir("", "zktrie-bench")
	assert.NoError(b, err)

	diskdb, err := leveldb.New(dir, 256, 0, "", false)
	assert.NoError(b, err)
	config := &Config{Cache: 256, Preimages: true}
	return dir, NewDatabaseWithConfig(diskdb, config)
}

const benchElemCountZk = 10000

func BenchmarkTrieGet(b *testing.B) {
	_, tmpdb := tempDBZK(b)
	trie, _ := NewSecure(common.Hash{}, tmpdb)
	defer func() {
		ldb := trie.db.diskdb.(*leveldb.Database)
		ldb.Close()
		os.RemoveAll(ldb.Path())
	}()

	var keys [][]byte
	for i := 0; i < benchElemCountZk; i++ {
		key := make([]byte, 32)
		binary.LittleEndian.PutUint64(key, uint64(i))

		err := trie.TryUpdate(key, key)
		keys = append(keys, key)
		assert.NoError(b, err)
	}

	fmt.Printf("Secure trie hash %v\n", trie.Hash())
	trie.db.Commit(common.Hash{}, true, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := trie.TryGet(keys[rand.Intn(len(keys))])
		assert.NoError(b, err)
	}
	b.StopTimer()
}

func BenchmarkTrieUpdateExisting(b *testing.B) {
	_, tmpdb := tempDBZK(b)
	trie, _ := NewSecure(common.Hash{}, tmpdb)
	defer func() {
		ldb := trie.db.diskdb.(*leveldb.Database)
		ldb.Close()
		os.RemoveAll(ldb.Path())
	}()

	b.ReportAllocs()

	var keys [][]byte
	for i := 0; i < benchElemCountZk; i++ {
		key := make([]byte, 32)
		binary.LittleEndian.PutUint64(key, uint64(i))

		err := trie.TryUpdate(key, key)
		keys = append(keys, key)
		assert.NoError(b, err)
	}

	trie.db.Commit(common.Hash{}, true, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := trie.TryUpdate(keys[rand.Intn(len(keys))], keys[rand.Intn(len(keys))])
		assert.NoError(b, err)
	}
	b.StopTimer()
}

func TestZkTrieDelete(t *testing.T) {
	key := make([]byte, 32)
	value := make([]byte, 32)
	emptyTrie := newEmptySecureTrie()

	var count int = 6
	var hashes []common.Hash
	hashes = append(hashes, emptyTrie.Hash())
	for i := 0; i < count; i++ {
		binary.LittleEndian.PutUint64(key, uint64(i))
		binary.LittleEndian.PutUint64(value, uint64(i))
		err := emptyTrie.TryUpdate(key, value)
		assert.NoError(t, err)
		hashes = append(hashes, emptyTrie.Hash())
	}

	// binary.LittleEndian.PutUint64(key, uint64(0xffffff))
	// err := emptyTrie.TryDelete(key)
	// assert.Equal(t, err, zktrie.ErrKeyNotFound)

	emptyTrie.Commit(nil)

	for i := count - 1; i >= 0; i-- {
		binary.LittleEndian.PutUint64(key, uint64(i))
		v, err := emptyTrie.TryGet(key)
		assert.NoError(t, err)
		assert.NotEmpty(t, v)
		err = emptyTrie.TryDelete(key)
		assert.NoError(t, err)
		hash := emptyTrie.Hash()
		assert.Equal(t, hashes[i].Hex(), hash.Hex())
	}
}
