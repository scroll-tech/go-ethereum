// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"fmt"

	itrie "github.com/scroll-tech/zktrie/trie"
	itypes "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rlp"
)

var ErrCommitDisabled = errors.New("no database for committing")

// TODO: using it for optimization
//var stPool = sync.Pool{
//	New: func() interface{} {
//		return NewStackTrie(nil)
//	},
//}
//
//func stackTrieFromPool(depth int, db ethdb.KeyValueWriter) *StackTrie {
//	st := stPool.Get().(*StackTrie)
//	st.depth = depth
//	st.db = db
//	return st
//}
//
//func returnToPool(st *StackTrie) {
//	st.Reset()
//	stPool.Put(st)
//}

const (
	emptyNode = iota
	parentNode
	leafNode
	hashedNode
)

// StackTrie is a trie implementation that expects keys to be inserted
// in order. Once it determines that a subtree will no longer be inserted
// into, it will hash it and free up the memory it uses.
type StackTrie struct {
	nodeType uint8                // node type (as in parentNode, leafNode, emptyNode and hashedNode)
	depth    int                  // depth to the root
	db       ethdb.KeyValueWriter // Pointer to the commit db, can be nil

	// properties for leaf node
	val       []itypes.Byte32
	flag      uint32
	binaryKey []byte

	// properties for parent node
	children [2]*StackTrie

	// properties for hashed node
	nodeHash *itypes.Hash
}

// NewStackTrie allocates and initializes an empty trie.
func NewStackTrie(db ethdb.KeyValueWriter) *StackTrie {
	return &StackTrie{
		nodeType: emptyNode,
		db:       db,
	}
}

func (st *StackTrie) TryUpdateWithKind(kind string, key, value []byte) error {
	if kind == "account" {
		var account types.StateAccount
		if err := rlp.DecodeBytes(value, &account); err != nil {
			return InvalidStateAccountRLPEncodingError
		}
		return st.TryUpdateAccount(key, &account)
	} else if kind == "storage" {
		return st.TryUpdate(key, value)
	} else {
		return InvalidUpdateKindError
	}
}

func (st *StackTrie) TryUpdate(key, value []byte) error {
	if err := CheckKeyLength(key, 32); err != nil {
		return err
	}
	if _, err := keybytesToHashKeyAndCheck(key); err != nil {
		return err
	}

	binary := keybytesToBinary(key)
	if len(value) == 0 {
		panic("deletion not supported")
	}
	st.insert(binary, 1, []itypes.Byte32{*itypes.NewByte32FromBytes(value)})
	return nil
}

func (st *StackTrie) Update(key, value []byte) {
	if err := st.TryUpdate(key, value); err != nil {
		log.Error(fmt.Sprintf("Unhandled trie error: %v", err))
	}
}

func (st *StackTrie) TryUpdateAccount(key []byte, account *types.StateAccount) error {
	if err := CheckKeyLength(key, 32); err != nil {
		return err
	}
	//TODO: cache the hash!
	if _, err := keybytesToHashKeyAndCheck(key); err != nil {
		return err
	}

	binary := keybytesToBinary(key)
	value, flag := account.MarshalFields()
	st.insert(binary, flag, value)
	return nil
}

func (st *StackTrie) UpdateAccount(key []byte, account *types.StateAccount) {
	if err := st.TryUpdateAccount(key, account); err != nil {
		log.Error(fmt.Sprintf("Unhandled tri error: %v", err))
	}
}

func (st *StackTrie) Reset() {
	st.db = nil
	st.binaryKey = nil
	st.val = nil
	st.depth = 0
	st.nodeHash = nil
	for i := range st.children {
		st.children[i] = nil
	}
	st.nodeType = emptyNode
}

func newLeafNode(depth int, binaryKey []byte, flag uint32, value []itypes.Byte32, db ethdb.KeyValueWriter) *StackTrie {
	return &StackTrie{
		nodeType:  leafNode,
		depth:     depth,
		binaryKey: binaryKey,
		flag:      flag,
		val:       value,
		db:        db,
	}
}

