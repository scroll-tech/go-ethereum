package types

import (
	"fmt"
	"io"

	"github.com/scroll-tech/go-ethereum/rlp"
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

type RowConsumption struct {
	Rows uint64

	// tmp workaround. will only keep `Detail` and remove `Rows` later
	Detail []SubCircuitRowUsage
}

func (rc *RowConsumption) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, rc.Rows)
}

func (rc *RowConsumption) DecodeRLP(s *rlp.Stream) error {
	_, size, err := s.Kind()
	if err != nil {
		return err
	}
	if size <= 8 {
		return s.Decode(&rc.Rows)
	} else {
		return fmt.Errorf("invalid input size %d for origin", size)
	}
}
