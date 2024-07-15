package da_syncer

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv1"
	"github.com/scroll-tech/da-codec/encoding/codecv2"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
)

var (
	callDataBlobSourceFetchBlockRange uint64 = 500
)

type CalldataBlobSource struct {
	ctx                           context.Context
	l1Client                      *L1Client
	blobClient                    BlobClient
	l1height                      uint64
	scrollChainABI                *abi.ABI
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
	db                            ethdb.Database
}

func NewCalldataBlobSource(ctx context.Context, l1height uint64, l1Client *L1Client, blobClient BlobClient, db ethdb.Database) (DataSource, error) {
	scrollChainABI, err := scrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}
	return &CalldataBlobSource{
		ctx:                           ctx,
		l1Client:                      l1Client,
		blobClient:                    blobClient,
		l1height:                      l1height,
		scrollChainABI:                scrollChainABI,
		l1CommitBatchEventSignature:   scrollChainABI.Events["CommitBatch"].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events["RevertBatch"].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events["FinalizeBatch"].ID,
		db:                            db,
	}, nil
}

func (ds *CalldataBlobSource) NextData() (DA, error) {
	to := ds.l1height + callDataBlobSourceFetchBlockRange
	l1Finalized, err := ds.l1Client.getFinalizedBlockNumber(ds.ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot get l1height, error: %v", err)
	}
	if to > l1Finalized.Uint64() {
		to = l1Finalized.Uint64()
	}
	if ds.l1height > to {
		return nil, sourceExhaustedErr
	}
	logs, err := ds.l1Client.fetchRollupEventsInRange(ds.ctx, ds.l1height, to)
	if err != nil {
		return nil, fmt.Errorf("cannot get events, l1height: %d, error: %v", ds.l1height, err)
	}
	da, err := ds.processLogsToDA(logs)
	if err == nil {
		ds.l1height = to + 1
	}
	return da, err
}

func (ds *CalldataBlobSource) L1Height() uint64 {
	return ds.l1height
}

func (ds *CalldataBlobSource) processLogsToDA(logs []types.Log) (DA, error) {
	var da DA
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case ds.l1CommitBatchEventSignature:
			event := &L1CommitBatchEvent{}
			if err := UnpackLog(ds.scrollChainABI, event, "CommitBatch", vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack commit rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new CommitBatch event", "batch index", batchIndex)

			daEntry, err := ds.getCommitBatchDa(batchIndex, &vLog)
			if err != nil {
				return nil, fmt.Errorf("failed to get commit batch da: %v, err: %w", batchIndex, err)
			}
			da = append(da, daEntry)

		case ds.l1RevertBatchEventSignature:
			event := &L1RevertBatchEvent{}
			if err := UnpackLog(ds.scrollChainABI, event, "RevertBatch", vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack revert rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new RevertBatch event", "batch index", batchIndex)
			da = append(da, NewRevertBatchDA(batchIndex))

		case ds.l1FinalizeBatchEventSignature:
			event := &L1FinalizeBatchEvent{}
			if err := UnpackLog(ds.scrollChainABI, event, "FinalizeBatch", vLog); err != nil {
				return nil, fmt.Errorf("failed to unpack finalized rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new FinalizeBatch event", "batch index", batchIndex)

			da = append(da, NewFinalizeBatchDA(batchIndex))

		default:
			return nil, fmt.Errorf("unknown event, topic: %v, tx hash: %v", vLog.Topics[0].Hex(), vLog.TxHash.Hex())
		}
	}
	return da, nil
}

type commitBatchArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
}

type commitBatchWithBlobProofArgs struct {
	Version                uint8
	ParentBatchHeader      []byte
	Chunks                 [][]byte
	SkippedL1MessageBitmap []byte
	BlobDataProof          []byte
}

