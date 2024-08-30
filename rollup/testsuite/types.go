package testsuite

import (
	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/common"
)

type Batch struct {
	BatchHeaderBytes  []byte
	Hash              common.Hash
	LastL2BlockNumber uint64
	*encoding.Batch
}

func (b *Batch) StateRoot() common.Hash {
	lastChunk := len(b.Chunks) - 1
	lastBlock := len(b.Chunks[lastChunk].Blocks) - 1
	return b.Chunks[lastChunk].Blocks[lastBlock].Header.Root
}
