package zktrie

import (
	"fmt"
)

// Prove constructs a merkle proof for SMT, it respect the protocol used by the ethereum-trie
// but save the node data with a compact form
func (mt *ZkTrieImpl) Prove(kHash *Hash, fromLevel uint, writeNode func(*Node) error) error {
	// force root hash calculation if needed
	if _, err := mt.Root(); err != nil {
		return err
	}

	mt.lock.RLock()
	defer mt.lock.RUnlock()

	path := getPath(mt.maxLevels, kHash[:])
	var nodes []*Node
	var lastN *Node
	tn := mt.rootKey
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.getNode(tn)
		if err != nil {
			fmt.Println("get node fail", err, tn.Hex(),
				lastN.ChildL.Hex(),
				lastN.ChildR.Hex(),
				path,
				i,
			)
			return err
		}
		nodeHash := tn
		lastN = n

		finished := true
		switch n.Type {
		case NodeTypeEmpty_New:
		case NodeTypeLeaf_New:
			// notice even we found a leaf whose entry didn't match the expected k,
			// we still include it as the proof of absence
		case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
			finished = false
			if path[i] {
				tn = n.ChildR
			} else {
				tn = n.ChildL
			}
		case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
			panic("encounter deprecated node types")
		default:
			return ErrInvalidNodeFound
		}

		nCopy := n.Copy()
		nCopy.nodeHash = nodeHash
		nodes = append(nodes, nCopy)
		if finished {
			break
		}
	}

	for _, n := range nodes {
		if fromLevel > 0 {
			fromLevel--
			continue
		}

		// TODO: notice here we may have broken some implicit on the proofDb:
		// the key is not kecca(value) and it even can not be derived from
		// the value by any means without a actually decoding
		if err := writeNode(n); err != nil {
			return err
		}
	}

	return nil
}
