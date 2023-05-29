package zktrie

import (
	"bytes"
	"errors"
	"fmt"

	itrie "github.com/scroll-tech/zktrie/trie"
	itypes "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

type Resolver func(*itypes.Hash) (*itrie.Node, error)

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *SecureTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	// omit sibling, which is not required for proving only
	_, err := t.ProveWithDeletion(key, fromLevel, proofDb)
	return err
}

func (t *SecureTrie) ProveWithDeletion(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) (sibling []byte, err error) {
	// standardize the key format, which is the same as trie interface
	key = itypes.ReverseByteOrder(key)
	reverseBitInPlace(key)
	err = t.zktrie.ProveWithDeletion(key, fromLevel,
		func(n *itrie.Node) error {
			nodeHash, err := n.NodeHash()
			if err != nil {
				return err
			}

			if n.Type == itrie.NodeTypeLeaf {
				preImage := t.GetKey(hashKeyToKeybytes(n.NodeKey))
				if len(preImage) > 0 {
					n.KeyPreimage = &itypes.Byte32{}
					copy(n.KeyPreimage[:], preImage)
					//return fmt.Errorf("key preimage not found for [%x] ref %x", n.NodeKey.Bytes(), k.Bytes())
				}
			}
			return proofDb.Put(nodeHash[:], n.Value())
		},
		func(_ *itrie.Node, n *itrie.Node) {
			// the sibling for each leaf should be unique except for EmptyNode
			if n != nil && n.Type != itrie.NodeTypeEmpty {
				sibling = n.Value()
			}
		},
	)
	if err != nil {
		return
	}

	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	err = proofDb.Put(magicHash, itrie.ProofMagicBytes())
	return
}

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
func (t *Trie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	// omit sibling, which is not required for proving only
	_, err := t.ProveWithDeletion(key, fromLevel, proofDb)
	return err
}

// ProveWithDeletion is the implement of Prove, it also return possible sibling node
// (if there is, i.e. the node of key exist and is not the only node in trie)
// so witness generator can predict the final state root after deletion of this key
// the returned sibling node has no key along with it for witness generator must decode
// the node for its purpose
func (t *Trie) ProveWithDeletion(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) (sibling []byte, err error) {
	// standardize the key format, which is the same as trie interface
	key = itypes.ReverseByteOrder(key)
	reverseBitInPlace(key)
	err = t.secureTrie.ProveWithDeletion(key, fromLevel,
		func(n *itrie.Node) error {
			nodeHash, err := n.NodeHash()
			if err != nil {
				return err
			}
			return proofDb.Put(nodeHash[:], n.Value())
		},
		func(_ *itrie.Node, n *itrie.Node) {
			// the sibling for each leaf should be unique except for EmptyNode
			if n != nil && n.Type != itrie.NodeTypeEmpty {
				sibling = n.Value()
			}
		},
	)
	if err != nil {
		return
	}

	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	err = proofDb.Put(magicHash, itrie.ProofMagicBytes())
	return
}

// VerifyProof checks merkle proofs. The given proof must contain the value for
// key in a trie with the given root hash. VerifyProof returns an error if the
// proof contains invalid trie nodes or the wrong value.
func VerifyProof(rootHash common.Hash, key []byte, proofDb ethdb.KeyValueReader) (value []byte, err error) {
	path := keybytesToBinary(key)
	wantHash := StoreHashFromNodeHash(rootHash)
	for i := 0; i < len(path); i++ {
		buf, _ := proofDb.Get(wantHash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node %d (hash %064x) missing", i, wantHash)
		}
		n, err := itrie.NewNodeFromBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %d: %v", i, err)
		}
		switch n.Type {
		case itrie.NodeTypeEmpty:
			return n.Data(), nil
		case itrie.NodeTypeLeaf:
			if bytes.Equal(key, hashKeyToKeybytes(n.NodeKey)) {
				return n.Data(), nil
			}
			// We found a leaf whose entry didn't match hIndex
			return nil, nil
		case itrie.NodeTypeParent:
			if path[i] > 0 {
				wantHash = n.ChildR
			} else {
				wantHash = n.ChildL
			}
		default:
			return nil, itrie.ErrInvalidNodeFound
		}
	}
	return nil, itrie.ErrKeyNotFound
}

