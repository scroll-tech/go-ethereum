package rollup_sync_service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"

	"github.com/scroll-tech/go-ethereum/rollup/rcfg"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
	"github.com/scroll-tech/go-ethereum/rollup/types/encoding"
	"github.com/scroll-tech/go-ethereum/rollup/types/encoding/codecv0"
	"github.com/scroll-tech/go-ethereum/rollup/types/encoding/codecv1"
	"github.com/scroll-tech/go-ethereum/rollup/withdrawtrie"
)

const (
	// defaultFetchBlockRange is the number of blocks that we collect in a single eth_getLogs query.
	defaultFetchBlockRange = uint64(100)

	// defaultSyncInterval is the frequency at which we query for new rollup event.
	defaultSyncInterval = 60 * time.Second

	// defaultMaxRetries is the maximum number of retries allowed when the local node is not synced up to the required block height.
	defaultMaxRetries = 20

	// defaultGetBlockInRangeRetryDelay is the time delay between retries when attempting to get blocks in range.
	// The service will wait for this duration if it detects that the local node has not synced up to the block height
	// of a specific L1 batch finalize event.
	defaultGetBlockInRangeRetryDelay = 60 * time.Second

	// defaultLogInterval is the frequency at which we print the latestProcessedBlock.
	defaultLogInterval = 5 * time.Minute
)

// RollupSyncService collects ScrollChain batch commit/revert/finalize events and stores metadata into db.
type RollupSyncService struct {
	ctx                           context.Context
	cancel                        context.CancelFunc
	client                        *L1Client
	db                            ethdb.Database
	latestProcessedBlock          uint64
	scrollChainABI                *abi.ABI
	l1CommitBatchEventSignature   common.Hash
	l1RevertBatchEventSignature   common.Hash
	l1FinalizeBatchEventSignature common.Hash
	bc                            *core.BlockChain
	stack                         *node.Node
}

func NewRollupSyncService(ctx context.Context, genesisConfig *params.ChainConfig, db ethdb.Database, l1Client sync_service.EthClient, bc *core.BlockChain, stack *node.Node) (*RollupSyncService, error) {
	// terminate if the caller does not provide an L1 client (e.g. in tests)
	if l1Client == nil || (reflect.ValueOf(l1Client).Kind() == reflect.Ptr && reflect.ValueOf(l1Client).IsNil()) {
		log.Warn("No L1 client provided, L1 rollup sync service will not run")
		return nil, nil
	}

	if genesisConfig.Scroll.L1Config == nil {
		return nil, fmt.Errorf("missing L1 config in genesis")
	}

	scrollChainABI, err := scrollChainMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to get scroll chain abi: %w", err)
	}

	client, err := newL1Client(ctx, l1Client, genesisConfig.Scroll.L1Config.L1ChainId, genesisConfig.Scroll.L1Config.ScrollChainAddress, scrollChainABI)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize l1 client: %w", err)
	}

	// Initialize the latestProcessedBlock with the block just before the L1 deployment block.
	// This serves as a default value when there's no L1 rollup events synced in the database.
	var latestProcessedBlock uint64
	if stack.Config().L1DeploymentBlock > 0 {
		latestProcessedBlock = stack.Config().L1DeploymentBlock - 1
	}

	block := rawdb.ReadRollupEventSyncedL1BlockNumber(db)
	if block != nil {
		// restart from latest synced block number
		latestProcessedBlock = *block
	}

	ctx, cancel := context.WithCancel(ctx)

	service := RollupSyncService{
		ctx:                           ctx,
		cancel:                        cancel,
		client:                        client,
		db:                            db,
		latestProcessedBlock:          latestProcessedBlock,
		scrollChainABI:                scrollChainABI,
		l1CommitBatchEventSignature:   scrollChainABI.Events["CommitBatch"].ID,
		l1RevertBatchEventSignature:   scrollChainABI.Events["RevertBatch"].ID,
		l1FinalizeBatchEventSignature: scrollChainABI.Events["FinalizeBatch"].ID,
		bc:                            bc,
		stack:                         stack,
	}

	return &service, nil
}

