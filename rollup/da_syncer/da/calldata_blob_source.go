package da

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

const (
	callDataBlobSourceFetchBlockRange  uint64 = 500
	commitBatchEventName                      = "CommitBatch"
	revertBatchEventName                      = "RevertBatch"
	finalizeBatchEventName                    = "FinalizeBatch"
	commitBatchMethodName                     = "commitBatch"
	commitBatchWithBlobProofMethodName        = "commitBatchWithBlobProof"

	// the length og method ID at the beginning of transaction data
	methodIDLength = 4
)

var (
	ErrSourceExhausted = errors.New("data source has been exhausted")
)

type CalldataBlobSource struct {
	ctx            context.Context
	l1Reader       *l1.Reader
	blobClient     blob_client.BlobClient
	l1height       uint64
	scrollChainABI *abi.ABI
	db             ethdb.Database

	l1Finalized uint64
}

func NewCalldataBlobSource(ctx context.Context, l1height uint64, l1Reader *l1.Reader, blobClient blob_client.BlobClient, db ethdb.Database) (*CalldataBlobSource, error) {
	scrollChainABI, err := l1.ScrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}
	return &CalldataBlobSource{
		ctx:            ctx,
		l1Reader:       l1Reader,
		blobClient:     blobClient,
		l1height:       l1height,
		scrollChainABI: scrollChainABI,
		db:             db,
	}, nil
}

func (ds *CalldataBlobSource) NextData() (Entries, error) {
	var err error
	to := ds.l1height + callDataBlobSourceFetchBlockRange

	// If there's not enough finalized blocks to request up to, we need to query finalized block number.
	// Otherwise, we know that there's more finalized blocks than we want to request up to
	// -> no need to query finalized block number
	if to > ds.l1Finalized {
		ds.l1Finalized, err = ds.l1Reader.GetLatestFinalizedBlockNumber()
		if err != nil {
			return nil, serrors.NewTemporaryError(fmt.Errorf("failed to query GetLatestFinalizedBlockNumber, error: %v", err))
		}
		// make sure we don't request more than finalized blocks
		to = min(to, ds.l1Finalized)
	}

	if ds.l1height > to {
		return nil, ErrSourceExhausted
	}

	rollupEvents, err := ds.l1Reader.FetchRollupEventsInRange(ds.l1height, to)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("cannot get rollup events, l1height: %d, error: %v", ds.l1height, err))
	}
	da, err := ds.processRollupEventsToDA(rollupEvents)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("failed to process rollup events to DA, error: %v", err))
	}

	ds.l1height = to + 1
	return da, nil
}

func (ds *CalldataBlobSource) L1Height() uint64 {
	return ds.l1height
}

func (ds *CalldataBlobSource) processRollupEventsToDA(rollupEvents l1.RollupEvents) (Entries, error) {
	var entries Entries
	var entry Entry
	var err error
	for _, rollupEvent := range rollupEvents {
		switch rollupEvent.Type() {
		case l1.CommitEventType:
			commitEvent, ok := rollupEvent.(*l1.CommitBatchEvent)
			// this should never happen because we just check event type
			if !ok {
				return nil, fmt.Errorf("unexpected type of rollup event: %T", rollupEvent)
			}
			if entry, err = ds.getCommitBatchDA(rollupEvent.BatchIndex().Uint64(), commitEvent); err != nil {
				return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", rollupEvent.BatchIndex().Uint64(), err)
			}

		case l1.RevertEventType:
			entry = NewRevertBatch(rollupEvent.BatchIndex().Uint64())

		case l1.FinalizeEventType:
			entry = NewFinalizeBatch(rollupEvent.BatchIndex().Uint64())

		default:
			return nil, fmt.Errorf("unknown rollup event, type: %v", rollupEvent.Type())
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

type commitBatchArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
}

func newCommitBatchArgs(method *abi.Method, values []interface{}) (*commitBatchArgs, error) {
	var args commitBatchArgs
	err := method.Inputs.Copy(&args, values)
	return &args, err
}

func newCommitBatchArgsFromCommitBatchWithProof(method *abi.Method, values []interface{}) (*commitBatchArgs, error) {
	var args commitBatchWithBlobProofArgs
	err := method.Inputs.Copy(&args, values)
	if err != nil {
		return nil, err
	}
	return &commitBatchArgs{
		Version:                args.Version,
		ParentBatchHeader:      args.ParentBatchHeader,
		Chunks:                 args.Chunks,
		SkippedL1MessageBitmap: args.SkippedL1MessageBitmap,
	}, nil
}

type commitBatchWithBlobProofArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
	BlobDataProof          []byte
}

func (ds *CalldataBlobSource) getCommitBatchDA(batchIndex uint64, commitEvent *l1.CommitBatchEvent) (Entry, error) {
	if batchIndex == 0 {
		return NewCommitBatchDAV0Empty(), nil
	}

	txData, err := ds.l1Reader.FetchTxData(commitEvent.TxHash(), commitEvent.BlockHash())
	if err != nil {
		return nil, err
	}
	if len(txData) < methodIDLength {
		return nil, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}

	method, err := ds.scrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return nil, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}
	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	if method.Name == commitBatchMethodName {
		args, err := newCommitBatchArgs(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}
		switch args.Version {
		case 0:
			return NewCommitBatchDAV0(ds.db, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap, commitEvent.BlockNumber())
		case 1:
			return NewCommitBatchDAV1(ds.ctx, ds.db, ds.l1Reader, ds.blobClient, commitEvent, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
		case 2:
			return NewCommitBatchDAV2(ds.ctx, ds.db, ds.l1Reader, ds.blobClient, commitEvent, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
		default:
			return nil, fmt.Errorf("failed to decode DA, codec version is unknown: codec version: %d", args.Version)
		}
	} else if method.Name == commitBatchWithBlobProofMethodName {
		args, err := newCommitBatchArgsFromCommitBatchWithProof(method, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}
		switch args.Version {
		case 3:
			// we can use V2 for version 3, because it's same
			return NewCommitBatchDAV2(ds.ctx, ds.db, ds.l1Reader, ds.blobClient, commitEvent, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
		case 4:
			return NewCommitBatchDAV4(ds.ctx, ds.db, ds.l1Reader, ds.blobClient, commitEvent, args.Version, batchIndex, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap)
		default:
			return nil, fmt.Errorf("failed to decode DA, codec version is unknown: codec version: %d", args.Version)
		}
	}

	return nil, fmt.Errorf("unknown method name: %s", method.Name)
}
