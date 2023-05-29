package zktrie

import (
	"bytes"

	itrie "github.com/scroll-tech/zktrie/trie"
	itypes "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/ethdb"
)

type ProofTracer struct {
	trie           *SecureTrie
	deletionTracer map[itypes.Hash]struct{}
	rawPaths       map[string][]*itrie.Node
	emptyTermPaths map[string][]*itrie.Node
}

// NewProofTracer create a proof tracer object
func (t *SecureTrie) NewProofTracer() *ProofTracer {
	return &ProofTracer{
		trie: t,
		// always consider 0 is "deleted"
		deletionTracer: map[itypes.Hash]struct{}{itypes.HashZero: {}},
		rawPaths:       make(map[string][]*itrie.Node),
		emptyTermPaths: make(map[string][]*itrie.Node),
	}
}

// Merge merge the input tracer into current and return current tracer
func (t *ProofTracer) Merge(another *ProofTracer) *ProofTracer {

	// sanity checking
	if !bytes.Equal(t.trie.Hash().Bytes(), another.trie.Hash().Bytes()) {
		panic("can not merge two proof tracer base on different trie")
	}

	for k := range another.deletionTracer {
		t.deletionTracer[k] = struct{}{}
	}

	for k, v := range another.rawPaths {
		t.rawPaths[k] = v
	}

	return t
}

// GetDeletionProofs generate current deletionTracer and collect deletion proofs
// which is possible to be used from all rawPaths, which enabling witness generator
// to predict the final state root after executing any deletion
// along any of the rawpath, no matter of the deletion occurs in any position of the mpt ops
// Note the collected sibling node has no key along with it since witness generator would
// always decode the node for its purpose
func (t *ProofTracer) GetDeletionProofs() ([][]byte, error) {

	retMap := map[itypes.Hash][]byte{}

	// check each path: reversively, skip the final leaf node
	for _, path := range t.rawPaths {

		checkPath := path[:len(path)-1]
		for i := len(checkPath); i > 0; i-- {
			n := checkPath[i-1]
			_, deletedL := t.deletionTracer[*n.ChildL]
			_, deletedR := t.deletionTracer[*n.ChildR]
			if deletedL && deletedR {
				nodeHash, _ := n.NodeHash()
				t.deletionTracer[*nodeHash] = struct{}{}
			} else {
				var siblingHash *itypes.Hash
				if deletedL {
					siblingHash = n.ChildR
				} else if deletedR {
					siblingHash = n.ChildL
				}
				if siblingHash != nil {
					sibling, err := t.trie.zktrie.Tree().GetNode(siblingHash)
					if err != nil {
						return nil, err
					}
					if sibling.Type != itrie.NodeTypeEmpty {
						retMap[*siblingHash] = sibling.Value()
					}
				}
				break
			}
		}
	}

	var ret [][]byte
	for _, bt := range retMap {
		ret = append(ret, bt)
	}

	return ret, nil
}

// MarkDeletion mark a key has been involved into deletion
func (t *ProofTracer) MarkDeletion(key []byte) {
	if path, existed := t.emptyTermPaths[string(key)]; existed {
		// copy empty node terminated path for final scanning
		t.rawPaths[string(key)] = path
	} else if path, existed = t.rawPaths[string(key)]; existed {
		// sanity check
		leafNode := path[len(path)-1]

		if leafNode.Type != itrie.NodeTypeLeaf {
			panic("all path recorded in proofTrace should be ended with leafNode")
		}

		nodeHash, _ := leafNode.NodeHash()
		t.deletionTracer[*nodeHash] = struct{}{}
	}
}

// Prove act the same as zktrie.Prove, while also collect the raw path
// for collecting deletion proofs in a post-work
func (t *ProofTracer) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
	var mptPath []*itrie.Node
	err := t.trie.zktrie.ProveWithDeletion(key, fromLevel,
		func(n *itrie.Node) error {
			nodeHash, err := n.NodeHash()
			if err != nil {
				return err
			}

			if n.Type == itrie.NodeTypeLeaf {
				preImage := t.trie.GetKey(hashKeyToKeybytes(n.NodeKey))
				if len(preImage) > 0 {
					n.KeyPreimage = &itypes.Byte32{}
					copy(n.KeyPreimage[:], preImage)
				}
			} else if n.Type == itrie.NodeTypeParent {
				mptPath = append(mptPath, n)
			} else if n.Type == itrie.NodeTypeEmpty {
				// empty node is considered as "unhit" but it should be also being added
				// into a temporary slot for possibly being marked as deletion later
				mptPath = append(mptPath, n)
				t.emptyTermPaths[string(key)] = mptPath
			}

			return proofDb.Put(nodeHash[:], n.Value())
		},
		func(n *itrie.Node, _ *itrie.Node) {
			// only "hit" path (i.e. the leaf node corresponding the input key can be found)
			// would be add into tracer
			mptPath = append(mptPath, n)
			t.rawPaths[string(key)] = mptPath
		},
	)
	if err != nil {
		return err
	}
	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	return proofDb.Put(magicHash, itrie.ProofMagicBytes())
}
