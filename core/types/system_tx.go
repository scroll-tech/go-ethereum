package types

import (
	"bytes"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/rlp"
)

type SystemTx struct {
	ChainID *big.Int
	Gas     uint64          // gas limit
	To      *common.Address // system contract
	Nonce   uint64          // nonce
	Data    []byte          // calldata
	V, R, S *big.Int        // signature values
}

func (tx *SystemTx) txType() byte { return SystemTxType }

func (tx *SystemTx) copy() TxData {
	cpy := &SystemTx{
		ChainID: new(big.Int),
		Gas:     tx.Gas,
		Nonce:   tx.Nonce,
		To:      copyAddressPtr(tx.To),
		Data:    common.CopyBytes(tx.Data),
		V:       new(big.Int),
		R:       new(big.Int),
		S:       new(big.Int),
	}

	if tx.ChainID != nil {
		cpy.ChainID.Set(tx.ChainID)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}

	return cpy
}

func (tx *SystemTx) chainID() *big.Int      { return tx.ChainID }
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
	return tx.V, tx.R, tx.S
}

func (tx *SystemTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}

func (tx *SystemTx) encode(b *bytes.Buffer) error {
	return rlp.Encode(b, tx)
}

func (tx *SystemTx) decode(input []byte) error {
	return rlp.DecodeBytes(input, tx)
}

var _ TxData = (*SystemTx)(nil)
