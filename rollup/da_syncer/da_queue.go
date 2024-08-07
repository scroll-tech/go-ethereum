package da_syncer

import (
	"context"
	"errors"

	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
)

// DAQueue is a pipeline stage that reads DA entries from a DataSource and provides them to the next stage.
type DAQueue struct {
	l1height          uint64
	dataSourceFactory *DataSourceFactory
	dataSource        DataSource
	da                da.Entries
}

func NewDAQueue(l1height uint64, dataSourceFactory *DataSourceFactory) *DAQueue {
	return &DAQueue{
		l1height:          l1height,
		dataSourceFactory: dataSourceFactory,
		dataSource:        nil,
		da:                make(da.Entries, 0),
	}
}

func (dq *DAQueue) NextDA(ctx context.Context) (da.Entry, error) {
	for len(dq.da) == 0 {
		err := dq.getNextData(ctx)
		if err != nil {
			return nil, err
		}
	}
	daEntry := dq.da[0]
	dq.da = dq.da[1:]
	return daEntry, nil
}

func (dq *DAQueue) getNextData(ctx context.Context) error {
	var err error
	if dq.dataSource == nil {
		dq.dataSource, err = dq.dataSourceFactory.OpenDataSource(ctx, dq.l1height)
		if err != nil {
			return err
		}
	}

	dq.da, err = dq.dataSource.NextData()
	if err == nil {
		return nil
	}

	// previous dataSource has been exhausted, create new
	if errors.Is(err, da.ErrSourceExhausted) {
		dq.l1height = dq.dataSource.L1Height()
		dq.dataSource = nil
		return dq.getNextData(ctx)
	}

	return err
}

func (dq *DAQueue) Reset(height uint64) {
	dq.l1height = height
	dq.dataSource = nil
	dq.da = make(da.Entries, 0)
}