func (s *RollupSyncService) Start() {
	if s == nil {
		return
	}

	log.Info("Starting rollup event sync background service", "latest processed block", s.latestProcessedBlock)

	go func() {
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()

		logTicker := time.NewTicker(defaultLogInterval)
		defer logTicker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-syncTicker.C:
				s.fetchRollupEvents()
			case <-logTicker.C:
				log.Info("Sync rollup events progress update", "latestProcessedBlock", s.latestProcessedBlock)
			}
		}
	}()
}

func (s *RollupSyncService) Stop() {
	if s == nil {
		return
	}

	log.Info("Stopping rollup event sync background service")

	if s.cancel != nil {
		s.cancel()
	}
}

func (s *RollupSyncService) fetchRollupEvents() {
	latestConfirmed, err := s.client.getLatestFinalizedBlockNumber()
	if err != nil {
		log.Warn("failed to get latest confirmed block number", "err", err)
		return
	}

	log.Trace("Sync service fetch rollup events", "latest processed block", s.latestProcessedBlock, "latest confirmed", latestConfirmed)

	// query in batches
	for from := s.latestProcessedBlock + 1; from <= latestConfirmed; from += defaultFetchBlockRange {
		if s.ctx.Err() != nil {
			log.Info("Context canceled", "reason", s.ctx.Err())
			return
		}

		to := from + defaultFetchBlockRange - 1
		if to > latestConfirmed {
			to = latestConfirmed
		}

		logs, err := s.client.fetchRollupEventsInRange(from, to)
		if err != nil {
			log.Error("failed to fetch rollup events in range", "from block", from, "to block", to, "err", err)
			return
		}

		if err := s.parseAndUpdateRollupEventLogs(logs, to); err != nil {
			log.Error("failed to parse and update rollup event logs", "err", err)
			return
		}

		s.latestProcessedBlock = to
	}
}

