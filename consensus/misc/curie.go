package misc

import (
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
)

// ApplyCurieHardFork modifies the state database according to the Curie hard-fork
// rules, updating the bytecode of the L1GasPriceOracle contract.
func ApplyCurieHardFork(statedb *state.StateDB) {
	log.Info("Applying Curie hard fork")
	statedb.SetCode(rcfg.L1GasPriceOracleAddress, rcfg.CurieL1GasPriceOracleBytecode)
}
