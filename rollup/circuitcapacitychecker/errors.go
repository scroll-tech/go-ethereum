package circuitcapacitychecker

import (
	"errors"
)

var (
	ErrUnknown                     = errors.New("unknown circuit capacity checker error")
	ErrTxRowConsumptionOverflow    = errors.New("tx row usage oveflow")
	ErrBlockRowConsumptionOverflow = errors.New("block row usage oveflow")
)
