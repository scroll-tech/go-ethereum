package fees

import (
	"bytes"
	// "context"
	"errors"
	// "math"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	// "github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

var (
	// errTransactionSigned represents the error case of passing in a signed
	// transaction to the L1 fee calculation routine. The signature is accounted
	// for externally
	errTransactionSigned = errors.New("transaction is signed")
)

// Message represents the interface of a message.
// It should be a subset of the methods found on
// types.Message
type Message interface {
	From() common.Address
	To() *common.Address
	GasPrice() *big.Int
	Gas() uint64
	Value() *big.Int
	Nonce() uint64
	Data() []byte
}

// StateDB represents the StateDB interface
// required to compute the L1 fee
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
}

// CalculateL1MsgFee computes the L1 portion of the fee given
// a Message and a StateDB
func CalculateL1MsgFee(msg Message, state StateDB) (*big.Int, error) {
	tx := asTransaction(msg)
	raw, err := rlpEncode(tx)
	if err != nil {
		return nil, err
	}

	gpo := rcfg.L2GasPriceOracleAddress
	overhead, scalar := readGPOStorageSlots(*gpo, state)
	l1Fee := CalculateL1Fee(
		raw,
		overhead,
		nil, // TODO: l1GasPrice,
		scalar)

	return l1Fee, nil
}

// asTransaction turns a Message into a types.Transaction
func asTransaction(msg Message) *types.Transaction {
	if msg.To() == nil {
		return types.NewContractCreation(
			msg.Nonce(),
			msg.Value(),
			msg.Gas(),
			msg.GasPrice(),
			msg.Data(),
		)
	}
	return types.NewTransaction(
		msg.Nonce(),
		*msg.To(),
		msg.Value(),
		msg.Gas(),
		msg.GasPrice(),
		msg.Data(),
	)
}

// rlpEncode RLP encodes the transaction into bytes
// When a signature is not included, set pad to true to
// fill in a dummy signature full on non 0 bytes
func rlpEncode(tx *types.Transaction) ([]byte, error) {
	raw := new(bytes.Buffer)
	if err := tx.EncodeRLP(raw); err != nil {
		return nil, err
	}

	r, v, s := tx.RawSignatureValues()
	if r.Cmp(common.Big0) != 0 || v.Cmp(common.Big0) != 0 || s.Cmp(common.Big0) != 0 {
		return nil, errTransactionSigned
	}

	// Slice off the 0 bytes representing the signature
	b := raw.Bytes()
	return b[:len(b)-3], nil
}

func readGPOStorageSlots(addr common.Address, state StateDB) (*big.Int, *big.Float) {
	overhead := state.GetState(addr, rcfg.OverheadSlot)
	scalar := state.GetState(addr, rcfg.ScalarSlot)
	scaled := ScalePrecision(scalar.Big(), rcfg.Precision)
	return overhead.Big(), scaled
}

// ScalePrecision will scale a value by precision
func ScalePrecision(scalar, precision *big.Int) *big.Float {
	fscalar := new(big.Float).SetInt(scalar)
	fdivisor := new(big.Float).SetInt(precision)
	// fscalar / fdivisor
	return new(big.Float).Quo(fscalar, fdivisor)
}

// TODO:
// CalculateL1Fee computes the L1 fee
func CalculateL1Fee(data []byte, overhead, l1GasPrice *big.Int, scalar *big.Float) *big.Int {
	// l1GasUsed := CalculateL1GasUsed(data, overhead)
	// l1Fee := new(big.Int).Mul(l1GasUsed, l1GasPrice)
	// return mulByFloat(l1Fee, scalar)
	return nil
}