// proofToPath converts a merkle proof to trie node path. The main purpose of
// this function is recovering a node path from the merkle proof stream. All
// necessary nodes will be resolved and leave the remaining as hashnode.
//
// The given edge proof is allowed to be an existent or non-existent proof.
func proofToPath(
	rootHash common.Hash,
	root *itrie.Node,
	key []byte,
	resolveNode Resolver,
	cache ethdb.KeyValueStore,
	allowNonExistent bool,
) (*itrie.Node, []byte, error) {
	// If the root node is empty, resolve it first.
	// Root node must be included in the proof.
	if root == nil {
		n, err := resolveNode(StoreHashFromNodeHash(rootHash))
		if err != nil {
			return nil, nil, err
		}
		root = n
	}
	var (
		path        []byte
		err         error
		current     *itrie.Node
		currentHash *itypes.Hash
	)
	path, current, currentHash = keybytesToBinary(key), root, StoreHashFromNodeHash(rootHash)
	for {
		if err = cache.Put(currentHash[:], current.CanonicalValue()); err != nil {
			return nil, nil, err
		}
		switch current.Type {
		case itrie.NodeTypeEmpty:
			// The trie doesn't contain the key. It's possible
			// the proof is a non-existing proof, but at least
			// we can prove all resolved nodes are correct, it's
			// enough for us to prove range.
			if allowNonExistent {
				return root, nil, nil
			}
			return nil, nil, errors.New("the node is not contained in trie")
		case itrie.NodeTypeParent:
			currentHash = current.ChildL
			if path[0] == 1 {
				currentHash = current.ChildR
			}
			current, err = resolveNode(currentHash)
			if err != nil {
				return nil, nil, err
			}
			path = path[1:]
		case itrie.NodeTypeLeaf:
			if bytes.Equal(key, hashKeyToKeybytes(current.NodeKey)) {
				return root, current.Data(), nil
			} else {
				if allowNonExistent {
					return root, nil, nil
				}
				return nil, nil, errors.New("the node is not contained in trie")
			}
		}
	}
}

// hasRightElement returns the indicator whether there exists more elements
// in the right side of the given path. The given path can point to an existent
// key or a non-existent one. This function has the assumption that the whole
// path should already be resolved.
func hasRightElement(node *itrie.Node, key []byte, resolveNode Resolver) bool {
	pos, path := 0, keybytesToBinary(key)
	for {
		switch node.Type {
		case itrie.NodeTypeParent:
			if path[pos] == 0 && !bytes.Equal(node.ChildR[:], itypes.HashZero[:]) {
				return true
			}
			hash := node.ChildL
			if path[pos] == 1 {
				hash = node.ChildR
			}
			node, _ = resolveNode(hash)
			pos += 1
		case itrie.NodeTypeLeaf:
			return bytes.Compare(hashKeyToKeybytes(node.NodeKey), key) > 0
		case itrie.NodeTypeEmpty:
			return false
		default:
			panic(fmt.Sprintf("%T: invalid node: %v", node, node)) // hashnode
		}
	}
}

func unset(h *itypes.Hash, l []byte, r []byte, pos int, resolveNode Resolver, cache ethdb.KeyValueStore) (*itypes.Hash, error) {
	if l == nil && r == nil {
		return &itypes.HashZero, nil
	}
	n, err := resolveNode(h)
	if err != nil {
		return nil, err
	}

	switch n.Type {
	case itrie.NodeTypeEmpty:
		return h, nil
	case itrie.NodeTypeLeaf:
		if (l != nil && bytes.Compare(hashKeyToBinary(n.NodeKey), l) < 0) ||
			(r != nil && bytes.Compare(hashKeyToBinary(n.NodeKey), r) > 0) {
			return h, nil
		}
		return &itypes.HashZero, nil
	case itrie.NodeTypeParent:
		lhash, rhash := n.ChildL, n.ChildR
		if l == nil || l[pos] == 0 {
			var rr []byte = nil
			if r != nil && r[pos] == 0 {
				rr = r
			}
			if lhash, err = unset(n.ChildL, l, rr, pos+1, resolveNode, cache); err != nil {
				return nil, err
			}
		}
		if r == nil || r[pos] == 1 {
			var ll []byte = nil
			if l != nil && l[pos] == 1 {
				ll = l
			}
			if rhash, err = unset(n.ChildR, ll, r, pos+1, resolveNode, cache); err != nil {
				return nil, err
			}
		}
		newParent := itrie.NewParentNode(lhash, rhash)
		if hash, err := newParent.NodeHash(); err != nil {
			return nil, fmt.Errorf("new node hash failed: %v", err)
		} else {
			if err := cache.Put(hash[:], newParent.CanonicalValue()); err != nil {
				return nil, err
			}
			return hash, nil
		}
	default:
		panic(fmt.Sprintf("%T: invalid node: %v", n, n)) // hashnode
	}
}