func (ds *CalldataBlobSource) getCommitBatchDa(batchIndex uint64, vLog *types.Log) (DAEntry, error) {
	if batchIndex == 0 {
		return NewCommitBatchDaV0(0, batchIndex, 0, []byte{}, []*codecv0.DAChunkRawTx{}, []*types.L1MessageTx{}, 0), nil
	}

	txData, err := ds.l1Client.fetchTxData(ds.ctx, vLog)
	if err != nil {
		return nil, err
	}
	const methodIDLength = 4
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

	if method.Name == "commitBatch" {
		var args commitBatchArgs
		err = method.Inputs.Copy(&args, values)
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}
		switch args.Version {
		case 0:
			return ds.decodeDAV0(batchIndex, vLog, &args)
		case 1:
			return ds.decodeDAV1(batchIndex, vLog, &args)
		case 2:
			return ds.decodeDAV2(batchIndex, vLog, &args)
		default:
			return nil, fmt.Errorf("failed to decode DA, codec version is unknown: codec version: %d", args.Version)
		}
	} else {
		var args commitBatchWithBlobProofArgs
		err = method.Inputs.Copy(&args, values)
		var usedArgs commitBatchArgs = commitBatchArgs{
			Version:                args.Version,
			ParentBatchHeader:      args.ParentBatchHeader,
			Chunks:                 args.Chunks,
			SkippedL1MessageBitmap: args.SkippedL1MessageBitmap,
		}
		if err != nil {
			return nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
		}
		return ds.decodeDAV2(batchIndex, vLog, &usedArgs)
	}

}

func (ds *CalldataBlobSource) decodeDAV0(batchIndex uint64, vLog *types.Log, args *commitBatchArgs) (DAEntry, error) {
	var chunks []*codecv0.DAChunkRawTx
	var l1Txs []*types.L1MessageTx
	chunks, err := codecv0.DecodeDAChunksRawTx(args.Chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack chunks: %v, err: %w", batchIndex, err)
	}

	parentTotalL1MessagePopped := getBatchTotalL1MessagePopped(args.ParentBatchHeader)
	totalL1MessagePopped := 0
	for _, chunk := range chunks {
		for _, block := range chunk.Blocks {
			totalL1MessagePopped += int(block.NumL1Messages)
		}
	}
	skippedBitmap, err := encoding.DecodeBitmap(args.SkippedL1MessageBitmap, totalL1MessagePopped)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bitmap: %v, err: %w", batchIndex, err)
	}
	// get all necessary l1msgs without skipped
	currentIndex := parentTotalL1MessagePopped
	for index := 0; index < totalL1MessagePopped; index++ {
		if encoding.IsL1MessageSkipped(skippedBitmap, currentIndex-parentTotalL1MessagePopped) {
			currentIndex++
			continue
		}
		l1Tx := rawdb.ReadL1Message(ds.db, currentIndex)
		if l1Tx == nil {
			return nil, fmt.Errorf("failed to read L1 message from db, l1 message index: %v", currentIndex)
		}
		l1Txs = append(l1Txs, l1Tx)
		currentIndex++
	}
	da := NewCommitBatchDaV0(args.Version, batchIndex, parentTotalL1MessagePopped, args.SkippedL1MessageBitmap, chunks, l1Txs, vLog.BlockNumber)
	return da, nil
}

