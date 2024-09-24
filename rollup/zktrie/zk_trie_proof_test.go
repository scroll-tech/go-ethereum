package zktrie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeSMTProof(t *testing.T) {
	magicBytes := ProofMagicBytes()
	node, err := DecodeSMTProof(magicBytes)
	assert.NoError(t, err)
	assert.Nil(t, node)

	k1 := NewHashFromBytes([]byte{1, 2, 3, 4, 5})
	k2 := NewHashFromBytes([]byte{6, 7, 8, 9, 0})
	origNode := NewParentNode(NodeTypeBranch_0, k1, k2)
	node, err = DecodeSMTProof(origNode.Value())
	assert.NoError(t, err)
	assert.Equal(t, origNode.Value(), node.Value())
}
