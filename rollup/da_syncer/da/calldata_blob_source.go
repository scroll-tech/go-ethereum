package da

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

const (
	callDataBlobSourceFetchBlockRange uint64 = 500
)

var (
	ErrSourceExhausted = errors.New("data source has been exhausted")
)

type CalldataBlobSource struct {
	ctx            context.Context
	l1Reader       *l1.Reader
	blobClient     blob_client.BlobClient
	l1Height       uint64
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
		l1Height:       l1height,
		scrollChainABI: scrollChainABI,
		db:             db,
	}, nil
}

func (ds *CalldataBlobSource) NextData() (Entries, error) {
	var err error
	to := ds.l1Height + callDataBlobSourceFetchBlockRange

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

	if ds.l1Height > to {
		return nil, ErrSourceExhausted
	}

	rollupEvents, err := ds.l1Reader.FetchRollupEventsInRange(ds.l1Height, to)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("cannot get rollup events, l1Height: %d, error: %v", ds.l1Height, err))
	}
	da, err := ds.processRollupEventsToDA(rollupEvents)
	if err != nil {
		return nil, serrors.NewTemporaryError(fmt.Errorf("failed to process rollup events to DA, error: %v", err))
	}

	ds.l1Height = to + 1
	return da, nil
}

func (ds *CalldataBlobSource) SetL1Height(l1Height uint64) {
	ds.l1Height = l1Height
}

func (ds *CalldataBlobSource) L1Height() uint64 {
	return ds.l1Height
}

func (ds *CalldataBlobSource) L1Finalized() uint64 {
	return ds.l1Finalized
}

