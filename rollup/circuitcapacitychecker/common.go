package circuitcapacitychecker

import (
	"errors"
)

var (
	ErrUnknown                     = errors.New("unknown circuit capacity checker error")
	ErrTxRowConsumptionOverflow    = errors.New("tx row consumption oveflow")
	ErrBlockRowConsumptionOverflow = errors.New("block row consumption oveflow")
)


type RowUsage struct {
    // pub is_ok: bool,
    // pub row_number: usize,
    // pub row_usage_details: Vec<(String, usize)>,
}

type WrappedRowUsage struct {
	AccRowUsage RowUsage
	TxRowUsages []RowUsage
	Err error
}