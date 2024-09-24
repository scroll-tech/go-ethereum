package zktrie

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/common"
)

// NodeType defines the type of node in the MT.
type NodeType byte

const (
	// NodeTypeParent indicates the type of parent Node that has children.
	NodeTypeParent NodeType = 0
	// NodeTypeLeaf indicates the type of a leaf Node that contains a key &
	// value.
	NodeTypeLeaf NodeType = 1
	// NodeTypeEmpty indicates the type of an empty Node.
	NodeTypeEmpty NodeType = 2

	// DBEntryTypeRoot indicates the type of a DB entry that indicates the
	// current Root of a MerkleTree
	DBEntryTypeRoot NodeType = 3

	NodeTypeLeaf_New  NodeType = 4
	NodeTypeEmpty_New NodeType = 5
	// branch node for both child are terminal nodes
	NodeTypeBranch_0 NodeType = 6
	// branch node for left child is terminal node and right child is branch
	NodeTypeBranch_1 NodeType = 7
	// branch node for left child is branch node and right child is terminal
	NodeTypeBranch_2 NodeType = 8
	// branch node for both child are branch nodes
	NodeTypeBranch_3 NodeType = 9
)

// DeduceUploadType deduce a new branch type from current branch when one of its child become non-terminal
func (n NodeType) DeduceUpgradeType(goRight bool) NodeType {
	if goRight {
		switch n {
		case NodeTypeBranch_0:
			return NodeTypeBranch_1
		case NodeTypeBranch_1:
			return n
		case NodeTypeBranch_2, NodeTypeBranch_3:
			return NodeTypeBranch_3
		}
	} else {
		switch n {
		case NodeTypeBranch_0:
			return NodeTypeBranch_2
		case NodeTypeBranch_1, NodeTypeBranch_3:
			return NodeTypeBranch_3
		case NodeTypeBranch_2:
			return n
		}
	}

	panic(fmt.Errorf("invalid NodeType: %d", n))
}

// DeduceDowngradeType deduce a new branch type from current branch when one of its child become terminal
func (n NodeType) DeduceDowngradeType(atRight bool) NodeType {
	if atRight {
		switch n {
		case NodeTypeBranch_1:
			return NodeTypeBranch_0
		case NodeTypeBranch_3:
			return NodeTypeBranch_2
		case NodeTypeBranch_0, NodeTypeBranch_2:
			panic(fmt.Errorf("can not downgrade a node with terminal child (%d)", n))
		}
	} else {
		switch n {
		case NodeTypeBranch_3:
			return NodeTypeBranch_1
		case NodeTypeBranch_2:
			return NodeTypeBranch_0
		case NodeTypeBranch_0, NodeTypeBranch_1:
			panic(fmt.Errorf("can not downgrade a node with terminal child (%d)", n))
		}
	}
	panic(fmt.Errorf("invalid NodeType: %d", n))
}

// Node is the struct that represents a node in the MT. The node should not be
// modified after creation because the cached key won't be updated.
type Node struct {
	// Type is the type of node in the tree.
	Type NodeType
	// ChildL is the node hash of the left child of a parent node.
	ChildL *Hash
	// ChildR is the node hash of the right child of a parent node.
	ChildR *Hash
	// NodeKey is the node's key stored in a leaf node.
	NodeKey *Hash
	// ValuePreimage can store at most 256 byte32 as fields (represnted by BIG-ENDIAN integer)
	// and the first 24 can be compressed (each bytes32 consider as 2 fields), in hashing the compressed
	// elemments would be calculated first
	ValuePreimage []Byte32
	// CompressedFlags use each bit for indicating the compressed flag for the first 24 fields
	CompressedFlags uint32
	// nodeHash is the cache of the hash of the node to avoid recalculating
	nodeHash *Hash
	// valueHash is the cache of the hash of valuePreimage to avoid recalculating, only valid for leaf node
	valueHash *Hash
	// KeyPreimage is the original key value that derives the NodeKey, kept here only for proof
	KeyPreimage *Byte32
}

// NewLeafNode creates a new leaf node.
func NewLeafNode(k *Hash, valueFlags uint32, valuePreimage []Byte32) *Node {
	return &Node{Type: NodeTypeLeaf_New, NodeKey: k, CompressedFlags: valueFlags, ValuePreimage: valuePreimage}
}

// NewParentNode creates a new parent node.
func NewParentNode(ntype NodeType, childL *Hash, childR *Hash) *Node {
	return &Node{Type: ntype, ChildL: childL, ChildR: childR}
}

// NewEmptyNode creates a new empty node.
func NewEmptyNode() *Node {
	return &Node{Type: NodeTypeEmpty_New}
}

