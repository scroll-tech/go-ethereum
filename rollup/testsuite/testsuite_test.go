package testsuite

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSuiteTest(t *testing.T) {
	ts := NewTestSuite(t)

	_, err := ts.l1.SendL1ToL2Message("to1", []byte{1, 2, 3}, true)
	require.NoError(t, err)

	_, err = ts.l1.SendL1ToL2Message("to2", []byte{7, 8, 9, 10}, true)
	require.NoError(t, err)

	ts.l1.CommitBlock()

	latestHeader, err := ts.l1.client.HeaderByNumber(context.Background(), nil)
	require.NoError(t, err)
	fmt.Println("Latest header", latestHeader.Number)

	events, err := ts.l1.FilterL1MessageQueueTransactions(0, latestHeader.Number.Uint64())
	require.NoError(t, err)
	for _, event := range events {
		fmt.Println(event.QueueIndex, "L1 block", event.Raw.BlockNumber, "sender", event.Sender, "data", event.Data, "to", event.Target)
	}

	fmt.Println("----------------------- L2 ---------------------")

	for i := 0; i < 5; i++ {
		_, err = ts.l2.SendDynamicFeeTransaction("default", "to1", big.NewInt(1), nil, false)
		require.NoError(t, err)
	}
	ts.l2.CommitBlock()
	ts.l2.CommitBlock()
	ts.l2.CommitBlock()

	commitBatch, err := ts.l2.CommitBatch()
	require.NoError(t, err)
	l1Block := ts.l1.CommitBlock()

	for _, tx := range l1Block.Transactions() {
		//fmt.Println(tx.Hash().String(), tx.Data())
		receipt, err := ts.l1.client.TransactionReceipt(context.Background(), tx.Hash())
		require.NoError(t, err)
		fmt.Println("  ", receipt.Status, receipt.Logs)
	}

	ts.RequireCommitBatchEvent(l1Block.NumberU64(), nil, commitBatch)
}
