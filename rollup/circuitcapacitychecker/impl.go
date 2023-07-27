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

// mutex for CircuitCapacityChecker concurrent creation
var creationMu sync.Mutex

func init() {
	C.init()
}

type CircuitCapacityChecker struct {
	sync.Mutex
	id uint64
}

func NewCircuitCapacityChecker() *CircuitCapacityChecker {
	creationMu.Lock()
	defer creationMu.Unlock()

	id := C.new_circuit_capacity_checker()
	return &CircuitCapacityChecker{id: uint64(id)}
}

func (ccc *CircuitCapacityChecker) Reset() {
	ccc.Lock()
	defer ccc.Unlock()

	C.reset_circuit_capacity_checker(C.uint64_t(ccc.id))
}

func (ccc *CircuitCapacityChecker) ApplyTransaction(traces *types.BlockTrace) (*types.RowConsumption, error) {
	ccc.Lock()
	defer ccc.Unlock()

	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return nil, ErrUnknown
	}

	tracesStr := C.CString(string(tracesByt))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Info("start to check circuit capacity for tx")
	rawResult := C.apply_tx(C.uint64_t(ccc.id), tracesStr)
	log.Info("check circuit capacity for tx done")

	result := &WrappedRowUsage{}
	if err = json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		return nil, err
	}

	if result.Error != "" {
		log.Error("apply_tx in CircuitCapacityChecker", "err", result.Error)
		return nil, ErrUnknown
	}
	if result.TxRowUsage == nil || result.AccRowUsage == nil {
		log.Error("apply_tx in CircuitCapacityChecker", "err", "TxRowUsage or AccRowUsage is empty unexpectedly")
		return nil, ErrUnknown
	}
	if !result.TxRowUsage.IsOk {
		return nil, ErrTxRowConsumptionOverflow
	}
	if !result.AccRowUsage.IsOk {
		return nil, ErrBlockRowConsumptionOverflow
	}
	return (*types.RowConsumption)(&result.AccRowUsage.RowUsageDetails), nil
}

func (ccc *CircuitCapacityChecker) ApplyBlock(traces *types.BlockTrace) (*types.RowConsumption, error) {
	ccc.Lock()
	defer ccc.Unlock()

	tracesByt, err := json.Marshal(traces)
	if err != nil {
		return nil, ErrUnknown
	}

	tracesStr := C.CString(string(tracesByt))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Info("start to check circuit capacity for block")
	rawResult := C.apply_block(C.uint64_t(ccc.id), tracesStr)
	log.Info("check circuit capacity for block done")

	result := &WrappedRowUsage{}
	if err = json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		return nil, err
	}

	if result.Error != "" {
		log.Error("apply_tx in CircuitCapacityChecker", "err", result.Error)
		return nil, ErrUnknown
	}
	if result.AccRowUsage == nil {
		log.Error("apply_block in CircuitCapacityChecker", "err", "AccRowUsage is empty unexpectedly")
		return nil, ErrUnknown
	}
	if !result.AccRowUsage.IsOk {
		return nil, ErrBlockRowConsumptionOverflow
	}
	return (*types.RowConsumption)(&result.AccRowUsage.RowUsageDetails), nil
}
