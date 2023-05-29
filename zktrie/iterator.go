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
	"container/heap"
	"errors"

	itrie "github.com/scroll-tech/zktrie/trie"
	itypes "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rlp"
)

// Iterator is a key-value trie iterator that traverses a Trie.
type Iterator struct {
	nodeIt NodeIterator

	Key   []byte // Current data key on which the iterator is positioned on
	Value []byte // Current data value on which the iterator is positioned on
	Err   error
}

// NewIterator creates a new key-value iterator from a node iterator.
// Note that the value returned by the iterator is raw. If the content is encoded
// (e.g. storage value is RLP-encoded), it's caller's duty to decode it.
func NewIterator(it NodeIterator) *Iterator {
	return &Iterator{
		nodeIt: it,
	}
}

// Next moves the iterator forward one key-value entry.
func (it *Iterator) Next() bool {
	for it.nodeIt.Next(true) {
		if it.nodeIt.Leaf() {
			it.Key = it.nodeIt.LeafKey()
			it.Value = it.nodeIt.LeafBlob()
			return true
		}
	}
	it.Key = nil
	it.Value = nil
	it.Err = it.nodeIt.Error()
	return false
}

// Prove generates the Merkle proof for the leaf node the iterator is currently
// positioned on.
func (it *Iterator) Prove() [][]byte {
	return it.nodeIt.LeafProof()
}

func (it *Iterator) AccountRLP() ([]byte, error) {
	account, err := types.UnmarshalStateAccount(it.Value)
	if err != nil {
		return nil, err
	}
	return rlp.EncodeToBytes(account)
}

// NodeIterator is an iterator to traverse the trie pre-order.
type NodeIterator interface {
	// Next moves the iterator to the next node. If the parameter is false, any child
	// nodes will be skipped.
	Next(bool) bool

	// Error returns the error status of the iterator.
	Error() error

	// Hash returns the hash of the current node.
	Hash() common.Hash

	// Parent returns the hash of the parent of the current node. The hash may be the one
	// grandparent if the immediate parent is an internal node with no hash.
	Parent() common.Hash

	// Path returns the hex-encoded path to the current node.
	// Callers must not retain references to the return value after calling Next.
	// For leaf nodes, the last element of the path is the 'terminator symbol' 0x10.
	Path() []byte

	// Leaf returns true iff the current node is a leaf node.
	Leaf() bool

	// LeafKey returns the key of the leaf. The method panics if the iterator is not
	// positioned at a leaf. Callers must not retain references to the value after
	// calling Next.
	LeafKey() []byte

	// LeafBlob returns the content of the leaf. The method panics if the iterator
	// is not positioned at a leaf. Callers must not retain references to the value
	// after calling Next.
	LeafBlob() []byte

	// LeafProof returns the Merkle proof of the leaf. The method panics if the
	// iterator is not positioned at a leaf. Callers must not retain references
	// to the value after calling Next.
	LeafProof() [][]byte

	// AddResolver sets an intermediate database to use for looking up trie nodes
	// before reaching into the real persistent layer.
	//
	// This is not required for normal operation, rather is an optimization for
	// cases where trie nodes can be recovered from some external mechanism without
	// reading from disk. In those cases, this resolver allows short circuiting
	// accesses and returning them from memory.
	//
	// Before adding a similar mechanism to any other place in Geth, consider
	// making trie.Database an interface and wrapping at that level. It's a huge
	// refactor, but it could be worth it if another occurrence arises.
	AddResolver(ethdb.KeyValueStore)
}

// nodeIteratorState represents the iteration state at one particular node of the
// trie, which can be resumed at a later invocation.
type nodeIteratorState struct {
	hash    common.Hash // Hash of the node being iterated (nil if not standalone)
	node    *itrie.Node // Trie node being iterated
	index   uint8       // Child to be processed next
	pathlen int         // Length of the path to this node
}

type nodeIterator struct {
	trie       *Trie                // Trie being iterated
	stack      []*nodeIteratorState // Hierarchy of trie nodes persisting the iteration state
	binaryPath []byte               // binary path to the current node
	err        error                // Failure set in case of an internal error in the iterator

	resolver ethdb.KeyValueStore // Optional intermediate resolver above the disk layer
}

// errIteratorEnd is stored in nodeIterator.err when iteration is done.
var errIteratorEnd = errors.New("end of iteration")

// seekError is stored in nodeIterator.err if the initial seek has failed.
type seekError struct {
	key []byte
	err error
}

func (e seekError) Error() string {
	return "seek error: " + e.err.Error()
}

func newNodeIterator(trie *Trie, start []byte) NodeIterator {
	it := &nodeIterator{trie: trie}
	it.err = it.seek(start)
	tmp := make([]byte, 0)
	tmp = append(tmp, it.binaryPath...)
	for len(tmp)%8 != 0 {
		tmp = append(tmp, 0)
	}
	return it
}