func (s *RollupSyncService) parseAndUpdateRollupEventLogs(logs []types.Log, endBlockNumber uint64) error {
	for _, vLog := range logs {
		switch vLog.Topics[0] {
		case s.l1CommitBatchEventSignature:
			event := &L1CommitBatchEvent{}
			if err := UnpackLog(s.scrollChainABI, event, "CommitBatch", vLog); err != nil {
				return fmt.Errorf("failed to unpack commit rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new CommitBatch event", "batch index", batchIndex)

			chunkBlockRanges, err := s.getChunkRanges(batchIndex, &vLog)
			if err != nil {
				return fmt.Errorf("failed to get chunk ranges, batch index: %v, err: %w", batchIndex, err)
			}
			rawdb.WriteBatchChunkRanges(s.db, batchIndex, chunkBlockRanges)

		case s.l1RevertBatchEventSignature:
			event := &L1RevertBatchEvent{}
			if err := UnpackLog(s.scrollChainABI, event, "RevertBatch", vLog); err != nil {
				return fmt.Errorf("failed to unpack revert rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new RevertBatch event", "batch index", batchIndex)

			rawdb.DeleteBatchChunkRanges(s.db, batchIndex)

		case s.l1FinalizeBatchEventSignature:
			event := &L1FinalizeBatchEvent{}
			if err := UnpackLog(s.scrollChainABI, event, "FinalizeBatch", vLog); err != nil {
				return fmt.Errorf("failed to unpack finalized rollup event log, err: %w", err)
			}
			batchIndex := event.BatchIndex.Uint64()
			log.Trace("found new FinalizeBatch event", "batch index", batchIndex)

			parentBatchMeta, chunks, err := s.getLocalInfoForBatch(batchIndex)
			if err != nil {
				return fmt.Errorf("failed to get local node info, batch index: %v, err: %w", batchIndex, err)
			}

			if len(chunks) == 0 {
				return fmt.Errorf("invalid argument: length of chunks is 0, batch index: %v", event.BatchIndex.Uint64())
			}

			endBlock, finalizedBatchMeta, err := validateBatch(event, parentBatchMeta, chunks, s.bc.Config(), s.stack)
			if err != nil {
				return fmt.Errorf("fatal: validateBatch failed: finalize event: %v, err: %w", event, err)
			}

			rawdb.WriteFinalizedL2BlockNumber(s.db, endBlock)
			rawdb.WriteFinalizedBatchMeta(s.db, batchIndex, finalizedBatchMeta)

			if batchIndex%100 == 0 {
				log.Info("finalized batch progress", "batch index", batchIndex, "finalized l2 block height", endBlock)
			}

		default:
			return fmt.Errorf("unknown event, topic: %v, tx hash: %v", vLog.Topics[0].Hex(), vLog.TxHash.Hex())
		}
	}

	// note: the batch updates above are idempotent, if we crash
	// before this line and reexecute the previous steps, we will
	// get the same result.
	rawdb.WriteRollupEventSyncedL1BlockNumber(s.db, endBlockNumber)
	return nil
}

func (s *RollupSyncService) getLocalInfoForBatch(batchIndex uint64) (*rawdb.FinalizedBatchMeta, []*encoding.Chunk, error) {
	chunkBlockRanges := rawdb.ReadBatchChunkRanges(s.db, batchIndex)
	if len(chunkBlockRanges) == 0 {
		return nil, nil, fmt.Errorf("failed to get batch chunk ranges, empty chunk block ranges")
	}

	endBlockNumber := chunkBlockRanges[len(chunkBlockRanges)-1].EndBlockNumber
	for i := 0; i < defaultMaxRetries; i++ {
		if s.ctx.Err() != nil {
			log.Info("Context canceled", "reason", s.ctx.Err())
			return nil, nil, s.ctx.Err()
		}

		localSyncedBlockHeight := s.bc.CurrentBlock().Number().Uint64()
		if localSyncedBlockHeight >= endBlockNumber {
			break // ready to proceed, exit retry loop
		}

		log.Debug("local node is not synced up to the required block height, waiting for next retry",
			"retries", i+1, "local synced block height", localSyncedBlockHeight, "required end block number", endBlockNumber)
		time.Sleep(defaultGetBlockInRangeRetryDelay)
	}

	localSyncedBlockHeight := s.bc.CurrentBlock().Number().Uint64()
	if localSyncedBlockHeight < endBlockNumber {
		return nil, nil, fmt.Errorf("local node is not synced up to the required block height: %v, local synced block height: %v", endBlockNumber, localSyncedBlockHeight)
	}

	chunks := make([]*encoding.Chunk, len(chunkBlockRanges))
	for i, cr := range chunkBlockRanges {
		chunks[i] = &encoding.Chunk{Blocks: make([]*encoding.Block, cr.EndBlockNumber-cr.StartBlockNumber+1)}
		for j := cr.StartBlockNumber; j <= cr.EndBlockNumber; j++ {
			block := s.bc.GetBlockByNumber(j)
			if block == nil {
				return nil, nil, fmt.Errorf("failed to get block by number: %v", i)
			}
			txData := encoding.TxsToTxsData(block.Transactions())
			state, err := s.bc.StateAt(block.Root())
			if err != nil {
				return nil, nil, fmt.Errorf("failed to get block state, block: %v, err: %w", block.Hash().Hex(), err)
			}
			withdrawRoot := withdrawtrie.ReadWTRSlot(rcfg.L2MessageQueueAddress, state)
			chunks[i].Blocks[j-cr.StartBlockNumber] = &encoding.Block{
				Header:       block.Header(),
				Transactions: txData,
				WithdrawRoot: withdrawRoot,
			}
		}
	}

	// get metadata of parent batch: default to genesis batch metadata.
	parentBatchMeta := &rawdb.FinalizedBatchMeta{}
	if batchIndex > 0 {
		parentBatchMeta = rawdb.ReadFinalizedBatchMeta(s.db, batchIndex-1)
	}

	return parentBatchMeta, chunks, nil
}

func (s *RollupSyncService) getChunkRanges(batchIndex uint64, vLog *types.Log) ([]*rawdb.ChunkBlockRange, error) {
	if batchIndex == 0 {
		return []*rawdb.ChunkBlockRange{{StartBlockNumber: 0, EndBlockNumber: 0}}, nil
	}

	tx, _, err := s.client.client.TransactionByHash(s.ctx, vLog.TxHash)
	if err != nil {
		log.Debug("failed to get transaction by hash, probably an unindexed transaction, fetching the whole block to get the transaction",
			"tx hash", vLog.TxHash.Hex(), "block number", vLog.BlockNumber, "block hash", vLog.BlockHash.Hex(), "err", err)
		block, err := s.client.client.BlockByHash(s.ctx, vLog.BlockHash)
		if err != nil {
			return nil, fmt.Errorf("failed to get block by hash, block number: %v, block hash: %v, err: %w", vLog.BlockNumber, vLog.BlockHash.Hex(), err)
		}

		found := false
		for _, txInBlock := range block.Transactions() {
			if txInBlock.Hash() == vLog.TxHash {
				tx = txInBlock
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("transaction not found in the block, tx hash: %v, block number: %v, block hash: %v", vLog.TxHash.Hex(), vLog.BlockNumber, vLog.BlockHash.Hex())
		}
	}

	return s.decodeChunkBlockRanges(tx.Data())
}

// decodeChunkBlockRanges decodes chunks in a batch based on the commit batch transaction's calldata.
func (s *RollupSyncService) decodeChunkBlockRanges(txData []byte) ([]*rawdb.ChunkBlockRange, error) {
	const methodIDLength = 4
	if len(txData) < methodIDLength {
		return nil, fmt.Errorf("transaction data is too short, length of tx data: %v, minimum length required: %v", len(txData), methodIDLength)
	}

	method, err := s.scrollChainABI.MethodById(txData[:methodIDLength])
	if err != nil {
		return nil, fmt.Errorf("failed to get method by ID, ID: %v, err: %w", txData[:methodIDLength], err)
	}

	values, err := method.Inputs.Unpack(txData[methodIDLength:])
	if err != nil {
		return nil, fmt.Errorf("failed to unpack transaction data using ABI, tx data: %v, err: %w", txData, err)
	}

	type commitBatchArgs struct {
		Version                uint8
		ParentBatchHeader      []byte
		Chunks                 [][]byte
		SkippedL1MessageBitmap []byte
	}
	var args commitBatchArgs
	if err = method.Inputs.Copy(&args, values); err != nil {
		return nil, fmt.Errorf("failed to decode calldata into commitBatch args, values: %+v, err: %w", values, err)
	}

	return decodeBlockRangesFromEncodedChunks(encoding.CodecVersion(args.Version), args.Chunks)
}

// validateBatch verifies the consistency between the L1 contract and L2 node data.
// The function will terminate the node and exit if any consistency check fails.
// It returns the number of the end block, a finalized batch meta data, and an error if any.
func validateBatch(event *L1FinalizeBatchEvent, parentBatchMeta *rawdb.FinalizedBatchMeta, chunks []*encoding.Chunk, chainCfg *params.ChainConfig, stack *node.Node) (uint64, *rawdb.FinalizedBatchMeta, error) {
	if len(chunks) == 0 {
		return 0, nil, fmt.Errorf("invalid argument: length of chunks is 0, batch index: %v", event.BatchIndex.Uint64())
	}

	startChunk := chunks[0]
	if len(startChunk.Blocks) == 0 {
		return 0, nil, fmt.Errorf("invalid argument: block count of start chunk is 0, batch index: %v", event.BatchIndex.Uint64())
	}
	startBlock := startChunk.Blocks[0]

	endChunk := chunks[len(chunks)-1]
	if len(endChunk.Blocks) == 0 {
		return 0, nil, fmt.Errorf("invalid argument: block count of end chunk is 0, batch index: %v", event.BatchIndex.Uint64())
	}
	endBlock := endChunk.Blocks[len(endChunk.Blocks)-1]

	localStateRoot := endBlock.Header.Root
	if localStateRoot != event.StateRoot {
		log.Error("State root mismatch", "batch index", event.BatchIndex.Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentBatchMeta.BatchHash.Hex(), "l1 finalized state root", event.StateRoot.Hex(), "l2 state root", localStateRoot.Hex())
		stack.Close()
		os.Exit(1)
	}

	localWithdrawRoot := endBlock.WithdrawRoot
	if localWithdrawRoot != event.WithdrawRoot {
		log.Error("Withdraw root mismatch", "batch index", event.BatchIndex.Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentBatchMeta.BatchHash.Hex(), "l1 finalized withdraw root", event.WithdrawRoot.Hex(), "l2 withdraw root", localWithdrawRoot.Hex())
		stack.Close()
		os.Exit(1)
	}

	// Note: All params of batch are calculated locally based on the block data.
	batch := &encoding.Batch{
		Index:                      event.BatchIndex.Uint64(),
		TotalL1MessagePoppedBefore: parentBatchMeta.TotalL1MessagePopped,
		ParentBatchHash:            parentBatchMeta.BatchHash,
		Chunks:                     chunks,
	}

	var localBatchHash common.Hash
	if !chainCfg.IsBanach(startBlock.Header.Number) {
		daBatch, err := codecv0.NewDABatch(batch)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create codecv0 DA batch, batch index: %v, err: %w", event.BatchIndex.Uint64(), err)
		}
		localBatchHash = daBatch.Hash()
	} else {
		daBatch, err := codecv1.NewDABatch(batch)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to create codecv1 DA batch, batch index: %v, err: %w", event.BatchIndex.Uint64(), err)
		}
		localBatchHash = daBatch.Hash()
	}

	// Note: If the batch headers match, this ensures the consistency of blocks and transactions
	// (including skipped transactions) between L1 and L2.
	if localBatchHash != event.BatchHash {
		log.Error("Batch hash mismatch", "batch index", event.BatchIndex.Uint64(), "start block", startBlock.Header.Number.Uint64(), "end block", endBlock.Header.Number.Uint64(), "parent batch hash", parentBatchMeta.BatchHash.Hex(), "parent TotalL1MessagePopped", parentBatchMeta.TotalL1MessagePopped, "l1 finalized batch hash", event.BatchHash.Hex(), "l2 batch hash", localBatchHash.Hex())
		chunksJson, err := json.Marshal(chunks)
		if err != nil {
			log.Error("marshal chunks failed", "err", err)
		}
		log.Error("Chunks", "chunks", string(chunksJson))
		stack.Close()
		os.Exit(1)
	}

	totalL1MessagePopped := parentBatchMeta.TotalL1MessagePopped
	for _, chunk := range chunks {
		totalL1MessagePopped += chunk.NumL1Messages(totalL1MessagePopped)
	}
	finalizedBatchMeta := &rawdb.FinalizedBatchMeta{
		BatchHash:            localBatchHash,
		TotalL1MessagePopped: totalL1MessagePopped,
		StateRoot:            localStateRoot,
		WithdrawRoot:         localWithdrawRoot,
	}
	return endBlock.Header.Number.Uint64(), finalizedBatchMeta, nil
}

// decodeBlockRangesFromEncodedChunks decodes the provided chunks into a list of block ranges.
func decodeBlockRangesFromEncodedChunks(codecVersion encoding.CodecVersion, chunks [][]byte) ([]*rawdb.ChunkBlockRange, error) {
	var chunkBlockRanges []*rawdb.ChunkBlockRange
	for _, chunk := range chunks {
		if len(chunk) < 1 {
			return nil, fmt.Errorf("invalid chunk, length is less than 1")
		}

		numBlocks := int(chunk[0])

		switch codecVersion {
		case encoding.CodecV0:
			if len(chunk) < 1+numBlocks*60 {
				return nil, fmt.Errorf("chunk size doesn't match with numBlocks, byte length of chunk: %v, expected minimum length: %v", len(chunk), 1+numBlocks*60)
			}
		case encoding.CodecV1:
			if len(chunk) == 1+numBlocks*60 {
				return nil, fmt.Errorf("chunk size doesn't match with numBlocks, byte length of chunk: %v, expected length: %v", len(chunk), 1+numBlocks*60)
			}
		default:
			return nil, fmt.Errorf("unexpected batch version %v", codecVersion)
		}

		blockContexts := make([]*encoding.BlockContext, numBlocks)
		for i := 0; i < numBlocks; i++ {
			startIdx := 1 + i*60 // add 1 to skip numBlocks byte
			endIdx := startIdx + 60
			blockContext, err := encoding.DecodeBlockContext(chunk[startIdx:endIdx])
			if err != nil {
				return nil, err
			}
			blockContexts[i] = blockContext
		}

		chunkBlockRanges = append(chunkBlockRanges, &rawdb.ChunkBlockRange{
			StartBlockNumber: blockContexts[0].BlockNumber,
			EndBlockNumber:   blockContexts[len(blockContexts)-1].BlockNumber,
		})
	}
	return chunkBlockRanges, nil
}
