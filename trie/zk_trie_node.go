package trie

import (
	"encoding/binary"
	"fmt"
	"math/big"

	zkt "github.com/scroll-tech/go-ethereum/core/types/zktrie"
)

// NodeType defines the type of node in the MT.
type NodeType byte

var byte32Zero zkt.Byte32

const (
	// NodeTypeMiddle indicates the type of middle Node that has children.
	NodeTypeMiddle NodeType = 0
	// NodeTypeLeaf indicates the type of a leaf Node that contains a key &
	// value.
	NodeTypeLeaf NodeType = 1
	// NodeTypeEmpty indicates the type of an empty Node.
	NodeTypeEmpty NodeType = 2

	// DBEntryTypeRoot indicates the type of a DB entry that indicates the
	// current Root of a MerkleTree
	DBEntryTypeRoot NodeType = 3
)

// Node is the struct that represents a node in the MT. The node should not be
// modified after creation because the cached key won't be updated.
type Node struct {
	// Type is the type of node in the tree.
	Type NodeType
	// ChildL is the left child of a middle node.
	ChildL *zkt.Hash
	// ChildR is the right child of a middle node.
	ChildR *zkt.Hash
	// Entry is the data stored in a leaf node.
	Entry [2]*zkt.Hash
	// key is a cache used to avoid recalculating key
	key              *zkt.Hash
	KeyPreimage      *zkt.Byte32
	ValuePreimageLen uint32
	ValuePreimage    []byte
}

// NewNodeLeaf creates a new leaf node.
func NewNodeLeaf(k, v *zkt.Hash, keyPreimage *zkt.Byte32, valuePreimage []byte) *Node {
	return &Node{Type: NodeTypeLeaf, Entry: [2]*zkt.Hash{k, v}, KeyPreimage: keyPreimage, ValuePreimageLen: uint32(len(valuePreimage)), ValuePreimage: valuePreimage[:]}
}

// NewNodeMiddle creates a new middle node.
func NewNodeMiddle(childL *zkt.Hash, childR *zkt.Hash) *Node {
	return &Node{Type: NodeTypeMiddle, ChildL: childL, ChildR: childR}
}

// NewNodeEmpty creates a new empty node.
func NewNodeEmpty() *Node {
	return &Node{Type: NodeTypeEmpty}
}

// NewNodeFromBytes creates a new node by parsing the input []byte.
func NewNodeFromBytes(b []byte) (*Node, error) {
	if len(b) < 1 {
		return nil, ErrNodeBytesBadSize
	}
	n := Node{Type: NodeType(b[0])}
	b = b[1:]
	switch n.Type {
	case NodeTypeMiddle:
		if len(b) != 2*zkt.ElemBytesLen {
			return nil, ErrNodeBytesBadSize
		}
		n.ChildL, n.ChildR = &zkt.Hash{}, &zkt.Hash{}
		copy(n.ChildL[:], b[:zkt.ElemBytesLen])
		copy(n.ChildR[:], b[zkt.ElemBytesLen:zkt.ElemBytesLen*2])
	case NodeTypeLeaf:
		if len(b) < 4*zkt.ElemBytesLen+4 {
			return nil, ErrNodeBytesBadSize
		}
		n.Entry = [2]*zkt.Hash{{}, {}}
		copy(n.Entry[0][:], b[0:32])
		copy(n.Entry[1][:], b[32:64])
		n.KeyPreimage = &zkt.Byte32{}
		copy(n.KeyPreimage[:], b[64:96])
		n.ValuePreimageLen = binary.LittleEndian.Uint32(b[96:100])
		n.ValuePreimage = make([]byte, n.ValuePreimageLen)
		copy(n.ValuePreimage[:], b[100:100+n.ValuePreimageLen])
	case NodeTypeEmpty:
		break
	default:
		return nil, ErrInvalidNodeFound
	}
	return &n, nil
}

// LeafKey computes the key of a leaf node given the hIndex and hValue of the
// entry of the leaf.
func LeafKey(k, v *zkt.Hash) (*zkt.Hash, error) {
	return zkt.HashElemsKey(big.NewInt(1), k.BigInt(), v.BigInt())
}

// Key computes the key of the node by hashing the content in a specific way
// for each type of node.  This key is used as the hash of the merklee tree for
// each node.
func (n *Node) Key() (*zkt.Hash, error) {
	if n.key == nil { // Cache the key to avoid repeated hash computations.
		// NOTE: We are not using the type to calculate the hash!
		switch n.Type {
		case NodeTypeMiddle: // H(ChildL || ChildR)
			var err error
			n.key, err = zkt.HashElems(n.ChildL.BigInt(), n.ChildR.BigInt())
			if err != nil {
				return nil, err
			}
		case NodeTypeLeaf:
			var err error
			n.key, err = LeafKey(n.Entry[0], n.Entry[1])
			if err != nil {
				return nil, err
			}
		case NodeTypeEmpty: // Zero
			n.key = &zkt.HashZero
		default:
			n.key = &zkt.HashZero
		}
	}
	return n.key, nil
}

// Value returns the value of the node.  This is the content that is stored in
// the backend database.
func (n *Node) Value() []byte {
	switch n.Type {
	case NodeTypeMiddle: // {Type || ChildL || ChildR}
		bytes := []byte{byte(n.Type)}
		bytes = append(bytes, n.ChildL[:]...)
		bytes = append(bytes, n.ChildR[:]...)
		return bytes
	case NodeTypeLeaf: // {Type || Data...}
		bytes := []byte{byte(n.Type)}
		bytes = append(bytes, n.Entry[0][:]...)
		bytes = append(bytes, n.Entry[1][:]...)
		bytes = append(bytes, n.KeyPreimage[:]...)
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, n.ValuePreimageLen)
		bytes = append(bytes, tmp...)
		bytes = append(bytes, n.ValuePreimage[:]...)
		return bytes
	case NodeTypeEmpty: // { Type }
		return []byte{byte(n.Type)}
	default:
		return []byte{}
	}
}

// String outputs a string representation of a node (different for each type).
func (n *Node) String() string {
	switch n.Type {
	case NodeTypeMiddle: // {Type || ChildL || ChildR}
		return fmt.Sprintf("Middle L:%s R:%s", n.ChildL, n.ChildR)
	case NodeTypeLeaf: // {Type || Data...}
		return fmt.Sprintf("Leaf I:%v D:%v", n.Entry[0], n.Entry[1])
	case NodeTypeEmpty: // {}
		return "Empty"
	default:
		return "Invalid Node"
	}
}