// NewNodeFromBytes creates a new node by parsing the input []byte.
func NewNodeFromBytes(b []byte) (*Node, error) {
	var n Node
	if err := n.SetBytes(b); err != nil {
		return nil, err
	}
	return &n, nil
}

// LeafHash computes the key of a leaf node given the hIndex and hValue of the
// entry of the leaf.
func LeafHash(k, v *Hash) (*Hash, error) {
	return HashElemsWithDomain(big.NewInt(int64(NodeTypeLeaf_New)), k.BigInt(), v.BigInt())
}

func (n *Node) SetBytes(b []byte) error {
	if len(b) < 1 {
		return ErrNodeBytesBadSize
	}
	nType := NodeType(b[0])
	b = b[1:]
	switch nType {
	case NodeTypeParent, NodeTypeBranch_0,
		NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
		if len(b) != 2*HashByteLen {
			return ErrNodeBytesBadSize
		}

		childL := n.ChildL
		childR := n.ChildR

		if childL == nil {
			childL = NewHashFromBytes(b[:HashByteLen])
		} else {
			childL.SetBytes(b[:HashByteLen])
		}

		if childR == nil {
			childR = NewHashFromBytes(b[HashByteLen : HashByteLen*2])
		} else {
			childR.SetBytes(b[HashByteLen : HashByteLen*2])
		}

		*n = Node{
			Type:   nType,
			ChildL: childL,
			ChildR: childR,
		}
	case NodeTypeLeaf, NodeTypeLeaf_New:
		if len(b) < HashByteLen+4 {
			return ErrNodeBytesBadSize
		}
		nodeKey := NewHashFromBytes(b[0:HashByteLen])
		mark := binary.LittleEndian.Uint32(b[HashByteLen : HashByteLen+4])
		preimageLen := int(mark & 255)
		compressedFlags := mark >> 8
		valuePreimage := slices.Grow(n.ValuePreimage[0:], preimageLen)
		curPos := HashByteLen + 4
		if len(b) < curPos+preimageLen*32+1 {
			return ErrNodeBytesBadSize
		}
		for i := 0; i < preimageLen; i++ {
			var byte32 Byte32
			copy(byte32[:], b[i*32+curPos:(i+1)*32+curPos])
			valuePreimage = append(valuePreimage, byte32)
		}
		curPos += preimageLen * 32
		preImageSize := int(b[curPos])
		curPos += 1

		var keyPreimage *Byte32
		if preImageSize != 0 {
			if len(b) < curPos+preImageSize {
				return ErrNodeBytesBadSize
			}

			keyPreimage = n.KeyPreimage
			if keyPreimage == nil {
				keyPreimage = new(Byte32)
			}
			copy(keyPreimage[:], b[curPos:curPos+preImageSize])
		}

		*n = Node{
			Type:            nType,
			NodeKey:         nodeKey,
			CompressedFlags: compressedFlags,
			ValuePreimage:   valuePreimage,
			KeyPreimage:     keyPreimage,
		}
	case NodeTypeEmpty, NodeTypeEmpty_New:
		*n = Node{Type: nType}
	default:
		return ErrInvalidNodeFound
	}
	return nil
}

// IsTerminal returns if the node is 'terminated', i.e. empty or leaf node
func (n *Node) IsTerminal() bool {
	switch n.Type {
	case NodeTypeEmpty_New, NodeTypeLeaf_New:
		return true
	case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
		return false
	case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
		panic("encounter deprecated node types")
	default:
		panic(fmt.Errorf("encounter unknown node types %d", n.Type))
	}

}

// NodeHash computes the hash digest of the node by hashing the content in a
// specific way for each type of node.  This key is used as the hash of the
// Merkle tree for each node.
func (n *Node) NodeHash() (*Hash, error) {
	if n.nodeHash == nil { // Cache the key to avoid repeated hash computations.
		// NOTE: We are not using the type to calculate the hash!
		switch n.Type {
		case NodeTypeBranch_0,
			NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3: // H(ChildL || ChildR)
			var err error
			n.nodeHash, err = HashElemsWithDomain(big.NewInt(int64(n.Type)),
				n.ChildL.BigInt(), n.ChildR.BigInt())
			if err != nil {
				return nil, err
			}
		case NodeTypeLeaf_New:
			var err error
			n.valueHash, err = HandlingElemsAndByte32(n.CompressedFlags, n.ValuePreimage)
			if err != nil {
				return nil, err
			}

			n.nodeHash, err = LeafHash(n.NodeKey, n.valueHash)
			if err != nil {
				return nil, err
			}

		case NodeTypeEmpty_New: // Zero
			n.nodeHash = &HashZero
		case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
			panic("encounter deprecated node types")
		default:
			n.nodeHash = &HashZero
		}
	}
	return n.nodeHash, nil
}

