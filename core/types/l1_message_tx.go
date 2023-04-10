package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

const L1MessageTxType = 0x7E

// payload, RLP encoded
type L1MessageTx struct {
	Nonce  uint64
	Gas    uint64          // gas limit
	To     *common.Address `rlp:"nil"` // nil means contract creation
	Value  *big.Int
	Data   []byte
	Sender common.Address
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *L1MessageTx) copy() TxData {
	cpy := &L1MessageTx{
		Nonce:  tx.Nonce,
		Gas:    tx.Gas,
		To:     copyAddressPtr(tx.To),
		Value:  new(big.Int),
		Data:   common.CopyBytes(tx.Data),
		Sender: tx.Sender,
	}
	return cpy
}

// accessors for innerTx.
func (tx *L1MessageTx) txType() byte           { return L1MessageTxType }
func (tx *L1MessageTx) chainID() *big.Int      { return common.Big0 } // todo: update
func (tx *L1MessageTx) accessList() AccessList { return nil }
func (tx *L1MessageTx) data() []byte           { return tx.Data }
func (tx *L1MessageTx) gas() uint64            { return tx.Gas }
func (tx *L1MessageTx) gasFeeCap() *big.Int    { return new(big.Int) }
func (tx *L1MessageTx) gasTipCap() *big.Int    { return new(big.Int) }
func (tx *L1MessageTx) gasPrice() *big.Int     { return new(big.Int) }
func (tx *L1MessageTx) value() *big.Int        { return tx.Value }
func (tx *L1MessageTx) nonce() uint64          { return tx.Nonce }
func (tx *L1MessageTx) to() *common.Address    { return tx.To }

func (tx *L1MessageTx) rawSignatureValues() (v, r, s *big.Int) {
	return common.Big0, common.Big0, common.Big0
}

func (tx *L1MessageTx) setSignatureValues(chainID, v, r, s *big.Int) {
	// this is a noop for l1 message transactions
}
