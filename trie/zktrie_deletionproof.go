package trie

import (
	"bytes"

	zktrie "github.com/scroll-tech/zktrie/trie"
	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/ethdb"
)

// Pick Node from its hash directly from database, notice it has different
// interface with the function of same name in `trie`
func (t *ZkTrie) TryGetNode(nodeHash *zkt.Hash) (*zktrie.Node, error) {
	if bytes.Equal(nodeHash[:], zkt.HashZero[:]) {
		return zktrie.NewEmptyNode(), nil
	}
	nBytes, err := t.db.Get(nodeHash[:])
	if err == zktrie.ErrKeyNotFound {
		return nil, zktrie.ErrKeyNotFound
	} else if err != nil {
		return nil, err
	}
	return zktrie.NewNodeFromBytes(nBytes)
}

type deletionProofTracer struct {
	*ZkTrie
	deletionTracer map[zkt.Hash]struct{}
	proofs         map[zkt.Hash][]byte
}

// NewDeletionTracer create a deletion tracer object
func (t *ZkTrie) NewDeletionTracer() *deletionProofTracer {
	return &deletionProofTracer{
		ZkTrie:         t,
		deletionTracer: map[zkt.Hash]struct{}{zkt.HashZero: {}},
		proofs:         make(map[zkt.Hash][]byte),
	}
}

// GetProofs collect the proofs
func (t *deletionProofTracer) GetProofs() (ret [][]byte) {
	for _, bt := range t.proofs {
		ret = append(ret, bt)
	}
	return
}

// ProveWithDeletion act the same as Prove, while also trace the possible sibling node
// from a series deletion records, the collected deletion proofs being collect
// enabling witness generator to predict the final state root after executing any deletion
// in the traced series, no matter of the deletion occurs in any position of the mpt ops
// Note the collected sibling node has no key along with it since witness generator would
// always decode the node for its purpose
func (t *deletionProofTracer) ProveWithDeletion(key []byte, proofDb ethdb.KeyValueWriter) error {
	var mptPath []*zktrie.Node
	err := t.ZkTrie.ProveWithDeletion(key, 0,
		func(n *zktrie.Node) error {
			nodeHash, err := n.NodeHash()
			if err != nil {
				return err
			}

			if n.Type == zktrie.NodeTypeLeaf {
				preImage := t.GetKey(n.NodeKey.Bytes())
				if len(preImage) > 0 {
					n.KeyPreimage = &zkt.Byte32{}
					copy(n.KeyPreimage[:], preImage)
				}
			} else if n.Type == zktrie.NodeTypeParent {
				mptPath = append(mptPath, n)
			}

			return proofDb.Put(nodeHash[:], n.Value())
		},
		func(delNode *zktrie.Node, n *zktrie.Node) {
			nodeHash, _ := delNode.NodeHash()
			t.deletionTracer[*nodeHash] = struct{}{}
			// the sibling for each leaf should be unique except for EmptyNode
			if n != nil && n.Type != zktrie.NodeTypeEmpty {
				nodeHash, _ := n.NodeHash()
				t.proofs[*nodeHash] = n.Value()
			}

		},
	)
	if err != nil {
		return err
	}
	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	err = proofDb.Put(magicHash, zktrie.ProofMagicBytes())
	if err != nil {
		return err
	}

	// now handle mptpath reversively
	for i := len(mptPath); i > 0; i-- {
		n := mptPath[i-1]
		_, deletedL := t.deletionTracer[*n.ChildL]
		_, deletedR := t.deletionTracer[*n.ChildR]
		if deletedL && deletedR {
			nodeHash, _ := n.NodeHash()
			t.deletionTracer[*nodeHash] = struct{}{}
		} else {
			if i != len(mptPath) {
				var siblingHash *zkt.Hash
				if deletedL {
					siblingHash = n.ChildR
				} else if deletedR {
					siblingHash = n.ChildL
				}
				if siblingHash != nil {
					sibling, err := t.TryGetNode(siblingHash)
					if err != nil {
						return err
					}
					if sibling.Type != zktrie.NodeTypeEmpty {
						t.proofs[*siblingHash] = sibling.Value()
					}
				}
			}
			return nil
		}
	}

	return nil
}
