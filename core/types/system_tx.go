package types

import (
	"bytes"
	"math/big"
	"sort"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/rlp"
)

type SystemTx struct {
	From  common.Address  // pre-determined sender
	To    *common.Address // system contract
	Gas   uint64          // gas limit
	Nonce uint64          // nonce
	Data  []byte          // calldata
}

func (tx *SystemTx) txType() byte { return SystemTxType }

func (tx *SystemTx) copy() TxData {
	return &SystemTx{
		From:  tx.From,
		Gas:   tx.Gas,
		Nonce: tx.Nonce,
		To:    copyAddressPtr(tx.To),
		Data:  common.CopyBytes(tx.Data),
	}
}

func (tx *SystemTx) chainID() *big.Int      { return new(big.Int) }
func (tx *SystemTx) accessList() AccessList { return nil }
func (tx *SystemTx) data() []byte           { return tx.Data }
func (tx *SystemTx) gas() uint64            { return tx.Gas }
func (tx *SystemTx) gasPrice() *big.Int     { return new(big.Int) }
func (tx *SystemTx) gasTipCap() *big.Int    { return new(big.Int) }
func (tx *SystemTx) gasFeeCap() *big.Int    { return new(big.Int) }
func (tx *SystemTx) value() *big.Int        { return new(big.Int) }
func (tx *SystemTx) nonce() uint64          { return tx.Nonce }
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
	sort.SliceStable(txs, func(i, j int) bool {
		txi := txs[i].AsSystemTx()
		txj := txs[j].AsSystemTx()
		cmp := bytes.Compare(txi.From.Bytes(), txj.From.Bytes())
		if cmp < 0 {
			return true
		} else if cmp == 0 {
			return txi.Nonce < txj.Nonce
		} else {
			return false
		}
	})
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

func (o *OrderedSystemTxs) Pop() {
	if len(o.txs) > 0 {
		currentSender := o.txs[0].AsSystemTx().From
		i := 1
		for ; i < len(o.txs); i++ {
			if o.txs[i].AsSystemTx().From != currentSender {
				break
			}
		}
		o.txs = o.txs[i:]
	}
}

var _ OrderedTransactionSet = (*OrderedSystemTxs)(nil)