func (ds *CalldataBlobSource) decodeDAV1(batchIndex uint64, vLog *types.Log, args *commitBatchArgs) (DAEntry, error) {
	var chunks []*codecv1.DAChunkRawTx
	var l1Txs []*types.L1MessageTx
	chunks, err := codecv1.DecodeDAChunksRawTx(args.Chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack chunks: %v, err: %w", batchIndex, err)
	}

	versionedHash, err := ds.l1Client.fetchTxBlobHash(ds.ctx, vLog)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob hash, err: %w", err)
	}
	blob, err := ds.blobClient.GetBlobByVersionedHash(ds.ctx, versionedHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob from blob client, err: %w", err)
	}
	// compute blob versioned hash and compare with one from tx
	c, err := kzg4844.BlobToCommitment(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob commitment")
	}
	blobVersionedHash := common.Hash(kzg4844.CalcBlobHashV1(sha256.New(), &c))
	if blobVersionedHash != versionedHash {
		return nil, fmt.Errorf("blobVersionedHash from blob source is not equal to versionedHash from tx, correct versioned hash: %s, fetched blob hash: %s", versionedHash.String(), blobVersionedHash.String())
	}
	// decode txs from blob
	err = codecv1.DecodeTxsFromBlob(blob, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to decode txs from blob: %w", err)
	}
	parentTotalL1MessagePopped := getBatchTotalL1MessagePopped(args.ParentBatchHeader)
	totalL1MessagePopped := 0
	for _, chunk := range chunks {
		for _, block := range chunk.Blocks {
			totalL1MessagePopped += int(block.NumL1Messages)
		}
	}
	skippedBitmap, err := encoding.DecodeBitmap(args.SkippedL1MessageBitmap, totalL1MessagePopped)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bitmap: %v, err: %w", batchIndex, err)
	}
	// get all necessary l1msgs without skipped
	currentIndex := parentTotalL1MessagePopped
	for index := 0; index < totalL1MessagePopped; index++ {
		for encoding.IsL1MessageSkipped(skippedBitmap, currentIndex-parentTotalL1MessagePopped) {
			currentIndex++
		}
		l1Tx := rawdb.ReadL1Message(ds.db, currentIndex)
		if l1Tx == nil {
			return nil, fmt.Errorf("failed to read L1 message from db, l1 message index: %v", currentIndex)
		}
		l1Txs = append(l1Txs, l1Tx)
		currentIndex++
	}
	da := NewCommitBatchDaV1(args.Version, batchIndex, parentTotalL1MessagePopped, args.SkippedL1MessageBitmap, chunks, l1Txs, vLog.BlockNumber)
	return da, nil
}

func (ds *CalldataBlobSource) decodeDAV2(batchIndex uint64, vLog *types.Log, args *commitBatchArgs) (DAEntry, error) {
	var chunks []*codecv2.DAChunkRawTx
	var l1Txs []*types.L1MessageTx
	chunks, err := codecv2.DecodeDAChunksRawTx(args.Chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack chunks: %v, err: %w", batchIndex, err)
	}

	versionedHash, err := ds.l1Client.fetchTxBlobHash(ds.ctx, vLog)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob hash, err: %w", err)
	}
	blob, err := ds.blobClient.GetBlobByVersionedHash(ds.ctx, versionedHash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob from blob client, err: %w", err)
	}
	// compute blob versioned hash and compare with one from tx
	c, err := kzg4844.BlobToCommitment(blob)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob commitment")
	}
	blobVersionedHash := common.Hash(kzg4844.CalcBlobHashV1(sha256.New(), &c))
	if blobVersionedHash != versionedHash {
		return nil, fmt.Errorf("blobVersionedHash from blob source is not equal to versionedHash from tx, correct versioned hash: %s, fetched blob hash: %s", versionedHash.String(), blobVersionedHash.String())
	}
	// decode txs from blob
	err = codecv2.DecodeTxsFromBlob(blob, chunks)
	if err != nil {
		return nil, fmt.Errorf("failed to decode txs from blob: %w", err)
	}
	parentTotalL1MessagePopped := getBatchTotalL1MessagePopped(args.ParentBatchHeader)
	totalL1MessagePopped := 0
	for _, chunk := range chunks {
		for _, block := range chunk.Blocks {
			totalL1MessagePopped += int(block.NumL1Messages)
		}
	}
	skippedBitmap, err := encoding.DecodeBitmap(args.SkippedL1MessageBitmap, totalL1MessagePopped)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bitmap: %v, err: %w", batchIndex, err)
	}
	// get all necessary l1msgs without skipped
	currentIndex := parentTotalL1MessagePopped
	for index := 0; index < totalL1MessagePopped; index++ {
		for encoding.IsL1MessageSkipped(skippedBitmap, currentIndex-parentTotalL1MessagePopped) {
			currentIndex++
		}
		l1Tx := rawdb.ReadL1Message(ds.db, currentIndex)
		if l1Tx == nil {
			return nil, fmt.Errorf("failed to read L1 message from db, l1 message index: %v", currentIndex)
		}
		l1Txs = append(l1Txs, l1Tx)
		currentIndex++
	}
	da := NewCommitBatchDaV2(args.Version, batchIndex, parentTotalL1MessagePopped, args.SkippedL1MessageBitmap, chunks, l1Txs, vLog.BlockNumber)
	return da, nil
}

func getBatchTotalL1MessagePopped(data []byte) uint64 {
	return binary.BigEndian.Uint64(data[17:25])
}
