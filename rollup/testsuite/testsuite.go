package testsuite

import (
	"fmt"
	"math/big"
	"testing"

	bindETH "github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/rollup/testsuite/contracts"
)

const defaultKeyAlias = "default"

type TestSuite struct {
	t *testing.T

	km *KeyManager
	l1 *L1
	l2 *L2
}

func NewTestSuite(test *testing.T) *TestSuite {
	//log.Root().SetHandler(log.LvlFilterHandler(log.LvlDebug, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	km := NewKeyManager()

	l1, err := NewL1(km)
	require.NoError(test, err)

	l2, err := NewL2(km, l1)
	require.NoError(test, err)

	t := &TestSuite{
		t:  test,
		km: l1.keyManager,
		l1: l1,
		l2: l2,
	}

	batch, err := l2.commitGenesisBatch()
	require.NoError(test, err)
	l1Block := l1.CommitBlock()
	fmt.Println("Genesis batch committed with hash", batch.Hash.String(), "L1 block", l1Block.Number().Uint64())

	t.RequireCommitBatchEvent(l1Block.NumberU64(), nil, batch)
	t.RequireFinalizeBatchEvent(l1Block.NumberU64(), nil, batch)

	return t
}

func (t *TestSuite) Close() {
	err := t.l1.backend.Close()
	require.NoError(t.t, err)

	err = t.l2.backend.Close()
	require.NoError(t.t, err)
}

func (t *TestSuite) RequireCommitBatchEvent(start uint64, end *uint64, expectedBatches ...*Batch) {
	opts := &bindETH.FilterOpts{
		Start: start,
		End:   end,
	}

	var expectedBatchesMap = make(map[common.Hash]*Batch)
	var batchIndices []*big.Int
	var batchHashes [][common.HashLength]byte
	for _, batch := range expectedBatches {
		batchIndices = append(batchIndices, big.NewInt(int64(batch.Index)))
		batchHashes = append(batchHashes, batch.Hash)
		expectedBatchesMap[batch.Hash] = batch
	}

	iter, err := t.l1.ScrollChain().FilterCommitBatch(opts, batchIndices, batchHashes)
	require.NoError(t.t, err)

	events := make([]*contracts.ScrollChainMockFinalizeCommitBatch, 0)
	for iter.Next() {
		events = append(events, iter.Event)
		require.Contains(t.t, expectedBatchesMap, (common.Hash)(iter.Event.BatchHash))
		require.EqualValues(t.t, expectedBatchesMap[iter.Event.BatchHash].Index, iter.Event.BatchIndex.Uint64())
	}
	require.Len(t.t, events, len(expectedBatches))
}

func (t *TestSuite) RequireFinalizeBatchEvent(start uint64, end *uint64, expectedBatches ...*Batch) {
	opts := &bindETH.FilterOpts{
		Start: start,
		End:   end,
	}

	var expectedBatchesMap = make(map[common.Hash]*Batch)
	var batchIndices []*big.Int
	var batchHashes [][common.HashLength]byte
	for _, batch := range expectedBatches {
		batchIndices = append(batchIndices, big.NewInt(int64(batch.Index)))
		batchHashes = append(batchHashes, batch.Hash)
		expectedBatchesMap[batch.Hash] = batch
	}

	iter, err := t.l1.ScrollChain().FilterFinalizeBatch(opts, batchIndices, batchHashes)
	require.NoError(t.t, err)

	events := make([]*contracts.ScrollChainMockFinalizeFinalizeBatch, 0)
	for iter.Next() {
		events = append(events, iter.Event)

		require.Contains(t.t, expectedBatchesMap, (common.Hash)(iter.Event.BatchHash))
		require.EqualValues(t.t, expectedBatchesMap[iter.Event.BatchHash].StateRoot(), iter.Event.StateRoot)
		// TODO: do we need to check for withdraw root?
	}
	require.Len(t.t, events, len(expectedBatches))
}
