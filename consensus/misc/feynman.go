package misc

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

// applyFeynmanHardFork modifies the state database according to the Feynman hard-fork rules,
// updating the bytecode and storage of the L1GasPriceOracle contract.
func applyFeynmanHardFork(statedb *state.StateDB) {
	log.Info("Applying Feynman hard fork")

	// update contract byte code
	statedb.SetCode(rcfg.L1GasPriceOracleAddress, rcfg.FeynmanL1GasPriceOracleBytecode)

	// initialize new storage slots
	statedb.SetState(rcfg.L1GasPriceOracleAddress, rcfg.IsFeynmanSlot, common.BytesToHash([]byte{1}))
	statedb.SetState(rcfg.L1GasPriceOracleAddress, rcfg.PenaltyThresholdSlot, common.BigToHash(rcfg.InitialPenaltyThreshold))
	statedb.SetState(rcfg.L1GasPriceOracleAddress, rcfg.PenaltyFactorSlot, common.BigToHash(rcfg.InitialPenaltyFactor))
}
