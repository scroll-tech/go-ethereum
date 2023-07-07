package circuitcapacitychecker

import (
	"errors"
)

var (
	ErrUnknown               = errors.New("unknown circuit capacity checker error")
	ErrTxRowUsageOverflow    = errors.New("tx row usage oveflow")
	ErrBlockRowUsageOverflow = errors.New("block row usage oveflow")
)