func (it *nodeIterator) AddResolver(resolver ethdb.KeyValueStore) {
	it.resolver = resolver
}

func (it *nodeIterator) Hash() common.Hash {
	if len(it.stack) == 0 {
		return common.Hash{}
	}
	return it.stack[len(it.stack)-1].hash
}

// Parent is the first full ancestor node, and each node in zktrie is
func (it *nodeIterator) Parent() common.Hash {
	if len(it.stack) < 2 {
		return common.Hash{}
	}
	return it.stack[len(it.stack)-2].hash
}

func (it *nodeIterator) Leaf() bool {
	if last := it.currentNode(); last != nil {
		return last.Type == itrie.NodeTypeLeaf
	}
	return false
}

func (it *nodeIterator) LeafKey() []byte {
	if last := it.currentNode(); last != nil {
		if last.Type == itrie.NodeTypeLeaf {
			return hashKeyToKeybytes(last.NodeKey)
		}
	}
	panic("not at leaf")
}

func (it *nodeIterator) LeafBlob() []byte {
	if last := it.currentNode(); last != nil {
		if last.Type == itrie.NodeTypeLeaf {
			return last.Data()
		}
	}
	panic("not at leaf")
}

func (it *nodeIterator) LeafProof() [][]byte {
	if last := it.currentNode(); last != nil {
		if last.Type == itrie.NodeTypeLeaf {
			proofs := make([][]byte, 0, len(it.stack))
			for _, item := range it.stack {
				proofs = append(proofs, item.node.Value())
			}
			return proofs
		}
	}
	panic("not at leaf")
}

func (it *nodeIterator) Path() []byte {
	return it.binaryPath
}

func (it *nodeIterator) Error() error {
	if it.err == errIteratorEnd {
		return nil
	}
	if seek, ok := it.err.(seekError); ok {
		return seek.err
	}
	return it.err
}

// Next moves the iterator to the next node, returning whether there are any
// further nodes. In case of an internal error this method returns false and
// sets the Error field to the encountered failure. If `descend` is false,
// skips iterating over any subnodes of the current node.
func (it *nodeIterator) Next(descend bool) bool {
	if it.err == errIteratorEnd {
		return false
	}
	if seek, ok := it.err.(seekError); ok {
		if it.err = it.seek(seek.key); it.err != nil {
			return false
		}
	}
	// Otherwise step forward with the iterator and report any errors.
	state, path, err := it.peek(descend)
	it.err = err
	if it.err != nil {
		return false
	}
	it.push(state, path)
	return true
}

func (it *nodeIterator) currentNode() *itrie.Node {
	if len(it.stack) > 0 {
		return it.stack[len(it.stack)-1].node
	}
	return nil
}

func (it *nodeIterator) currentKey() []byte {
	if last := it.currentNode(); last != nil {
		if last.Type == itrie.NodeTypeLeaf {
			return keybytesToBinary(hashKeyToKeybytes(last.NodeKey))
		} else {
			return it.binaryPath
		}
	}
	return nil
}

func (it *nodeIterator) seek(key []byte) error {
	//The path we're looking for is the binary encoded key without terminator.
	binaryKey := keybytesToBinary(key)
	// Move forward until we're just before the closest match to key.

	for {
		state, path, err := it.peekSeek(binaryKey)
		if err != nil {
			return seekError{key, err}
		} else if state == nil || bytes.Compare(path, binaryKey) >= 0 {
			return nil
		}
		it.push(state, path)
	}
}

// init initializes the iterator.
func (it *nodeIterator) init() (*nodeIteratorState, []byte, error) {
	root, err := it.trie.root()
	if err != nil {
		return nil, nil, err
	}
	state := &nodeIteratorState{hash: it.trie.Hash(), node: root, index: 0}
	if root.Type == itrie.NodeTypeLeaf {
		return state, hashKeyToBinary(root.NodeKey), nil
	}
	return state, nil, nil
}

// peek creates the next state of the iterator.
func (it *nodeIterator) peek(descend bool) (*nodeIteratorState, []byte, error) {
	// Initialize the iterator if we've just started.
	if len(it.stack) == 0 {
		state, path, err := it.init()
		return state, path, err
	}
	if !descend {
		// If we're skipping children, pop the current node first
		it.pop()
	}

	// Continue iteration to the next child
	for len(it.stack) > 0 {
		parent := it.stack[len(it.stack)-1]
		if parent.node.Type == itrie.NodeTypeParent && parent.index < 2 {
			nodeHash := parent.node.ChildL
			if parent.index == 1 {
				nodeHash = parent.node.ChildR
			}
			node, err := it.resolveHash(nodeHash)
			if err != nil {
				return nil, nil, err
			}

			state := &nodeIteratorState{
				hash:    common.BytesToHash(nodeHash.Bytes()),
				node:    node,
				index:   0,
				pathlen: len(it.binaryPath),
			}

			var binaryPath []byte
			if node.Type == itrie.NodeTypeLeaf {
				binaryPath = hashKeyToBinary(node.NodeKey)
			} else {
				binaryPath = append(it.binaryPath, parent.index)
			}
			return state, binaryPath, nil
		}
		// No more child nodes, move back up.
		it.pop()
	}
	return nil, nil, errIteratorEnd
}

