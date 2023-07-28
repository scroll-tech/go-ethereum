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

// mutex for concurrent CircuitCapacityChecker creations
var creationMu sync.Mutex

func init() {
	C.init()
}

type CircuitCapacityChecker struct {
	// mutex for each CircuitCapacityChecker itself
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
		log.Error("json marshal traces fail in ApplyTransaction", "id", ccc.id)
		return nil, ErrUnknown
	}

	tracesStr := C.CString(string(tracesByt))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Debug("start to check circuit capacity for tx", "id", ccc.id)
	rawResult := C.apply_tx(C.uint64_t(ccc.id), tracesStr)
	log.Debug("check circuit capacity for tx done", "id", ccc.id)

	result := &WrappedRowUsage{}
	if err = json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		log.Error("json unmarshal apply_tx invocation result fail", "id", ccc.id)
		return nil, ErrUnknown
	}

	if result.Error != "" {
		log.Error("apply_tx in CircuitCapacityChecker", "err", result.Error, "id", ccc.id)
		return nil, ErrUnknown
	}
	if result.TxRowUsage == nil || result.AccRowUsage == nil {
		log.Error("apply_tx in CircuitCapacityChecker", "err", "TxRowUsage or AccRowUsage is empty unexpectedly", "id", ccc.id)
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
		log.Error("json marshal traces fail in ApplyBlock", "id", ccc.id)
		return nil, ErrUnknown
	}

	tracesStr := C.CString(string(tracesByt))
	defer func() {
		C.free(unsafe.Pointer(tracesStr))
	}()

	log.Debug("start to check circuit capacity for block", "id", ccc.id, "blockNumber", traces.Header.Number, "blockHash", traces.Header.Hash())
	rawResult := C.apply_block(C.uint64_t(ccc.id), tracesStr)
	log.Debug("check circuit capacity for block done", "id", ccc.id, "blockNumber", traces.Header.Number, "blockHash", traces.Header.Hash())

	result := &WrappedRowUsage{}
	if err = json.Unmarshal([]byte(C.GoString(rawResult)), result); err != nil {
		log.Error("json unmarshal apply_tx invocation result fail", "id", ccc.id)
		return nil, ErrUnknown
	}

	if result.Error != "" {
		log.Error("apply_tx in CircuitCapacityChecker", "err", result.Error, "id", ccc.id)
		return nil, ErrUnknown
	}
	if result.AccRowUsage == nil {
		log.Error("apply_block in CircuitCapacityChecker", "err", "AccRowUsage is empty unexpectedly", "id", ccc.id)
		return nil, ErrUnknown
	}
	if !result.AccRowUsage.IsOk {
		return nil, ErrBlockRowConsumptionOverflow
	}
	return (*types.RowConsumption)(&result.AccRowUsage.RowUsageDetails), nil
}
