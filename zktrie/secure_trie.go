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
	"fmt"

	itrie "github.com/scroll-tech/zktrie/trie"
	itypes "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

const (
	debug            = false
	storageKeyLength = 32
	accountKeyLength = 20
)

func checkKeybyteSize(b []byte, sizes ...int) bool {
	if !debug {
		return true
	}
	for _, size := range sizes {
		if len(b) == size {
			return true
		}
	}
	panic(fmt.Sprintf("invalid keybyte size, got %v, want %v", len(b), sizes))
}

var magicHash []byte = []byte("THIS IS THE MAGIC INDEX FOR ZKTRIE")

// SecureTrie is a wrapper of Trie which make the key secure
type SecureTrie struct {
	zktrie *itrie.ZkTrie
	db     *Database

	// trie is constructed for inner trie method invoke
	trie *Trie
}

func NewSecure(root common.Hash, db *Database) (*SecureTrie, error) {
	if db == nil {
		panic("zktrie.NewSecure called without a database")
	}

	// for proof generation
	impl, err := itrie.NewZkTrieImplWithRoot(db, StoreHashFromNodeHash(root), itrie.NodeKeyValidBytes*8)
	if err != nil {
		return nil, err
	}

	trie := newTrieWithImpl(impl, db)
	return &SecureTrie{zktrie: trie.secureTrie, db: db, trie: trie}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *SecureTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

func (t *SecureTrie) TryGet(key []byte) ([]byte, error) {
	checkKeybyteSize(key, accountKeyLength, storageKeyLength)
	return t.zktrie.TryGet(key)
}

func (t *SecureTrie) TryGetNode(path []byte) ([]byte, int, error) {
	return t.trie.TryGetNode(path)
}

func (t *SecureTrie) UpdateAccount(key []byte, account *types.StateAccount) {
	if err := t.TryUpdateAccount(key, account); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdateAccount will update the account value in trie
func (t *SecureTrie) TryUpdateAccount(key []byte, account *types.StateAccount) error {
	checkKeybyteSize(key, accountKeyLength)
	value, flag := account.MarshalFields()
	return t.zktrie.TryUpdate(key, flag, value)
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *SecureTrie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate will update the storage value in trie. value is restricted to length of bytes32.
func (t *SecureTrie) TryUpdate(key, value []byte) error {
	checkKeybyteSize(key, storageKeyLength)
	return t.zktrie.TryUpdate(key, 1, []itypes.Byte32{*itypes.NewByte32FromBytes(value)})
}

// Delete removes any existing value for key from the trie.
func (t *SecureTrie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

func (t *SecureTrie) TryDelete(key []byte) error {
	checkKeybyteSize(key, accountKeyLength, storageKeyLength)
	return t.zktrie.TryDelete(key)
}

// GetKey returns the preimage of a hashed key that was
// previously used to store a value.
func (t *SecureTrie) GetKey(key []byte) []byte {
	// TODO: use a kv cache in memory
	if err := CheckKeyLength(key, 32); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
		return nil
	}
	hash, err := keybytesToHashKeyAndCheck(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
		return nil
	}
	k, err := itypes.NewBigIntFromHashBytes(hash.Bytes())
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
		return nil
	}
	if t.db.preimages != nil {
		return t.db.preimages.preimage(common.BytesToHash(k.Bytes()))
	}
	return nil
}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (t *SecureTrie) Commit(onleaf LeafCallback) (common.Hash, int, error) {
	// in current implmentation, every update of trie already writes into database
	// so Commmit does nothing
	// TODO: apply the corresponding onleaf callback!
	return t.Hash(), 0, nil
}

// Hash returns the root hash of SecureBinaryTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *SecureTrie) Hash() common.Hash {
	var hash common.Hash
	hash.SetBytes(t.zktrie.Hash())
	return hash
}

// Copy returns a copy of SecureBinaryTrie.
func (t *SecureTrie) Copy() *SecureTrie {
	root := t.trie.Hash()
	secure, err := NewSecure(root, t.db)
	if err != nil {
		log.Crit("secure trie copy failed", "root", root, "err", err)
	}
	return secure
}

// NodeIterator returns an iterator that returns nodes of the underlying trie. Iteration
// starts at the key after the given start key.
func (t *SecureTrie) NodeIterator(start []byte) NodeIterator {
	return newNodeIterator(t.trie, start)
}

func (t *SecureTrie) String() string {
	return t.trie.String()
}
