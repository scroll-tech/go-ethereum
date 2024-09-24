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
	"math/big"
)

// ZkTrie wraps a trie with key hashing. In a secure trie, all
// access operations hash the key using keccak256. This prevents
// calling code from creating long chains of nodes that
// increase the access time.
//
// Contrary to a regular trie, a ZkTrie can only be created with
// New and must have an attached database. The database also stores
// the preimage of each key.
//
// ZkTrie is not safe for concurrent use.
type ZkTrie struct {
	tree *ZkTrieImpl
}

// NodeKeyValidBytes is the number of least significant bytes in the node key
// that are considered valid to addressing the leaf node, and thus limits the
// maximum trie depth to NodeKeyValidBytes * 8.
// We need to truncate the node key because the key is the output of Poseidon
// hash and the key space doesn't fully occupy the range of power of two. It can
// lead to an ambiguous bit representation of the key in the finite field
// causing a soundness issue in the zk circuit.
const NodeKeyValidBytes = 31

// NewSecure creates a trie
// SecureBinaryTrie bypasses all the buffer mechanism in *Database, it directly uses the
// underlying diskdb
func NewZkTrie(root Byte32, db ZktrieDatabase) (*ZkTrie, error) {
	maxLevels := NodeKeyValidBytes * 8
	tree, err := NewZkTrieImplWithRoot((db), NewHashFromBytes(root.Bytes()), maxLevels)
	if err != nil {
		return nil, err
	}
	return &ZkTrie{
		tree: tree,
	}, nil
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *ZkTrie) TryGet(key []byte) ([]byte, error) {
	k, err := ToSecureKey(key)
	if err != nil {
		return nil, err
	}

	return t.tree.TryGet(NewHashFromBigInt(k))
}

// Tree exposed underlying ZkTrieImpl
func (t *ZkTrie) Tree() *ZkTrieImpl {
	return t.tree
}

// TryGetNode attempts to retrieve a trie node by compact-encoded path. It is not
// possible to use keybyte-encoding as the path might contain odd nibbles.
func (t *ZkTrie) TryGetNode(path []byte) ([]byte, int, error) {
	panic("unimplemented")
}

func (t *ZkTrie) updatePreimage(preimage []byte, hashField *big.Int) {
	t.tree.db.UpdatePreimage(preimage, hashField)
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
func (t *ZkTrie) TryUpdate(key []byte, vFlag uint32, vPreimage []Byte32) error {
	k, err := ToSecureKey(key)
	if err != nil {
		return err
	}
	t.updatePreimage(key, k)
	return t.tree.TryUpdate(NewHashFromBigInt(k), vFlag, vPreimage)
}

// TryDelete removes any existing value for key from the trie.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *ZkTrie) TryDelete(key []byte) error {
	k, err := ToSecureKey(key)
	if err != nil {
		return err
	}

	kHash := NewHashFromBigInt(k)
	//mitigate the create-delete issue: do not delete unexisted key
	if r, _ := t.tree.TryGet(kHash); r == nil {
		return nil
	}

	return t.tree.TryDelete(kHash)
}

// Hash returns the root hash of SecureBinaryTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (t *ZkTrie) Hash() []byte {
	root, err := t.tree.Root()
	if err != nil {
		panic("root failed in trie.Hash")
	}
	return root.Bytes()
}

// Commit flushes the trie to database
func (t *ZkTrie) Commit() error {
	return t.tree.Commit()
}

// Copy returns a copy of SecureBinaryTrie.
func (t *ZkTrie) Copy() *ZkTrie {
	return &ZkTrie{
		tree: t.tree.Copy(),
	}
}

// Prove is a simlified calling of ProveWithDeletion
func (t *ZkTrie) Prove(key []byte, fromLevel uint, writeNode func(*Node) error) error {
	return t.ProveWithDeletion(key, fromLevel, writeNode, nil)
}

// ProveWithDeletion constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
//
// If the trie contain value for key, the onHit is called BEFORE writeNode being called,
// both the hitted leaf node and its sibling node is provided as arguments so caller
// would receive enough information for launch a deletion and calculate the new root
// base on the proof data
// Also notice the sibling can be nil if the trie has only one leaf
func (t *ZkTrie) ProveWithDeletion(key []byte, fromLevel uint, writeNode func(*Node) error, onHit func(*Node, *Node)) error {
	k, err := NewHashFromCheckedBytes(key)
	if err != nil {
		return err
	}
	var prev *Node
	return t.tree.Prove(k, fromLevel, func(n *Node) (err error) {
		defer func() {
			if err == nil {
				err = writeNode(n)
			}
			prev = n
		}()

		if prev != nil {
			switch prev.Type {
			case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
			default:
				// sanity check: we should stop after obtain leaf/empty
				panic("unexpected behavior in prove")
			}
		}

		if onHit == nil {
			return
		}

		// check and call onhit
		if n.Type == NodeTypeLeaf_New && bytes.Equal(n.NodeKey.Bytes(), k.Bytes()) {
			if prev == nil {
				// for sole element trie
				onHit(n, nil)
			} else {
				var sibling, nHash *Hash
				nHash, err = n.NodeHash()
				if err != nil {
					return
				}

				if bytes.Equal(nHash.Bytes(), prev.ChildL.Bytes()) {
					sibling = prev.ChildR
				} else {
					sibling = prev.ChildL
				}

				if siblingNode, err := t.tree.getNode(sibling); err == nil {
					onHit(n, siblingNode)
				} else {
					onHit(n, nil)
				}
			}

		}
		return
	})
}
