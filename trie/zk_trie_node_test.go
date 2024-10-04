package trie

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewNode(t *testing.T) {
	t.Run("Test NewEmptyNode", func(t *testing.T) {
		node := NewEmptyNode()
		assert.Equal(t, NodeTypeEmpty_New, node.Type)

		hash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, hash)

		hash, err = node.ValueHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, hash)
	})

	t.Run("Test NewLeafNode", func(t *testing.T) {
		k := NewHashFromBytes(bytes.Repeat([]byte("0"), 32))
		vp := []Byte32{*NewByte32FromBytes(bytes.Repeat([]byte("b"), 32))}
		node := NewLeafNode(k, 1, vp)
		assert.Equal(t, NodeTypeLeaf_New, node.Type)
		assert.Equal(t, uint32(1), node.CompressedFlags)
		assert.Equal(t, vp, node.ValuePreimage)

		hash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, "2536e274d373c4ca79bc85c6aa140fe911eb7fe04939e1311004bbaf3c13c32a", hash.Hex())

		hash, err = node.ValueHash()
		assert.NoError(t, err)
		hashFromVp, err := vp[0].Hash()
		assert.NoError(t, err)
		assert.Equal(t, hashFromVp.Text(16), hash.Hex())
	})

	t.Run("Test NewParentNode", func(t *testing.T) {
		k := NewHashFromBytes(bytes.Repeat([]byte("0"), 32))
		node := NewParentNode(NodeTypeBranch_3, k, k)
		assert.Equal(t, NodeTypeBranch_3, node.Type)
		assert.Equal(t, k, node.ChildL)
		assert.Equal(t, k, node.ChildR)

		hash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, "242d3e8a6a7683f9858a08cdf1db2a4448638c168e32168ef4e5e9e2e8794629", hash.Hex())

		hash, err = node.ValueHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, hash)
	})

	t.Run("Test NewParentNodeWithEmptyChild", func(t *testing.T) {
		k := NewHashFromBytes(bytes.Repeat([]byte("0"), 32))
		r, err := NewEmptyNode().NodeHash()
		assert.NoError(t, err)
		node := NewParentNode(NodeTypeBranch_2, k, r)

		assert.Equal(t, NodeTypeBranch_2, node.Type)
		assert.Equal(t, k, node.ChildL)
		assert.Equal(t, r, node.ChildR)

		hash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, "005bc4e8f3b3f2ff0b980d4f3c32973de6a01f89ddacb08b0e7903d1f1f0c50f", hash.Hex())

		hash, err = node.ValueHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, hash)
	})

	t.Run("Test Invalid Node", func(t *testing.T) {
		node := &Node{Type: 99}

		invalidNodeHash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, invalidNodeHash)
	})
}

