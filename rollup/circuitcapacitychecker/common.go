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
	IsOk      bool   `json:"is_ok"`
	RowNumber uint64 `json:"row_number"`
	// pub row_usage_details: Vec<(String, usize)>,
}

type WrappedRowUsage struct {
	AccRowUsage RowUsage   `json:"acc_row_usage"`
	TxRowUsages []RowUsage `json:"tx_row_usages"`
	Err         string     `json:"error"`
}
