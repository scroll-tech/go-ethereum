// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package misc

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
)

// Protocol-enforced maximum L2 base fee.
// We would only go above this if L1 base fee hits 2931 Gwei.
const MaximumL2BaseFee = 10000000000

// L2 base fee formula constants and defaults.
// l2BaseFee = (l1BaseFee * scalar) / PRECISION + overhead.
// `scalar` accounts for finalization costs. `overhead` accounts for sequencing and proving costs.
// we use 1e18 for precision to match the contract implementation.
var (
	BaseFeePrecision = new(big.Int).SetUint64(1e18)

	// scalar and overhead are updated automatically in `Blockchain.writeBlockWithState`.
	baseFeeScalar   = big.NewInt(0)
	baseFeeOverhead = big.NewInt(0)

	lock sync.RWMutex
)

func UpdateL2BaseFeeOverhead(newOverhead *big.Int) {
	if newOverhead == nil {
		log.Error("Failed to set L2 base fee overhead, new value is <nil>")
		return
	}
	lock.Lock()
	defer lock.Unlock()
	baseFeeOverhead.Set(newOverhead)
}

func UpdateL2BaseFeeScalar(newScalar *big.Int) {
	if newScalar == nil {
		log.Error("Failed to set L2 base fee scalar, new value is <nil>")
		return
	}
	lock.Lock()
	defer lock.Unlock()
	baseFeeScalar.Set(newScalar)
}

// VerifyEip1559Header verifies some header attributes which were changed in EIP-1559,
// - gas limit check
// - basefee check
func VerifyEip1559Header(config *params.ChainConfig, parent, header *types.Header) error {
	// Verify that the gas limit remains within allowed bounds
	if err := VerifyGaslimit(parent.GasLimit, header.GasLimit); err != nil {
		return err
	}
	// Verify the header is not malformed
	if header.BaseFee == nil {
		return fmt.Errorf("header is missing baseFee")
	}
	// note: we do not verify L2 base fee, the sequencer has the
	// right to set any base fee below the maximum. L2 base fee
	// is not subject to L2 consensus or zk verification.
	if header.BaseFee.Cmp(big.NewInt(MaximumL2BaseFee)) > 0 {
		return fmt.Errorf("invalid baseFee: have %s, maximum %d", header.BaseFee, MaximumL2BaseFee)
	}
	return nil
}

// CalcBaseFee calculates the basefee of the header.
func CalcBaseFee(config *params.ChainConfig, parent *types.Header, parentL1BaseFee *big.Int) *big.Int {
	if config.Clique != nil && config.Clique.ShadowForkHeight != 0 && parent.Number.Uint64() >= config.Clique.ShadowForkHeight {
		return big.NewInt(10000000) // 0.01 Gwei
	}

	lock.RLock()
	defer lock.RUnlock()
	return calcBaseFee(baseFeeScalar, baseFeeOverhead, parentL1BaseFee)
}

// MinBaseFee calculates the minimum L2 base fee based on the current coefficients.
func MinBaseFee() *big.Int {
	lock.RLock()
	defer lock.RUnlock()
	return calcBaseFee(baseFeeScalar, baseFeeOverhead, big.NewInt(0))
}

func calcBaseFee(scalar, overhead, parentL1BaseFee *big.Int) *big.Int {
	baseFee := new(big.Int).Set(parentL1BaseFee)
	baseFee.Mul(baseFee, scalar)
	baseFee.Div(baseFee, BaseFeePrecision)
	baseFee.Add(baseFee, overhead)

	if baseFee.Cmp(big.NewInt(MaximumL2BaseFee)) > 0 {
		baseFee = big.NewInt(MaximumL2BaseFee)
	}

	return baseFee
}