// peekSeek is like peek, but it also tries to skip resolving hashes by skipping
// over the siblings that do not lead towards the desired seek position.
func (it *nodeIterator) peekSeek(seekBinaryKey []byte) (*nodeIteratorState, []byte, error) {
	// Initialize the iterator if we've just started.
	if len(it.stack) == 0 {
		state, path, err := it.init()
		return state, path, err
	}

	// Continue iteration to the next child
	parent := it.stack[len(it.stack)-1]
	if parent.node.Type == itrie.NodeTypeParent {
		if len(seekBinaryKey) <= len(it.binaryPath) {
			panic("walk into path longer than seek binary key")
		}
		var nodeHash *itypes.Hash
		if seekBinaryKey[len(it.binaryPath)] == 0 {
			parent.index = 0
			nodeHash = parent.node.ChildL
		} else {
			parent.index = 1
			nodeHash = parent.node.ChildR
		}

		node, err := it.resolveHash(nodeHash)
		if err != nil {
			return nil, nil, err
		}
		var binaryPath []byte
		if node.Type == itrie.NodeTypeLeaf {
			binaryPath = hashKeyToBinary(node.NodeKey)
		} else {
			binaryPath = append(it.binaryPath, parent.index)
		}

		return &nodeIteratorState{
			hash:    common.BytesToHash(nodeHash.Bytes()),
			node:    node,
			index:   0,
			pathlen: len(it.binaryPath),
		}, binaryPath, nil
	}

	// reach leaf of empty node, seek is done!
	return nil, nil, nil
}

func (it *nodeIterator) resolveHash(hash *itypes.Hash) (*itrie.Node, error) {
	if it.resolver != nil {
		if blob, err := it.resolver.Get(hash[:]); err == nil && len(blob) > 0 {
			if resolved, err := itrie.NewNodeFromBytes(blob); err == nil {
				return resolved, nil
			}
		}
	}
	resolved, err := it.trie.getNodeByHash(hash)
	return resolved, err
}

func (it *nodeIterator) push(state *nodeIteratorState, path []uint8) {
	if len(it.stack) > 0 {
		it.stack[len(it.stack)-1].index++
	}
	it.binaryPath = path
	it.stack = append(it.stack, state)
}

func (it *nodeIterator) pop() {
	parent := it.stack[len(it.stack)-1]
	it.binaryPath = it.binaryPath[:parent.pathlen]
	it.stack = it.stack[:len(it.stack)-1]
}

func compareNodes(a, b NodeIterator) int {
	if cmp := bytes.Compare(a.Path(), b.Path()); cmp != 0 {
		return cmp
	}
	if a.Leaf() && !b.Leaf() {
		return -1
	} else if b.Leaf() && !a.Leaf() {
		return 1
	}
	if cmp := bytes.Compare(a.Hash().Bytes(), b.Hash().Bytes()); cmp != 0 {
		return cmp
	}
	if a.Leaf() && b.Leaf() {
		return bytes.Compare(a.LeafBlob(), b.LeafBlob())
	}
	return 0
}

type differenceIterator struct {
	a, b  NodeIterator // Nodes returned are those in b - a.
	eof   bool         // Indicates a has run out of elements
	count int          // Number of nodes scanned on either trie
}

// NewDifferenceIterator constructs a NodeIterator that iterates over elements in b that
// are not in a. Returns the iterator, and a pointer to an integer recording the number
// of nodes seen.
func NewDifferenceIterator(a, b NodeIterator) (NodeIterator, *int) {
	a.Next(true)
	it := &differenceIterator{
		a: a,
		b: b,
	}
	return it, &it.count
}

func (it *differenceIterator) Hash() common.Hash {
	return it.b.Hash()
}

func (it *differenceIterator) Parent() common.Hash {
	return it.b.Parent()
}

func (it *differenceIterator) Leaf() bool {
	return it.b.Leaf()
}

func (it *differenceIterator) LeafKey() []byte {
	return it.b.LeafKey()
}

func (it *differenceIterator) LeafBlob() []byte {
	return it.b.LeafBlob()
}

func (it *differenceIterator) LeafProof() [][]byte {
	return it.b.LeafProof()
}

func (it *differenceIterator) Path() []byte {
	return it.b.Path()
}

func (it *differenceIterator) AddResolver(resolver ethdb.KeyValueStore) {
	panic("not implemented")
}

