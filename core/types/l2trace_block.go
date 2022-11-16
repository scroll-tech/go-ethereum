package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/params"
)

type BlockTrace struct {
	Number       *hexutil.Big        `json:"number"`
	Header       *Header             `json:"header"`
	Hash         common.Hash         `json:"hash"`
	GasLimit     uint64              `json:"gasLimit"`
	Difficulty   *hexutil.Big        `json:"difficulty"`
	BaseFee      *hexutil.Big        `json:"baseFee"`
	Coinbase     *AccountWrapper     `json:"coinbase"`
	Time         uint64              `json:"time"`
	Transactions []*TransactionTrace `json:"transactions"`
}

type TransactionTrace struct {
	ChainId   *hexutil.Big   `json:"chainId"`
	IsCreate  bool           `json:"isCreate"`
	From      common.Address `json:"from"`
	TxContent *Transaction   `json:"txContent"`
}

// NewTraceBlock supports necessary fields for roller.
func NewTraceBlock(config *params.ChainConfig, block *Block, coinbase *AccountWrapper) *BlockTrace {
	txs := make([]*TransactionTrace, block.Transactions().Len())
	for i, tx := range block.Transactions() {
		txs[i] = newTraceTransaction(tx, block.NumberU64(), config)
	}

	return &BlockTrace{
		Number:       (*hexutil.Big)(block.Number()),
		Header:       block.Header(),
		Hash:         block.Hash(),
		GasLimit:     block.GasLimit(),
		Difficulty:   (*hexutil.Big)(block.Difficulty()),
		BaseFee:      (*hexutil.Big)(block.BaseFee()),
		Coinbase:     coinbase,
		Time:         block.Time(),
		Transactions: txs,
	}
}

// newTraceTransaction returns a transaction that will serialize to the trace
// representation, with the given location metadata set (if available).
func newTraceTransaction(tx *Transaction, blockNumber uint64, config *params.ChainConfig) *TransactionTrace {
	signer := MakeSigner(config, big.NewInt(0).SetUint64(blockNumber))
	from, _ := Sender(signer, tx)
	result := &TransactionTrace{
		ChainId:   (*hexutil.Big)(tx.ChainId()),
		From:      from,
		IsCreate:  tx.To() == nil,
		TxContent: tx,
	}
	return result
}
