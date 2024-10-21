package da_syncer

import (
	"context"
	"errors"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

type DataSource interface {
	NextData() (da.Entries, error)
	L1Height() uint64
}

type DataSourceFactory struct {
	config        Config
	genesisConfig *params.ChainConfig
	l1Reader      *l1.Reader
	blobClient    blob_client.BlobClient
	msgStorage    *l1.MsgStorage
}

func NewDataSourceFactory(blockchain *core.BlockChain, genesisConfig *params.ChainConfig, config Config, l1Reader *l1.Reader, msgStorage *l1.MsgStorage, blobClient blob_client.BlobClient) *DataSourceFactory {
	return &DataSourceFactory{
		config:        config,
		genesisConfig: genesisConfig,
		l1Reader:      l1Reader,
		blobClient:    blobClient,
	}
}

func (ds *DataSourceFactory) OpenDataSource(ctx context.Context, l1height uint64) (DataSource, error) {
	if ds.config.FetcherMode == L1RPC {
		return da.NewCalldataBlobSource(ctx, l1height, ds.l1Reader, ds.blobClient, ds.msgStorage)
	} else {
		return nil, errors.New("snapshot_data_source: not implemented")
	}
}
