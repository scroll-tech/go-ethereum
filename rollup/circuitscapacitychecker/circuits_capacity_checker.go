//go:build placeholder

package circuitscapacitychecker

/*
#cgo LDFLAGS: ${SRCDIR}/libzkp/libzkp.so -lm -ldl -lzktrie -L${SRCDIR}/libzkp/ -Wl,-rpath=${SRCDIR}/libzkp
#include <stdlib.h>
#include "./libzkp/libzkp.h"
*/
import "C" //nolint:typecheck

import (
	"github.com/scroll-tech/go-ethereum/core/types"
)

type CircuitsCapacityChecker struct{}

func NewCircuitsCapacityChecker() *CircuitsCapacityChecker {
	return &CircuitsCapacityChecker{}
}

// TODO:
func (ccc *CircuitsCapacityChecker) Reset() {
	// panic if call fails?
}

// TODO:
func (ccc *CircuitsCapacityChecker) ApplyTransaction(logs []*types.Log) error {
	return nil
}