// unsetInternal removes all internal node references(hashnode, embedded node).
// It should be called after a trie is constructed with two edge paths. Also
// the given boundary keys must be the one used to construct the edge paths.
//
// It's the key step for range proof. All visited nodes should be marked dirty
// since the node content might be modified. Besides it can happen that some
// fullnodes only have one child which is disallowed. But if the proof is valid,
// the missing children will be filled, otherwise it will be thrown anyway.
//
// Note we have the assumption here the given boundary keys are different
// and right is larger than left.
func unsetInternal(h *itypes.Hash, left []byte, right []byte, cache ethdb.KeyValueStore) (*itypes.Hash, error) {
	left, right = keybytesToBinary(left), keybytesToBinary(right)
	return unset(h, left, right, 0, nodeResolver(cache), cache)
}

func nodeResolver(proof ethdb.KeyValueReader) Resolver {
	return func(hash *itypes.Hash) (*itrie.Node, error) {
		buf, _ := proof.Get(hash[:])
		if buf == nil {
			return nil, fmt.Errorf("proof node (hash %064x) missing", hash)
		}
		n, err := itrie.NewNodeFromBytes(buf)
		if err != nil {
			return nil, fmt.Errorf("bad proof node %v", err)
		}
		return n, err
	}
}

// VerifyRangeProof checks whether the given leaf nodes and edge proof
// can prove the given trie leaves range is matched with the specific root.
// Besides, the range should be consecutive (no gap inside) and monotonic
// increasing.
//
// Note the given proof actually contains two edge proofs. Both of them can
// be non-existent proofs. For example the first proof is for a non-existent
// key 0x03, the last proof is for a non-existent key 0x10. The given batch
// leaves are [0x04, 0x05, .. 0x09]. It's still feasible to prove the given
// batch is valid.
//
// The firstKey is paired with firstProof, not necessarily the same as keys[0]
// (unless firstProof is an existent proof). Similarly, lastKey and lastProof
// are paired.
//
// Expect the normal case, this function can also be used to verify the following
// range proofs:
//
//   - All elements proof. In this case the proof can be nil, but the range should
//     be all the leaves in the trie.
//
//   - One element proof. In this case no matter the edge proof is a non-existent
//     proof or not, we can always verify the correctness of the proof.
//
//   - Zero element proof. In this case a single non-existent proof is enough to prove.
//     Besides, if there are still some other leaves available on the right side, then
//     an error will be returned.
//
// Except returning the error to indicate the proof is valid or not, the function will
// also return a flag to indicate whether there exists more accounts/slots in the trie.
//
// Note: This method does not verify that the proof is of minimal form. If the input
// proofs are 'bloated' with neighbour leaves or random data, aside from the 'useful'
// data, then the proof will still be accepted.
func VerifyRangeProof(rootHash common.Hash, kind string, firstKey []byte, lastKey []byte, keys [][]byte, values [][]byte, proof ethdb.KeyValueReader) (bool, error) {
	if len(keys) != len(values) {
		return false, fmt.Errorf("inconsistent proof data, keys: %d, values: %d", len(keys), len(values))
	}
	// Ensure the received batch is monotonic increasing and contains no deletions
	for i := 0; i < len(keys)-1; i++ {
		if bytes.Compare(keys[i], keys[i+1]) >= 0 {
			return false, errors.New("range is not monotonically increasing")
		}
	}
	for _, value := range values {
		if len(value) == 0 {
			return false, errors.New("range contains deletion")
		}
	}
	// Special case, there is no edge proof at all. The given range is expected
	// to be the whole leaf-set in the trie.
	if proof == nil {
		tr := NewStackTrie(nil)
		for index, key := range keys {
			if err := tr.TryUpdateWithKind(kind, key, values[index]); err != nil {
				return false, err
			}
		}
		if have, want := tr.Hash(), rootHash; have != want {
			return false, fmt.Errorf("invalid proof, want hash %x, got %x", want, have)
		}
		return false, nil // No more elements
	}

	trieCache := memorydb.New()

	// Special case, there is a provided edge proof but zero key/value
	// pairs, ensure there are no more accounts / slots in the trie.
	if len(keys) == 0 {
		root, val, err := proofToPath(rootHash, nil, firstKey, nodeResolver(proof), trieCache, true)
		if err != nil {
			return false, err
		}
		if val != nil || hasRightElement(root, firstKey, nodeResolver(trieCache)) {
			return false, errors.New("more entries available")
		}
		return hasRightElement(root, firstKey, nodeResolver(trieCache)), nil
	}
	// Special case, there is only one element and two edge keys are same.
	// In this case, we can't construct two edge paths. So handle it here.
	if len(keys) == 1 && bytes.Equal(firstKey, lastKey) {
		root, val, err := proofToPath(rootHash, nil, firstKey, nodeResolver(proof), trieCache, false)
		if err != nil {
			return false, err
		}
		if !bytes.Equal(firstKey, keys[0]) {
			return false, errors.New("correct proof but invalid key")
		}
		if !bytes.Equal(val, values[0]) {
			return false, errors.New("correct proof but invalid data")
		}
		return hasRightElement(root, firstKey, nodeResolver(trieCache)), nil
	}
	// Ok, in all other cases, we require two edge paths available.
	// First check the validity of edge keys.
	if bytes.Compare(firstKey, lastKey) >= 0 {
		return false, errors.New("invalid edge keys")
	}
	// todo(rjl493456442) different length edge keys should be supported
	if len(firstKey) != len(lastKey) {
		return false, errors.New("inconsistent edge keys")
	}
	if !(bytes.Compare(firstKey, keys[0]) <= 0 && bytes.Compare(keys[len(keys)-1], lastKey) <= 0) {
		return false, errors.New("keys are out of range [firstKey, lastKey]")
	}
	// Convert the edge proofs to edge trie paths. Then we can
	// have the same tree architecture with the original one.
	// For the first edge proof, non-existent proof is allowed.
	root, _, err := proofToPath(rootHash, nil, firstKey, nodeResolver(proof), trieCache, true)
	if err != nil {
		return false, err
	}
	// Pass the root node here, the second path will be merged
	// with the first one. For the last edge proof, non-existent
	// proof is also allowed.
	root, _, err = proofToPath(rootHash, root, lastKey, nodeResolver(proof), trieCache, true)
	if err != nil {
		return false, err
	}
	// Remove all internal references. All the removed parts should
	// be re-filled(or re-constructed) by the given leaves range.
	unsetRootHash, err := unsetInternal(StoreHashFromNodeHash(rootHash), firstKey, lastKey, trieCache)
	if err != nil {
		return false, err
	}
	// Rebuild the trie with the leaf stream, the shape of trie
	// should be same with the original one.
	tr, err := New(common.BytesToHash(unsetRootHash.Bytes()), NewDatabase(trieCache))
	if err != nil {
		return false, err
	}
	for index, key := range keys {
		if err := tr.TryUpdateWithKind(kind, key, values[index]); err != nil {
			return false, err
		}
	}
	if tr.Hash() != rootHash {
		return false, fmt.Errorf("invalid proof, want hash %x, got %x", rootHash, tr.Hash())
	}
	return hasRightElement(root, keys[len(keys)-1], nodeResolver(tr.db)), nil
}
