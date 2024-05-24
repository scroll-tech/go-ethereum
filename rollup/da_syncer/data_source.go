package da_syncer

import (
	"context"
	"errors"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/params"
)

var (
	sourceExhaustedErr = errors.New("data source has been exhausted")
)

type DataSource interface {
	NextData() (DA, error)
	L1Height() uint64
}

type DataSourceFactory struct {
	config        Config
	genesisConfig *params.ChainConfig
	l1Client      *L1Client
	blobClient    BlobClient
	db            ethdb.Database
}

func NewDataSourceFactory(blockchain *core.BlockChain, genesisConfig *params.ChainConfig, config Config, l1Client *L1Client, blobClient BlobClient, db ethdb.Database) *DataSourceFactory {
	return &DataSourceFactory{
		config:        config,
		genesisConfig: genesisConfig,
		l1Client:      l1Client,
		blobClient:    blobClient,
		db:            db,
	}
}

func (ds *DataSourceFactory) OpenDataSource(ctx context.Context, l1height uint64) (DataSource, error) {
	if ds.config.FetcherMode == L1RPC {
		return NewCalldataBlobSource(ctx, l1height, ds.l1Client, ds.blobClient, ds.db)
	} else {
		return nil, errors.New("snapshot_data_source: not implemented")
	}
}

func isBernoulliByL1Height(l1height uint64) bool {
	return false
}
