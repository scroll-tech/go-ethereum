package misc

import (
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

// applyGalileoV2HardFork modifies the state database according to the GalileoV2 hard-fork rules,
// updating the bytecode and storage of the L1GasPriceOracle contract.
func applyGalileoV2HardFork(statedb *state.StateDB) {
	log.Info("Applying GalileoV2 hard fork")

	// update contract byte code
	statedb.SetCode(rcfg.L1GasPriceOracleAddress, rcfg.GalileoV2L1GasPriceOracleBytecode)

	// initialize new storage slots
	statedb.SetState(rcfg.L1GasPriceOracleAddress, rcfg.IsGalileoSlot, common.BytesToHash([]byte{1}))
}