func (ds *CalldataBlobSource) processRollupEventsToDA(rollupEvents l1.RollupEvents) (Entries, error) {
	var entries Entries
	var entry Entry
	var err error

	var emptyHash common.Hash
	var lastCommitTransactionHash common.Hash
	var lastCommitEvents []*l1.CommitBatchEvent
	for _, rollupEvent := range rollupEvents {
		switch rollupEvent.Type() {
		case l1.CommitEventType:
			commitEvent, ok := rollupEvent.(*l1.CommitBatchEvent)
			// this should never happen because we just check event type
			if !ok {
				return nil, fmt.Errorf("unexpected type of rollup event: %T", rollupEvent)
			}

			// if this is a different commit transaction, we need to create a new DA
			if lastCommitTransactionHash != commitEvent.TxHash() {
				entry, err = ds.getCommitBatchDA(lastCommitEvents)
				if err != nil {
					return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", rollupEvent.BatchIndex().Uint64(), err)
				}
				entries = append(entries, entry)
				lastCommitEvents = nil
				lastCommitTransactionHash = emptyHash
			}

			// add commit event to the list of previous commit events, so we can process events created in the same tx together
			lastCommitTransactionHash = commitEvent.TxHash()
			lastCommitEvents = append(lastCommitEvents, commitEvent)
		case l1.RevertEventType:
			// if we have any previous commit events, we need to create a new DA before processing the revert event
			if len(lastCommitEvents) > 0 {
				entry, err = ds.getCommitBatchDA(lastCommitEvents)
				if err != nil {
					return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", rollupEvent.BatchIndex().Uint64(), err)
				}
				entries = append(entries, entry)
				lastCommitEvents = nil
				lastCommitTransactionHash = emptyHash
			}

			revertEvent, ok := rollupEvent.(*l1.RevertBatchEvent)
			// this should never happen because we just check event type
			if !ok {
				return nil, fmt.Errorf("unexpected type of rollup event: %T", rollupEvent)
			}

			entry = NewRevertBatch(revertEvent)
			entries = append(entries, entry)
		case l1.FinalizeEventType:
			// if we have any previous commit events, we need to create a new DA before processing the finalized event
			if len(lastCommitEvents) > 0 {
				entry, err = ds.getCommitBatchDA(lastCommitEvents)
				if err != nil {
					return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", rollupEvent.BatchIndex().Uint64(), err)
				}
				entries = append(entries, entry)
				lastCommitEvents = nil
				lastCommitTransactionHash = emptyHash
			}

			finalizeEvent, ok := rollupEvent.(*l1.FinalizeBatchEvent)
			// this should never happen because we just check event type
			if !ok {
				return nil, fmt.Errorf("unexpected type of rollup event: %T", rollupEvent)
			}

			entry = NewFinalizeBatch(finalizeEvent)
			entries = append(entries, entry)
		default:
			return nil, fmt.Errorf("unknown rollup event, type: %v", rollupEvent.Type())
		}
	}

	// if we have any previous commit events, we need to process them before returning
	if len(lastCommitEvents) > 0 {
		entry, err = ds.getCommitBatchDA(lastCommitEvents)
		if err != nil {
			return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", lastCommitEvents[0].BatchIndex().Uint64(), err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (ds *CalldataBlobSource) getCommitBatchDA(commitEvents []*l1.CommitBatchEvent) (Entries, error) {
	if len(commitEvents) == 0 {
		return nil, fmt.Errorf("commit events are empty")
	}

	if commitEvents[0].BatchIndex().Uint64() == 0 {
		return Entries{NewCommitBatchDAV0Empty()}, nil
	}

	firstCommitEvent := commitEvents[0]
	args, err := ds.l1Reader.FetchCommitTxData(firstCommitEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch commit tx data of batch %d, tx hash: %v, err: %w", firstCommitEvent.BatchIndex().Uint64(), firstCommitEvent.TxHash().Hex(), err)
	}

	codec, err := encoding.CodecFromVersion(encoding.CodecVersion(args.Version))
	if err != nil {
		return nil, fmt.Errorf("unsupported codec version: %v, batch index: %v, err: %w", args.Version, firstCommitEvent.BatchIndex().Uint64(), err)
	}

	var entries Entries
	var entry Entry
	var previousEvent *l1.CommitBatchEvent
	for _, commitEvent := range commitEvents {
		// sanity check events
		if commitEvent.TxHash() != firstCommitEvent.TxHash() {
			return nil, fmt.Errorf("commit events have different tx hashes, batch index: %d, tx: %s - batch index: %d, tx: %s", firstCommitEvent.BatchIndex().Uint64(), firstCommitEvent.TxHash().Hex(), commitEvent.BatchIndex().Uint64(), commitEvent.TxHash().Hex())
		}
		if previousEvent != nil && commitEvent.BatchIndex().Uint64() != previousEvent.BatchIndex().Uint64()+1 {
			return nil, fmt.Errorf("commit events are not in sequence, batch index: %d, hash: %s - previous batch index: %d, hash: %s", commitEvent.BatchIndex().Uint64(), commitEvent.BatchHash().Hex(), previousEvent.BatchIndex().Uint64(), previousEvent.BatchHash().Hex())
		}
		previousEvent = commitEvent

		switch codec.Version() {
		case 0:
			if entry, err = NewCommitBatchDAV0(ds.db, codec, commitEvent, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap); err != nil {
				return nil, fmt.Errorf("failed to decode DA, batch index: %d, err: %w", commitEvent.BatchIndex().Uint64(), err)
			}
		case 1, 2, 3, 4:
			if entry, err = NewCommitBatchDAWithBlob(ds.ctx, ds.db, ds.l1Reader, ds.blobClient, codec, commitEvent, args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap, args.BlobHashes); err != nil {
				return nil, fmt.Errorf("failed to decode DA, batch index: %d, err: %w", commitEvent.BatchIndex().Uint64(), err)
			}
		case 6:
			// TODO: implement codec version 6
			//  - there shouldn't be any need for args.ParentBatchHeader, args.Chunks, args.SkippedL1MessageBitmap
			//  - get blob hash from args for this commit event
			//  - sanity check somehow that this is the correct blob hash -> compute batch hash?
			return nil, nil
		default:
			return nil, fmt.Errorf("failed to decode DA, codec version is unknown: codec version: %d", args.Version)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