func TestNewNodeFromBytes(t *testing.T) {
	t.Run("ParentNode", func(t *testing.T) {
		k1 := NewHashFromBytes(bytes.Repeat([]byte("0"), 32))
		k2 := NewHashFromBytes(bytes.Repeat([]byte("0"), 32))
		node := NewParentNode(NodeTypeBranch_0, k1, k2)
		b := node.Value()

		node, err := NewNodeFromBytes(b)
		assert.NoError(t, err)

		assert.Equal(t, NodeTypeBranch_0, node.Type)
		assert.Equal(t, k1, node.ChildL)
		assert.Equal(t, k2, node.ChildR)

		hash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, "12b90fefb7b19131d25980a38ca92edb66bb91828d305836e4ab7e961165c83f", hash.Hex())

		hash, err = node.ValueHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, hash)
	})

	t.Run("LeafNode", func(t *testing.T) {
		k := NewHashFromBytes(bytes.Repeat([]byte("0"), 32))
		vp := make([]Byte32, 1)
		node := NewLeafNode(k, 1, vp)

		node.KeyPreimage = NewByte32FromBytes(bytes.Repeat([]byte("b"), 32))

		nodeBytes := node.Value()
		newNode, err := NewNodeFromBytes(nodeBytes)
		assert.NoError(t, err)

		assert.Equal(t, node.Type, newNode.Type)
		assert.Equal(t, node.NodeKey, newNode.NodeKey)
		assert.Equal(t, node.ValuePreimage, newNode.ValuePreimage)
		assert.Equal(t, node.KeyPreimage, newNode.KeyPreimage)

		hash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, "2f7094f04ed1592909311471ba67d84d7d11e2438c055f4d5d43189390c5cf5a", hash.Hex())

		hash, err = node.ValueHash()
		assert.NoError(t, err)
		hashFromVp, err := vp[0].Hash()

		assert.Equal(t, NewHashFromBigInt(hashFromVp), hash)
	})

	t.Run("EmptyNode", func(t *testing.T) {
		node := NewEmptyNode()
		b := node.Value()

		node, err := NewNodeFromBytes(b)
		assert.NoError(t, err)

		assert.Equal(t, NodeTypeEmpty_New, node.Type)

		hash, err := node.NodeHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, hash)

		hash, err = node.ValueHash()
		assert.NoError(t, err)
		assert.Equal(t, &HashZero, hash)
	})

	t.Run("BadSize", func(t *testing.T) {
		testCases := [][]byte{
			{},
			{0, 1, 2},
			func() []byte {
				b := make([]byte, HashByteLen+3)
				b[0] = byte(NodeTypeLeaf)
				return b
			}(),
			func() []byte {
				k := NewHashFromBytes([]byte{1, 2, 3, 4, 5})
				vp := make([]Byte32, 1)
				node := NewLeafNode(k, 1, vp)
				b := node.Value()
				return b[:len(b)-32]
			}(),
			func() []byte {
				k := NewHashFromBytes([]byte{1, 2, 3, 4, 5})
				vp := make([]Byte32, 1)
				node := NewLeafNode(k, 1, vp)
				node.KeyPreimage = NewByte32FromBytes([]byte{6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37})

				b := node.Value()
				return b[:len(b)-1]
			}(),
		}

		for _, b := range testCases {
			node, err := NewNodeFromBytes(b)
			assert.ErrorIs(t, err, ErrNodeBytesBadSize)
			assert.Nil(t, node)
		}
	})

	t.Run("InvalidType", func(t *testing.T) {
		b := []byte{255}

		node, err := NewNodeFromBytes(b)
		assert.ErrorIs(t, err, ErrInvalidNodeFound)
		assert.Nil(t, node)
	})
}

func TestNodeValueAndData(t *testing.T) {
	k := NewHashFromBytes(bytes.Repeat([]byte("a"), 32))
	vp := []Byte32{*NewByte32FromBytes(bytes.Repeat([]byte("b"), 32))}

	node := NewLeafNode(k, 1, vp)
	canonicalValue := node.CanonicalValue()
	assert.Equal(t, []byte{0x4, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x1, 0x1, 0x0, 0x0, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x0}, canonicalValue)
	assert.Equal(t, []byte{0x4, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x1, 0x1, 0x0, 0x0, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x0}, node.Value())
	node.KeyPreimage = NewByte32FromBytes(bytes.Repeat([]byte("c"), 32))
	assert.Equal(t, []byte{0x4, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x1, 0x1, 0x0, 0x0, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x20, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63, 0x63}, node.Value())
	assert.Equal(t, []byte{0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62, 0x62}, node.Data())

	parentNode := NewParentNode(NodeTypeBranch_3, k, k)
	canonicalValue = parentNode.CanonicalValue()
	assert.Equal(t, []byte{0x9, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}, canonicalValue)
	assert.Nil(t, parentNode.Data())

	emptyNode := &Node{Type: NodeTypeEmpty_New}
	assert.Equal(t, []byte{byte(emptyNode.Type)}, emptyNode.CanonicalValue())
	assert.Nil(t, emptyNode.Data())

	invalidNode := &Node{Type: 99}
	assert.Equal(t, []byte{}, invalidNode.CanonicalValue())
	assert.Nil(t, invalidNode.Data())
}

func TestNodeString(t *testing.T) {
	k := NewHashFromBytes(bytes.Repeat([]byte("a"), 32))
	vp := []Byte32{*NewByte32FromBytes(bytes.Repeat([]byte("b"), 32))}

	leafNode := NewLeafNode(k, 1, vp)
	assert.Equal(t, fmt.Sprintf("Leaf I:%v Items: %d, First:%v", leafNode.NodeKey, len(leafNode.ValuePreimage), leafNode.ValuePreimage[0]), leafNode.String())

	parentNode := NewParentNode(NodeTypeBranch_3, k, k)
	assert.Equal(t, fmt.Sprintf("Parent L:%s R:%s", parentNode.ChildL, parentNode.ChildR), parentNode.String())

	emptyNode := NewEmptyNode()
	assert.Equal(t, "Empty", emptyNode.String())

	invalidNode := &Node{Type: 99}
	assert.Equal(t, "Invalid Node", invalidNode.String())
}
