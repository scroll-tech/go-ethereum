package types

import (
	"bytes"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/rlp"
)

type SystemTx struct {
	Sender common.Address  // pre-determined sender
	To     *common.Address // system contract
	Data   []byte          // calldata
}

// not accountend
const SystemTxGas = 1_000_000

func (tx *SystemTx) txType() byte { return SystemTxType }

func (tx *SystemTx) copy() TxData {
	return &SystemTx{
		Sender: tx.Sender,
		To:     copyAddressPtr(tx.To),
		Data:   common.CopyBytes(tx.Data),
	}
}

func (tx *SystemTx) chainID() *big.Int      { return new(big.Int) }
func (tx *SystemTx) accessList() AccessList { return nil }
func (tx *SystemTx) data() []byte           { return tx.Data }
func (tx *SystemTx) gas() uint64            { return SystemTxGas }
func (tx *SystemTx) gasPrice() *big.Int     { return new(big.Int) }
func (tx *SystemTx) gasTipCap() *big.Int    { return new(big.Int) }
func (tx *SystemTx) gasFeeCap() *big.Int    { return new(big.Int) }
func (tx *SystemTx) value() *big.Int        { return new(big.Int) }
func (tx *SystemTx) nonce() uint64          { return 0 }
func (tx *SystemTx) to() *common.Address    { return tx.To }

func (tx *SystemTx) rawSignatureValues() (v, r, s *big.Int) {
	return new(big.Int), new(big.Int), new(big.Int)
}

func (tx *SystemTx) setSignatureValues(chainID, v, r, s *big.Int) {}

func (tx *SystemTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *SystemTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

var _ TxData = (*SystemTx)(nil)

func NewOrderedSystemTxs(stxs []*SystemTx) *OrderedSystemTxs {
	txs := make([]*Transaction, 0, len(stxs))
	for _, stx := range stxs {
		txs = append(txs, NewTx(stx))
	}
	return &OrderedSystemTxs{txs: txs}
}

type OrderedSystemTxs struct {
	txs []*Transaction
}

func (o *OrderedSystemTxs) Peek() *Transaction {
	if len(o.txs) > 0 {
		return o.txs[0]
	}
	return nil
}

func (o *OrderedSystemTxs) Shift() {
	if len(o.txs) > 0 {
		o.txs = o.txs[1:]
	}
}

func (o *OrderedSystemTxs) Pop() {}

var _ OrderedTransactionSet = (*OrderedSystemTxs)(nil)