func (it *differenceIterator) Next(bool) bool {
	// Invariants:
	// - We always advance at least one element in b.
	// - At the start of this function, a's path is lexically greater than b's.
	if !it.b.Next(true) {
		return false
	}
	it.count++

	if it.eof {
		// a has reached eof, so we just return all elements from b
		return true
	}

	for {
		switch compareNodes(it.a, it.b) {
		case -1:
			// b jumped past a; advance a
			if !it.a.Next(true) {
				it.eof = true
				return true
			}
			it.count++
		case 1:
			// b is before a
			return true
		case 0:
			// a and b are identical; skip this whole subtree if the nodes have hashes
			hasHash := it.a.Hash() == common.Hash{}
			if !it.b.Next(hasHash) {
				return false
			}
			it.count++
			if !it.a.Next(hasHash) {
				it.eof = true
				return true
			}
			it.count++
		}
	}
}

func (it *differenceIterator) Error() error {
	if err := it.a.Error(); err != nil {
		return err
	}
	return it.b.Error()
}

type nodeIteratorHeap []NodeIterator

func (h nodeIteratorHeap) Len() int { return len(h) }
func (h nodeIteratorHeap) Less(i, j int) bool {
	return compareNodes(h[i], h[j]) < 0
}
func (h nodeIteratorHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *nodeIteratorHeap) Push(x interface{}) { *h = append(*h, x.(NodeIterator)) }
func (h *nodeIteratorHeap) Pop() interface{} {
	n := len(*h)
	x := (*h)[n-1]
	*h = (*h)[0 : n-1]
	return x
}

type unionIterator struct {
	items *nodeIteratorHeap // Nodes returned are the union of the ones in these iterators
	count int               // Number of nodes scanned across all tries
}

// NewUnionIterator constructs a NodeIterator that iterates over elements in the union
// of the provided NodeIterators. Returns the iterator, and a pointer to an integer
// recording the number of nodes visited.
func NewUnionIterator(iters []NodeIterator) (NodeIterator, *int) {
	h := make(nodeIteratorHeap, len(iters))
	copy(h, iters)
	heap.Init(&h)

	ui := &unionIterator{items: &h}
	return ui, &ui.count
}

func (it *unionIterator) Hash() common.Hash {
	return (*it.items)[0].Hash()
}

func (it *unionIterator) Parent() common.Hash {
	return (*it.items)[0].Parent()
}

func (it *unionIterator) Leaf() bool {
	return (*it.items)[0].Leaf()
}

func (it *unionIterator) LeafKey() []byte {
	return (*it.items)[0].LeafKey()
}

func (it *unionIterator) LeafBlob() []byte {
	return (*it.items)[0].LeafBlob()
}

func (it *unionIterator) LeafProof() [][]byte {
	return (*it.items)[0].LeafProof()
}

func (it *unionIterator) Path() []byte {
	return (*it.items)[0].Path()
}

func (it *unionIterator) AddResolver(resolver ethdb.KeyValueStore) {
	panic("not implemented")
}

// Next returns the next node in the union of tries being iterated over.
//
// It does this by maintaining a heap of iterators, sorted by the iteration
// order of their next elements, with one entry for each source trie. Each
// time Next() is called, it takes the least element from the heap to return,
// advancing any other iterators that also point to that same element. These
// iterators are called with descend=false, since we know that any nodes under
// these nodes will also be duplicates, found in the currently selected iterator.
// Whenever an iterator is advanced, it is pushed back into the heap if it still
// has elements remaining.
//
// In the case that descend=false - eg, we're asked to ignore all subnodes of the
// current node - we also advance any iterators in the heap that have the current
// path as a prefix.
func (it *unionIterator) Next(descend bool) bool {
	if len(*it.items) == 0 {
		return false
	}

	// Get the next key from the union
	least := heap.Pop(it.items).(NodeIterator)

	// Skip over other nodes as long as they're identical, or, if we're not descending, as
	// long as they have the same prefix as the current node.
	for len(*it.items) > 0 && ((!descend && bytes.HasPrefix((*it.items)[0].Path(), least.Path())) || compareNodes(least, (*it.items)[0]) == 0) {
		skipped := heap.Pop(it.items).(NodeIterator)
		// Skip the whole subtree if the nodes have hashes; otherwise just skip this node
		if skipped.Next(skipped.Hash() == common.Hash{}) {
			it.count++
			// If there are more elements, push the iterator back on the heap
			heap.Push(it.items, skipped)
		}
	}
	if least.Next(descend) {
		it.count++
		heap.Push(it.items, least)
	}
	return len(*it.items) > 0
}

func (it *unionIterator) Error() error {
	for i := 0; i < len(*it.items); i++ {
		if err := (*it.items)[i].Error(); err != nil {
			return err
		}
	}
	return nil
}
