// +build !oldTree
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

	"github.com/scroll-tech/go-ethereum/ethdb"

	"math/big"

	"github.com/iden3/go-iden3-crypto/poseidon"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/types/smt"
	"github.com/scroll-tech/go-ethereum/log"
)

// SecureBinaryTrie wraps a trie with key hashing. In a secure trie, all
// access operations hash the key using keccak256. This prevents
// calling code from creating long chains of nodes that
// increase the access time.
//
// Contrary to a regular trie, a SecureBinaryTrie can only be created with
// New and must have an attached database. The database also stores
// the preimage of each key.
//
// SecureBinaryTrie is not safe for concurrent use.
type SecureBinaryTrie struct {
	tree *MerkleTree
}

// NewSecure creates a trie
// SecureBinaryTrie bypasses all the buffer mechanism in *Database, it directly uses the
// underlying diskdb
func NewSecure(root common.Hash, ethdb *Database) (*SecureBinaryTrie, error) {
	rootHash, err := smt.NewHashFromBytes(root.Bytes())
	if err != nil {
		return nil, err
	}
	tree, err := NewMerkleTreeWithRoot(NewEthKVStorage(ethdb), rootHash, 256)
	if err != nil {
		return nil, err
	}
	return &SecureBinaryTrie{
		tree: tree,
	}, nil
}

// Get returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
func (t *SecureBinaryTrie) Get(key []byte) []byte {
	res, err := t.TryGet(key)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	return res
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureBinaryTrie) TryGet(key []byte) ([]byte, error) {
	word := smt.NewByte32FromBytesPadding(key)
	node, err := t.tree.GetLeafNodeByWord(word)
	if err == ErrKeyNotFound {
		// according to https://github.com/ethereum/go-ethereum/blob/37f9d25ba027356457953eab5f181c98b46e9988/trie/trie.go#L135
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if node.ValuePreimageLen == 32 {
		return smt.UnPadBytes32(node.ValuePreimage), nil
	}
	return node.ValuePreimage[:], nil
}

// TryGetNode attempts to retrieve a trie node by compact-encoded path. It is not
// possible to use keybyte-encoding as the path might contain odd nibbles.
func (t *SecureBinaryTrie) TryGetNode(path []byte) ([]byte, int, error) {
	panic("unimplemented")
}

// TryUpdateAccount will abstract the write of an account to the
// secure trie.
func (t *SecureBinaryTrie) TryUpdateAccount(key []byte, acc *types.StateAccount) error {
	keyPreimage := smt.NewByte32FromBytesPadding(key)

	vHash, err := acc.Hash()
	if err != nil {
		return err
	}
	value := acc.MarshalBytes()

	_, err = t.tree.UpdateVarWord(keyPreimage, vHash, value)
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
func (t *SecureBinaryTrie) Update(key, value []byte) {
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
//
// NOTE: value is restricted to length of bytes32.
func (t *SecureBinaryTrie) TryUpdate(key, value []byte) error {
	kPreimage := smt.NewByte32FromBytesPadding(key)
	vPreimage := smt.NewByte32FromBytesPadding(value)
	_, err := t.tree.UpdateWord(kPreimage, vPreimage)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes any existing value for key from the trie.
func (t *SecureBinaryTrie) Delete(key []byte) {
	if err := t.TryDelete(key); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *SecureBinaryTrie) TryDelete(key []byte) error {
	return t.TryUpdate(key, []byte{})
	//kPreimage := smt.NewByte32FromBytesPadding(key)
	//return t.tree.DeleteWord(kPreimage)
}

// GetKey returns the preimage of a hashed key that was
// previously used to store a value.
func (t *SecureBinaryTrie) GetKey(kHashBytes []byte) []byte {
	// TODO: use a kv cache in memory
	kHash, err := smt.NewBigIntFromHashBytes(kHashBytes)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	node, err := t.tree.GetLeafNode(kHash)
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
	if node == nil {
		return nil
	}
	return smt.UnPadBytes32(node.KeyPreimage[:])
}

// Commit writes all nodes and the secure hash pre-images to the trie's database.
// Nodes are stored with their sha3 hash as the key.
//
// Committing flushes nodes from memory. Subsequent Get calls will load nodes
// from the database.
func (t *SecureBinaryTrie) Commit(LeafCallback) (common.Hash, int, error) {
	// FIXME
	return t.Hash(), 0, nil
}

// Hash returns the root hash of SecureBinaryTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *SecureBinaryTrie) Hash() common.Hash {
	var hash common.Hash
	hash.SetBytes(t.tree.rootKey.Bytes())
	return hash
}

// Copy returns a copy of SecureBinaryTrie.
func (t *SecureBinaryTrie) Copy() *SecureBinaryTrie {
	cpy, err := NewMerkleTreeWithRoot(t.tree.db, t.tree.rootKey, t.tree.maxLevels)
	if err != nil {
		panic("clone trie failed")
	}
	return &SecureBinaryTrie{
		tree: cpy,
	}
}

// NodeIterator returns an iterator that returns nodes of the underlying trie. Iteration
// starts at the key after the given start key.
func (t *SecureBinaryTrie) NodeIterator(start []byte) NodeIterator {
	/// FIXME
	panic("not implemented")
}

// hashKey returns the hash of key as an ephemeral buffer.
// The caller must not hold onto the return value because it will become
// invalid on the next call to hashKey or secKey.
func (t *SecureBinaryTrie) hashKey(key []byte) []byte {
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

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *SecureBinaryTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	word := smt.NewByte32FromBytesPadding(key)
	k, err := word.Hash()
	if err != nil {
		return err
	}
	return t.tree.Prove(k, fromLevel, proofDb)
}
