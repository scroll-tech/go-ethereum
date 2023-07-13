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
	"sync"
	"unsafe"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
)

func init() {
	C.init()
}

type CircuitCapacityChecker struct {
	sync.Mutex
	id uint64
}

func NewCircuitCapacityChecker() *CircuitCapacityChecker {
	id := C.new_circuit_capacity_checker()
	return &CircuitCapacityChecker{id: uint64(id)}
}

func (ccc *CircuitCapacityChecker) Reset() {
	ccc.Lock()
	defer ccc.Unlock()

	C.reset_circuit_capacity_checker(C.uint64_t(ccc.id))
}

func (ccc *CircuitCapacityChecker) ApplyTransaction(traces *types.BlockTrace) (uint64, error) {
	ccc.Lock()
	defer ccc.Unlock()

	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return 0, ErrUnknown
	}

	tracesStr := C.CString(string(tracesByt))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Info("start to check circuit capacity for tx")
	result := C.apply_tx(C.uint64_t(ccc.id), tracesStr)
	log.Info("check circuit capacity for tx done")

	switch result {
	case 0:
		return 0, ErrUnknown
	case -1:
		return ErrBlockRowConsumptionOverflow
	case -2:
		return ErrTxRowConsumptionOverflow
	default:
		return result, nil
	}
}

func (ccc *CircuitCapacityChecker) ApplyBlock(traces *types.BlockTrace) (uint64, error) {
	ccc.Lock()
	defer ccc.Unlock()

	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return 0, ErrUnknown
	}

	tracesStr := C.CString(string(tracesByt))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Info("start to check circuit capacity for block")
	result := C.apply_block(C.uint64_t(ccc.id), tracesStr)
	log.Info("check circuit capacity for block done")

	if result == 0 {
		return 0, ErrUnknown
	}
	if result < 0 {
		return 0, ErrBlockRowConsumptionOverflow
	}
	return uint64(result), nil
}
