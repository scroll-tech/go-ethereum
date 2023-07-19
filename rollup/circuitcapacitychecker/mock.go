//go:build !circuit_capacity_checker

package circuitcapacitychecker

import (
	"github.com/scroll-tech/go-ethereum/core/types"
)

type CircuitCapacityChecker struct{}

func NewCircuitCapacityChecker() *CircuitCapacityChecker {
	return &CircuitCapacityChecker{}
}

func (ccc *CircuitCapacityChecker) Reset() {
}

func (ccc *CircuitCapacityChecker) ApplyTransaction(traces *types.BlockTrace) ([]SubCircuitRowUsage, error) {
	return 0, nil
}

func (ccc *CircuitCapacityChecker) ApplyBlock(traces *types.BlockTrace) ([]SubCircuitRowUsage, error) {
	return 0, nil
}