// ValueHash computes the hash digest of the value stored in the leaf node. For
// other node types, it returns the zero hash.
func (n *Node) ValueHash() (*Hash, error) {
	if n.Type != NodeTypeLeaf_New {
		return &HashZero, nil
	}
	if _, err := n.NodeHash(); err != nil {
		return nil, err
	}
	return n.valueHash, nil
}

// Data returns the wrapped data inside LeafNode and cast them into bytes
// for other node type it just return nil
func (n *Node) Data() []byte {
	switch n.Type {
	case NodeTypeLeaf_New:
		var data []byte
		hdata := (*reflect.SliceHeader)(unsafe.Pointer(&data))
		//TODO: uintptr(reflect.ValueOf(n.ValuePreimage).UnsafePointer()) should be more elegant but only available until go 1.18
		hdata.Data = uintptr(unsafe.Pointer(&n.ValuePreimage[0]))
		hdata.Len = 32 * len(n.ValuePreimage)
		hdata.Cap = hdata.Len
		return data
	default:
		return nil
	}
}

// CanonicalValue returns the byte form of a node required to be persisted, and strip unnecessary fields
// from the encoding (current only KeyPreimage for Leaf node) to keep a minimum size for content being
// stored in backend storage
func (n *Node) CanonicalValue() []byte {
	switch n.Type {
	case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3: // {Type || ChildL || ChildR}
		bytes := []byte{byte(n.Type)}
		bytes = append(bytes, n.ChildL.Bytes()...)
		bytes = append(bytes, n.ChildR.Bytes()...)
		return bytes
	case NodeTypeLeaf_New: // {Type || Data...}
		bytes := []byte{byte(n.Type)}
		bytes = append(bytes, n.NodeKey.Bytes()...)
		tmp := make([]byte, 4)
		compressedFlag := (n.CompressedFlags << 8) + uint32(len(n.ValuePreimage))
		binary.LittleEndian.PutUint32(tmp, compressedFlag)
		bytes = append(bytes, tmp...)
		for _, elm := range n.ValuePreimage {
			bytes = append(bytes, elm[:]...)
		}
		bytes = append(bytes, 0)
		return bytes
	case NodeTypeEmpty_New: // { Type }
		return []byte{byte(n.Type)}
	case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
		panic("encounter deprecated node types")
	default:
		return []byte{}
	}
}

// Value returns the encoded bytes of a node, include all information of it
func (n *Node) Value() []byte {
	outBytes := n.CanonicalValue()
	switch n.Type {
	case NodeTypeLeaf_New: // {Type || Data...}
		if n.KeyPreimage != nil {
			outBytes[len(outBytes)-1] = byte(len(n.KeyPreimage))
			outBytes = append(outBytes, n.KeyPreimage[:]...)
		}
	}

	return outBytes
}

// String outputs a string representation of a node (different for each type).
func (n *Node) String() string {
	switch n.Type {
	// {Type || ChildL || ChildR}
	case NodeTypeBranch_0:
		return fmt.Sprintf("Parent L(t):%s R(t):%s", n.ChildL, n.ChildR)
	case NodeTypeBranch_1:
		return fmt.Sprintf("Parent L(t):%s R:%s", n.ChildL, n.ChildR)
	case NodeTypeBranch_2:
		return fmt.Sprintf("Parent L:%s R(t):%s", n.ChildL, n.ChildR)
	case NodeTypeBranch_3:
		return fmt.Sprintf("Parent L:%s R:%s", n.ChildL, n.ChildR)
	case NodeTypeLeaf_New: // {Type || Data...}
		return fmt.Sprintf("Leaf I:%v Items: %d, First:%v", n.NodeKey, len(n.ValuePreimage), n.ValuePreimage[0])
	case NodeTypeEmpty_New: // {}
		return "Empty"
	case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
		return "deprecated Node"
	default:
		return "Invalid Node"
	}
}

// Copy creates a new Node instance from the given node
func (n *Node) Copy() *Node {
	newNode, err := NewNodeFromBytes(n.Value())
	if err != nil {
		panic("failed to copy trie node")
	}
	return newNode
}

type ChildResolver struct{}

// ForEach iterates over the children of a node and calls the given function
// note: original implementation from geth works recursively, but our Node definition
// doesn't allow that. So we only iterate over the children of the current node, which
// should be fine.
func (r ChildResolver) ForEach(node []byte, onChild func(common.Hash)) {
	n, err := NewNodeFromBytes(node)
	if err != nil {
		panic(fmt.Sprintf("node %x: %v", node, err))
	}

	var childHash common.Hash
	if n.ChildL != nil {
		childHash.SetBytes(n.ChildL.Bytes())
		onChild(childHash)
	}
	if n.ChildR != nil {
		childHash.SetBytes(n.ChildR.Bytes())
		onChild(childHash)
	}
}
