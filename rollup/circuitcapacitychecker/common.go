package circuitcapacitychecker

import (
	"errors"
)

var (
	ErrUnknown                     = errors.New("unknown circuit capacity checker error")
	ErrTxRowConsumptionOverflow    = errors.New("tx row consumption oveflow")
	ErrBlockRowConsumptionOverflow = errors.New("block row consumption oveflow")
)

type SubCircuitRowUsage struct {
	Name      string `json:"name"`
	RowNumber uint64 `json:"row_number"`
}

type RowUsage struct {
	IsOk            bool                 `json:"is_ok"`
	RowNumber       uint64               `json:"row_number"`
	RowUsageDetails []SubCircuitRowUsage `json:"row_usage_details"`
}

type WrappedRowUsage struct {
	AccRowUsage RowUsage `json:"acc_row_usage"`
	TxRowUsage  RowUsage `json:"tx_row_usage"`
	Error       string   `json:"error"`
}
