package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/params"
)

type TransactionData struct {
	IsCreate bool           `json:"isCreate"`
	From     common.Address `json:"from"`
	*Transaction
	V *hexutil.Big `json:"v"`
	R *hexutil.Big `json:"r"`
	S *hexutil.Big `json:"s"`
}

// NewTraceTransaction returns a transaction that will serialize to the trace
// representation, with the given location metadata set (if available).
func NewTraceTransaction(tx *Transaction, blockNumber uint64, config *params.ChainConfig) *TransactionData {
	signer := MakeSigner(config, big.NewInt(0).SetUint64(blockNumber))
	from, _ := Sender(signer, tx)
	v, r, s := tx.RawSignatureValues()
	return &TransactionData{
		From:        from,
		IsCreate:    tx.To() == nil,
		Transaction: tx,
		V:           (*hexutil.Big)(v),
		R:           (*hexutil.Big)(r),
		S:           (*hexutil.Big)(s),
	}
}
