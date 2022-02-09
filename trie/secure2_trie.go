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
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie/db"
	"github.com/iden3/go-iden3-crypto/poseidon"
	"math/big"
)

// Secure2Trie wraps a trie with key hashing. In a secure trie, all
// access operations hash the key using keccak256. This prevents
// calling code from creating long chains of nodes that
// increase the access time.
//
// Contrary to a regular trie, a SecureTrie can only be created with
// New and must have an attached database. The database also stores
// the preimage of each key.
//
// SecureTrie is not safe for concurrent use.
type Secure2Trie struct {
	tree *MerkleTree
}

// NewSecure2 creates a trie
func NewSecure2(db db.Storage, maxLevels int) (*Secure2Trie, error) {
	if db == nil {
		panic("trie.NewSecure called without a database")
	}
	tree, err := NewMerkleTree(db, maxLevels)
	if err != nil {
		return nil, err
	}
	return &Secure2Trie{
		tree: tree,
	}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *Secure2Trie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Secure2Trie) TryGet(key []byte) ([]byte, error) {
	var word Byte32
	copy(word[:], key[:32])
	node, err := t.tree.GetLeafNodeByWord(&word)
	if err != nil {
		return nil, err
	}
	return node.ValuePreimage[:], nil
}

// TryUpdateAccount will abstract the write of an account to the
// secure trie.
func (t *Secure2Trie) TryUpdateAccount(key []byte, acc *types.StateAccount) error {
	keyPreimage := new(Byte32)
	copy(keyPreimage[:], key[:])

	vHash, err := acc.Hash()
	if err != nil {
		return err
	}
	value := acc.MarshalBytes()

	err = t.tree.AddVarWord(keyPreimage, vHash, value)
	if err != nil {
		return err
	}
	return nil
}

// Update associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
func (t *Secure2Trie) Update(key, value []byte) {
	if err := t.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryUpdate associates key with value in the trie. Subsequent calls to
// Get will return value. If value has length zero, any existing value
// is deleted from the trie and calls to Get will return nil.
//
// The value bytes must not be modified by the caller while they are
// stored in the trie.
//
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Secure2Trie) TryUpdate(key, value []byte) error {
	var kPreimage, vPreimage Byte32
	copy(kPreimage[:], key[:32])
	copy(vPreimage[:], value[:32])
	_, err := t.tree.UpdateWord(&kPreimage, &vPreimage)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes any existing value for key from the trie.
func (t *Secure2Trie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Secure2Trie) TryDelete(key []byte) error {
	var kPreimage Byte32
	copy(kPreimage[:], key[:32])
	return t.tree.DeleteWord(&kPreimage)
}

// GetKey returns the preimage of a hashed key that was
// previously used to store a value.
func (t *Secure2Trie) GetKey(kHashBytes []byte) []byte {
	kHash, err := NewHashFromBytes(kHashBytes)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	node, err := t.tree.GetNode(kHash)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return node.KeyPreimage[:]
}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (t *Secure2Trie) Commit(LeafCallback) (common.Hash, int, error) {
	// FIXME
	return t.Hash(), 0, nil
}

// Hash returns the root hash of SecureTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *Secure2Trie) Hash() common.Hash {
	var hash common.Hash
	hash.SetBytes(t.tree.rootKey.Bytes())
	return hash
}

// Copy returns a copy of SecureTrie.
func (t *Secure2Trie) Copy() *Secure2Trie {
	cpy := *t
	return &cpy
}

// NodeIterator returns an iterator that returns nodes of the underlying trie. Iteration
// starts at the key after the given start key.
func (t *Secure2Trie) NodeIterator(start []byte) NodeIterator {
	/// FIXME
	panic("not implemented")
}

// hashKey returns the hash of key as an ephemeral buffer.
// The caller must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
func (t *Secure2Trie) hashKey(key []byte) []byte {
	if len(key) != 32 {
		panic("non byte32 input to hashKey")
	}
	low16 := new(big.Int).SetBytes(key[:16])
	high16 := new(big.Int).SetBytes(key[16:])
	hash, err := poseidon.Hash([]*big.Int{low16, high16})
	if err != nil {
		panic(err)
	}
	return hash.Bytes()
}
