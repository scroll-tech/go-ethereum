package types

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

// L1BlockHashesTx
type L1BlockHashesTx struct {
	FirstAppliedL1Block uint64
	LastAppliedL1Block  uint64
	BlockHashesRange    []common.Hash
	To                  *common.Address
	Data                []byte
	Sender              common.Address
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *L1BlockHashesTx) copy() TxData {
	cpyBlockHashesRange := make([]common.Hash, len(tx.BlockHashesRange))
	copy(cpyBlockHashesRange, tx.BlockHashesRange)

	cpy := &L1BlockHashesTx{
		FirstAppliedL1Block: tx.FirstAppliedL1Block,
		LastAppliedL1Block:  tx.LastAppliedL1Block,
		BlockHashesRange:    cpyBlockHashesRange,
		To:                  copyAddressPtr(tx.To),
		Data:                common.CopyBytes(tx.Data),
		Sender:              tx.Sender,
	}
	return cpy
}

// accessors for innerTx.
func (tx *L1BlockHashesTx) txType() byte           { return L1BlockHashesTxType }
func (tx *L1BlockHashesTx) chainID() *big.Int      { return common.Big0 }
func (tx *L1BlockHashesTx) accessList() AccessList { return nil }
func (tx *L1BlockHashesTx) data() []byte           { return tx.Data }
func (tx *L1BlockHashesTx) gas() uint64            { return 1_000_000 } // TODO(l1blockhashes): Benchmark it based on worst-case scenario with maxL1BlockHashesPerTx
func (tx *L1BlockHashesTx) gasFeeCap() *big.Int    { return new(big.Int) }
func (tx *L1BlockHashesTx) gasTipCap() *big.Int    { return new(big.Int) }
func (tx *L1BlockHashesTx) gasPrice() *big.Int     { return new(big.Int) }
func (tx *L1BlockHashesTx) value() *big.Int        { return new(big.Int) }
func (tx *L1BlockHashesTx) nonce() uint64          { return 0 }
func (tx *L1BlockHashesTx) to() *common.Address    { return tx.To }

func (tx *L1BlockHashesTx) rawSignatureValues() (v, r, s *big.Int) {
	return common.Big0, common.Big0, common.Big0
}

func (tx *L1BlockHashesTx) setSignatureValues(chainID, v, r, s *big.Int) {
	// this is a noop for l1 blockhashes transactions
}
