package types

import (
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

//go:generate gencodec -type RowConsumptionEntry -field-override rowConsumptionEntryMarshaling -out gen_row_consumption_json.go
type RowConsumptionEntry struct {
	Key  string `json:"key" gencodec:"required"`
	Rows uint64 `json:"rows" gencodec:"required"`
}

type RowConsumption []RowConsumptionEntry

// field type overrides for gencodec
type rowConsumptionEntryMarshaling struct {
	Rows hexutil.Uint64
}


