package da_syncer

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/da"
	"github.com/scroll-tech/go-ethereum/rollup/missing_header_fields"
	"github.com/scroll-tech/go-ethereum/rollup/rollup_sync_service"
)

type DataSource interface {
	NextData() (da.Entries, error)
	L1Height() uint64
}

type DataSourceFactory struct {
	ctx                        context.Context
	genesisConfig              *params.ChainConfig
	config                     Config
	l1Client                   *rollup_sync_service.L1Client
	blobClient                 blob_client.BlobClient
	db                         ethdb.Database
	missingHeaderFieldsManager *missing_header_fields.Manager
}

func NewDataSourceFactory(ctx context.Context, genesisConfig *params.ChainConfig, config Config, l1Client *rollup_sync_service.L1Client, blobClient blob_client.BlobClient, db ethdb.Database) *DataSourceFactory {
	missingHeaderFieldsManager := missing_header_fields.NewManager(ctx,
		filepath.Join(config.AdditionalDataDir, missing_header_fields.DefaultFileName),
		genesisConfig.Scroll.DAConfig.MissingHeaderFieldsURL,
		genesisConfig.Scroll.DAConfig.MissingHeaderFieldsSHA256,
	)

	return &DataSourceFactory{
		genesisConfig:              genesisConfig,
		config:                     config,
		l1Client:                   l1Client,
		blobClient:                 blobClient,
		db:                         db,
		missingHeaderFieldsManager: missingHeaderFieldsManager,
	}
}

func (ds *DataSourceFactory) OpenDataSource(ctx context.Context, l1height uint64) (DataSource, error) {
	if ds.config.FetcherMode == L1RPC {
		return da.NewCalldataBlobSource(ctx, l1height, ds.l1Client, ds.blobClient, ds.db, ds.missingHeaderFieldsManager)
	} else {
		return nil, errors.New("snapshot_data_source: not implemented")
	}
}
