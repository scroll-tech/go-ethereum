//go:build circuit_capacity_checker

package circuitcapacitychecker

/*
#cgo LDFLAGS: -lm -ldl -lzkp -lzktrie
#include <stdlib.h>
#include "./libzkp/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"encoding/json"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

type CircuitCapacityChecker struct{}

func NewCircuitCapacityChecker() *CircuitCapacityChecker {
	C.new_circuit_capacity_checker()
	return &CircuitCapacityChecker{}
}

func (ccc *CircuitCapacityChecker) Reset() {
	C.reset_circuit_capacity_checker()
}

func (ccc *CircuitCapacityChecker) ApplyTransaction(traces *types.BlockTrace) error {
	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return ErrUnknown
	}

	tracesStr := C.CString(string(tracesByt))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Info("start to check circuit capacity")
	result := C.apply_tx(tracesStr)
	log.Info("check circuit capacity done")

	switch result {
	case 0:
		return nil
	case 1:
		return ErrBlockRowUsageOverflow
	case 2:
		return ErrTxRowUsageOverflow
	default:
		return ErrUnknown
	}
}
