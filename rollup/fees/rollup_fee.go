package fees

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

var (
	// txExtraDataBytes is the number of bytes that we commit to L1 in addition
	// to the RLP-encoded signed transaction. Note that these are all assumed
	// to be non-zero.
	// - tx length prefix: 4 bytes
	txExtraDataBytes = uint64(4)
)

// Message represents the interface of a message.
// It should be a subset of the methods found on
// types.Message
type Message interface {
	From() common.Address
	To() *common.Address
	GasPrice() *big.Int
	Gas() uint64
	GasFeeCap() *big.Int
	GasTipCap() *big.Int
	Value() *big.Int
	Nonce() uint64
	Data() []byte
	AccessList() types.AccessList
}

// StateDB represents the StateDB interface
// required to compute the L1 fee
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
	GetBalance(addr common.Address) *big.Int
}

func EstimateL1DataFeeForMessage(msg Message, baseFee, chainID *big.Int, signer types.Signer, state StateDB) (*big.Int, error) {
	unsigned := asUnsignedTx(msg, baseFee, chainID)
	tx, err := unsigned.WithSignature(signer, bytes.Repeat([]byte{0xff}, crypto.SignatureLength))
	if err != nil {
		return nil, err
	}

	raw, err := rlpEncode(tx)
	if err != nil {
		return nil, err
	}

	l1BaseFee, overhead, scalar := readGPOStorageSlots(rcfg.L1GasPriceOracleAddress, state)
	l1DataFee := CalculateL1Fee(raw, overhead, l1BaseFee, scalar)
	return l1DataFee, nil
}

// asUnsignedTx turns a Message into a types.Transaction
func asUnsignedTx(msg Message, baseFee, chainID *big.Int) *types.Transaction {
	if msg.AccessList() == nil {
		return asUnsignedLegacyTx(msg)
	}

	if baseFee == nil {
		return asUnsignedAccessListTx(msg, chainID)
	}

	return asUnsignedDynamicTx(msg, chainID)
}

func asUnsignedLegacyTx(msg Message) *types.Transaction {
	return types.NewTx(&types.LegacyTx{
		Nonce:    msg.Nonce(),
		To:       msg.To(),
		Value:    msg.Value(),
		Gas:      msg.Gas(),
		GasPrice: msg.GasPrice(),
		Data:     msg.Data(),
	})
}

func asUnsignedAccessListTx(msg Message, chainID *big.Int) *types.Transaction {
	return types.NewTx(&types.AccessListTx{
		Nonce:      msg.Nonce(),
		To:         msg.To(),
		Value:      msg.Value(),
		Gas:        msg.Gas(),
		GasPrice:   msg.GasPrice(),
		Data:       msg.Data(),
		AccessList: msg.AccessList(),
		ChainID:    chainID,
	})
}

func asUnsignedDynamicTx(msg Message, chainID *big.Int) *types.Transaction {
	return types.NewTx(&types.DynamicFeeTx{
		Nonce:      msg.Nonce(),
		To:         msg.To(),
		Value:      msg.Value(),
		Gas:        msg.Gas(),
		GasFeeCap:  msg.GasFeeCap(),
		GasTipCap:  msg.GasTipCap(),
		Data:       msg.Data(),
		AccessList: msg.AccessList(),
		ChainID:    chainID,
	})
}

// rlpEncode RLP encodes the transaction into bytes
func rlpEncode(tx *types.Transaction) ([]byte, error) {
	raw := new(bytes.Buffer)
	if err := tx.EncodeRLP(raw); err != nil {
		return nil, err
	}

	return raw.Bytes(), nil
}

func readGPOStorageSlots(addr common.Address, state StateDB) (*big.Int, *big.Int, *big.Int) {
	l1BaseFee := state.GetState(addr, rcfg.L1BaseFeeSlot)
	overhead := state.GetState(addr, rcfg.OverheadSlot)
	scalar := state.GetState(addr, rcfg.ScalarSlot)
	return l1BaseFee.Big(), overhead.Big(), scalar.Big()
}

// CalculateL1Fee computes the L1 fee
func CalculateL1Fee(data []byte, overhead, l1GasPrice *big.Int, scalar *big.Int) *big.Int {
	l1GasUsed := CalculateL1GasUsed(data, overhead)
	l1Fee := new(big.Int).Mul(l1GasUsed, l1GasPrice)
	return mulAndScale(l1Fee, scalar, rcfg.Precision)
}

// CalculateL1GasUsed computes the L1 gas used based on the calldata and
// constant sized overhead. The overhead can be decreased as the cost of the
// batch submission goes down via contract optimizations. This will not overflow
// under standard network conditions.
func CalculateL1GasUsed(data []byte, overhead *big.Int) *big.Int {
	zeroes, ones := zeroesAndOnes(data)
	zeroesGas := zeroes * params.TxDataZeroGas
	onesGas := (ones + txExtraDataBytes) * params.TxDataNonZeroGasEIP2028
	l1Gas := new(big.Int).SetUint64(zeroesGas + onesGas)
	return new(big.Int).Add(l1Gas, overhead)
}

// zeroesAndOnes counts the number of 0 bytes and non 0 bytes in a byte slice
func zeroesAndOnes(data []byte) (uint64, uint64) {
	var zeroes uint64
	var ones uint64
	for _, byt := range data {
		if byt == 0 {
			zeroes++
		} else {
			ones++
		}
	}
	return zeroes, ones
}

// mulAndScale multiplies a big.Int by a big.Int and then scale it by precision,
// rounded towards zero
func mulAndScale(x *big.Int, y *big.Int, precision *big.Int) *big.Int {
	z := new(big.Int).Mul(x, y)
	return new(big.Int).Quo(z, precision)
}

func CalculateFees(tx *types.Transaction, state StateDB) (*big.Int, *big.Int, *big.Int, error) {
	raw, err := rlpEncode(tx)
	if err != nil {
		return nil, nil, nil, err
	}

	l1BaseFee, overhead, scalar := readGPOStorageSlots(rcfg.L1GasPriceOracleAddress, state)
	l1Fee := CalculateL1Fee(raw, overhead, l1BaseFee, scalar)

	l2GasLimit := new(big.Int).SetUint64(tx.Gas())
	l2Fee := new(big.Int).Mul(tx.GasPrice(), l2GasLimit)
	fee := new(big.Int).Add(l1Fee, l2Fee)
	return l1Fee, l2Fee, fee, nil
}

func VerifyFee(signer types.Signer, tx *types.Transaction, state StateDB) error {
	from, err := types.Sender(signer, tx)
	if err != nil {
		return errors.New("invalid transaction: invalid sender")
	}

	balance := state.GetBalance(from)

	l1Fee, l2Fee, _, err := CalculateFees(tx, state)
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	cost := tx.Value()
	cost = cost.Add(cost, l2Fee)
	if balance.Cmp(cost) < 0 {
		return errors.New("invalid transaction: insufficient funds for gas * price + value")
	}

	cost = cost.Add(cost, l1Fee)
	if balance.Cmp(cost) < 0 {
		return errors.New("invalid transaction: insufficient funds for l1fee + gas * price + value")
	}

	// TODO: check GasPrice is in an expected range

	return nil
}