func newEmptyNode(depth int, db ethdb.KeyValueWriter) *StackTrie {
	return &StackTrie{
		nodeType: emptyNode,
		depth:    depth,
		db:       db,
	}
}

func (st *StackTrie) insert(binary []byte, flag uint32, value []itypes.Byte32) {
	switch st.nodeType {
	case parentNode:
		idx := binary[st.depth]
		if idx == 1 {
			st.children[0].hash()
		}
		st.children[idx].insert(binary, flag, value)
	case leafNode:
		if st.depth == len(st.binaryKey) {
			panic("Trying to insert into existing key")
		}

		origLeaf := newLeafNode(st.depth+1, st.binaryKey, flag, st.val, st.db)
		origIdx := st.binaryKey[st.depth]

		st.nodeType = parentNode
		st.binaryKey = nil
		st.val = nil
		st.children[origIdx] = origLeaf
		st.children[origIdx^1] = newEmptyNode(st.depth+1, st.db)

		newIdx := binary[st.depth]
		if origIdx == newIdx {
			st.children[newIdx].insert(binary, flag, value)
		} else {
			// new fork
			if origIdx > newIdx {
				panic("Trying to insert key in reverse order")
			}
			st.children[origIdx].hash()
			st.children[newIdx] = newLeafNode(st.depth+1, binary, flag, value, st.db)
		}
	case emptyNode:
		st.nodeType = leafNode
		st.flag = flag
		st.binaryKey = binary
		st.val = value
	case hashedNode:
		panic("trying to insert into hashed node")
	default:
		panic("invalid node type")
	}
}

func (st *StackTrie) hash() {
	if st.nodeType == hashedNode {
		return
	}

	var (
		n   *itrie.Node
		err error
	)

	switch st.nodeType {
	case parentNode:
		st.children[0].hash()
		st.children[1].hash()
		n = itrie.NewParentNode(st.children[0].nodeHash, st.children[1].nodeHash)
		// recycle children mem
		st.children[0] = nil
		st.children[1] = nil
	case leafNode:
		//TODO: convert binary to hash key directly
		n = itrie.NewLeafNode(keybytesToHashKey(binaryToKeybytes(st.binaryKey)), st.flag, st.val)
	case emptyNode:
		n = itrie.NewEmptyNode()
	default:
		panic("invalid node type")
	}
	st.nodeType = hashedNode
	st.nodeHash, err = n.NodeHash()
	if err != nil {
		log.Error(fmt.Sprintf("Unhandled stack trie error: %v", err))
		return
	}

	if st.db != nil {
		// TODO! Is it safe to Put the slice here?
		// Do all db implementations copy the value provided?
		if err := st.db.Put(st.nodeHash[:], n.CanonicalValue()); err != nil {
			log.Error(fmt.Sprintf("Unhandled stacktrie db put error: %v", err))
		}
	}
}

// Hash returns the hash of the current node
func (st *StackTrie) Hash() common.Hash {
	st.hash()
	return common.BytesToHash(st.nodeHash.Bytes())
}

// Commit will firstly hash the entrie trie if it's still not hashed
// and then commit all nodes to the associated database. Actually most
// of the trie nodes MAY have been committed already. The main purpose
// here is to commit the root node.
//
// The associated database is expected, otherwise the whole commit
// functionality should be disabled.
func (st *StackTrie) Commit() (common.Hash, error) {
	if st.db == nil {
		return common.Hash{}, ErrCommitDisabled
	}
	st.hash()
	return common.BytesToHash(st.nodeHash.Bytes()), nil
}

func (st *StackTrie) String() string {
	switch st.nodeType {
	case parentNode:
		return fmt.Sprintf("Parent(%s, %s)", st.children[0], st.children[1])
	case leafNode:
		return fmt.Sprintf("Leaf(%q)", binaryToKeybytes(st.binaryKey))
	case hashedNode:
		return fmt.Sprintf("Hashed(%s)", st.nodeHash.Hex())
	case emptyNode:
		return "Empty"
	default:
		panic("unknown node type")
	}
}
