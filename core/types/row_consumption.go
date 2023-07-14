package types

import (
	"fmt"
	"io"

	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/rlp"
)

//go:generate gencodec -type RowConsumption -field-override rowConsumptionMarshaling -out gen_row_consumption_json.go

type RowConsumption struct {
	Rows uint64 `json:"rowConsumption" gencodec:"required"`
}

// field type overrides for gencodec
type rowConsumptionMarshaling struct {
	Rows hexutil.Uint64
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
