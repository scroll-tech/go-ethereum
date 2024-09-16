// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package miner

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/ccc"
	"github.com/scroll-tech/go-ethereum/rollup/fees"
	"github.com/scroll-tech/go-ethereum/rollup/pipeline"
	"github.com/scroll-tech/go-ethereum/trie"
)

const (
	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096

	// chainHeadChanSize is the size of channel listening to ChainHeadEvent.
	chainHeadChanSize = 10
)

var (
	deadCh = make(chan time.Time)

	ErrUnexpectedL1MessageIndex = errors.New("unexpected L1 message index")

	// Metrics for the skipped txs
	l1SkippedCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l1", nil)
	l2SkippedCounter = metrics.NewRegisteredCounter("miner/skipped_txs/l2", nil)

	collectL1MsgsTimer = metrics.NewRegisteredTimer("miner/collect_l1_msgs", nil)
	prepareTimer       = metrics.NewRegisteredTimer("miner/prepare", nil)
	collectL2Timer     = metrics.NewRegisteredTimer("miner/collect_l2_txns", nil)
	l2CommitTimer      = metrics.NewRegisteredTimer("miner/commit", nil)
	cccStallTimer      = metrics.NewRegisteredTimer("miner/ccc_stall", nil)
	idleTimer          = metrics.NewRegisteredTimer("miner/idle", nil)

	commitReasonCCCCounter      = metrics.NewRegisteredCounter("miner/commit_reason_ccc", nil)
	commitReasonDeadlineCounter = metrics.NewRegisteredCounter("miner/commit_reason_deadline", nil)
	commitGasCounter            = metrics.NewRegisteredCounter("miner/commit_gas", nil)
)

// prioritizedTransaction represents a single transaction that
// should be processed as the first transaction in the next block.
type prioritizedTransaction struct {
	blockNumber uint64
	tx          *types.Transaction
}

// work represents the active block building task
type work struct {
	deadlineTimer   *time.Timer
	deadlineReached bool
	cccLogger       *ccc.Logger
	vmConfig        vm.Config

	reorgReason error

	// accumulated state
	nextL1MsgIndex uint64
	gasPool        *core.GasPool
	blockSize      common.StorageSize

	header        *types.Header
	state         *state.StateDB
	txs           types.Transactions
	receipts      types.Receipts
	coalescedLogs []*types.Log
}

func (w *work) deadlineCh() <-chan time.Time {
	if w == nil {
		return deadCh
	}
	return w.deadlineTimer.C
}

type reorgTrigger struct {
	block  *types.Block
	reason error
}

// worker is the main object which takes care of submitting new work to consensus engine
// and gathering the sealing result.
type worker struct {
	config      *Config
	chainConfig *params.ChainConfig
	engine      consensus.Engine
	eth         Backend
	chain       *core.BlockChain

	// Feeds
	pendingLogsFeed event.Feed

	// Subscriptions
	mux          *event.TypeMux
	txsCh        chan core.NewTxsEvent
	txsSub       event.Subscription
	chainHeadCh  chan core.ChainHeadEvent
	chainHeadSub event.Subscription

	// Channels
	startCh chan struct{}
	exitCh  chan struct{}
	reorgCh chan reorgTrigger

	wg sync.WaitGroup

	current *work

	mu       sync.RWMutex // The lock used to protect the coinbase and extra fields
	coinbase common.Address
	extra    []byte

	snapshotMu       sync.RWMutex // The lock used to protect the snapshots below
	snapshotBlock    *types.Block
	snapshotReceipts types.Receipts
	snapshotState    *state.StateDB

	// atomic status counters
	running atomic.Bool  // The indicator whether the consensus engine is running or not.
	newTxs  atomic.Int32 // New arrival transaction count since last sealing work submitting.
	syncing atomic.Bool  // The indicator whether the node is still syncing.

	// noempty is the flag used to control whether the feature of pre-seal empty
	// block is enabled. The default value is false(pre-seal is enabled by default).
	// But in some special scenario the consensus engine will seal blocks instantaneously,
	// in this case this feature will add all empty blocks into canonical chain
	// non-stop and no real transaction will be included.
	noempty uint32

	// External functions
	isLocalBlock func(block *types.Header) bool // Function used to determine whether the specified block is mined by local miner.

	prioritizedTx *prioritizedTransaction
	asyncChecker  *ccc.AsyncChecker

	// Test hooks
	beforeTxHook func() // Method to call before processing a transaction.

	errCountdown int
	skipTxHash   common.Hash
}

func newWorker(config *Config, chainConfig *params.ChainConfig, engine consensus.Engine, eth Backend, mux *event.TypeMux, isLocalBlock func(*types.Header) bool, init bool) *worker {
	worker := &worker{
		config:       config,
		chainConfig:  chainConfig,
		engine:       engine,
		eth:          eth,
		chain:        eth.BlockChain(),
		mux:          mux,
		isLocalBlock: isLocalBlock,
		coinbase:     config.Etherbase,
		extra:        config.ExtraData,
		txsCh:        make(chan core.NewTxsEvent, txChanSize),
		chainHeadCh:  make(chan core.ChainHeadEvent, chainHeadChanSize),
		exitCh:       make(chan struct{}),
		startCh:      make(chan struct{}, 1),
		reorgCh:      make(chan reorgTrigger, 1),
	}
	worker.asyncChecker = ccc.NewAsyncChecker(worker.chain, config.CCCMaxWorkers, false).WithOnFailingBlock(worker.onBlockFailingCCC)

	// Subscribe NewTxsEvent for tx pool
	worker.txsSub = eth.TxPool().SubscribeTransactions(worker.txsCh, true)

	// Subscribe events for blockchain
	worker.chainHeadSub = eth.BlockChain().SubscribeChainHeadEvent(worker.chainHeadCh)

	// Sanitize account fetch limit.
	if worker.config.MaxAccountsNum == 0 {
		log.Warn("Sanitizing miner account fetch limit", "provided", worker.config.MaxAccountsNum, "updated", math.MaxInt)
		worker.config.MaxAccountsNum = math.MaxInt
	}

	worker.wg.Add(1)
	go worker.mainLoop()

	// Submit first work to initialize pending state.
	if init {
		worker.startCh <- struct{}{}
	}
	return worker
}

// disablePreseal disables pre-sealing mining feature
func (w *worker) disablePreseal() {
	atomic.StoreUint32(&w.noempty, 1)
}

// enablePreseal enables pre-sealing mining feature
func (w *worker) enablePreseal() {
	atomic.StoreUint32(&w.noempty, 0)
}

// checkHeadRowConsumption will start some initial workers to CCC check block close to the HEAD
func (w *worker) checkHeadRowConsumption() error {
	checkStart := uint64(1)
	numOfBlocksToCheck := uint64(w.config.CCCMaxWorkers + 1)
	currentHeight := w.chain.CurrentHeader().Number.Uint64()
	if currentHeight > numOfBlocksToCheck {
		checkStart = currentHeight - numOfBlocksToCheck
	}

	for curBlockNum := checkStart; curBlockNum <= currentHeight; curBlockNum++ {
		block := w.chain.GetBlockByNumber(curBlockNum)
		// only spawn CCC checkers for blocks with no row consumption data stored in DB
		if rawdb.ReadBlockRowConsumption(w.chain.Database(), block.Hash()) == nil {
			if err := w.asyncChecker.Check(block); err != nil {
				return err
			}
		}
	}

	return nil
}

// mainLoop is a standalone goroutine to regenerate the sealing task based on the received event.
func (w *worker) mainLoop() {
	defer w.wg.Done()
	defer w.asyncChecker.Wait()
	defer w.txsSub.Unsubscribe()
	defer w.chainHeadSub.Unsubscribe()
	defer func() {
		// training wheels on
		// lets not crash the node and allow us some time to inspect
		p := recover()
		if p != nil {
			log.Error("worker mainLoop panic", "panic", p)
		}
	}()

	var err error
	for {
		if _, isRetryable := err.(retryableCommitError); isRetryable {
			if _, err = w.tryCommitNewWork(time.Now(), w.current.header.ParentHash, w.current.reorgReason); err != nil {
				continue
			}
		} else if err != nil {
			log.Error("failed to mine block", "err", err)
			w.current = nil
		}

		// check for reorgs first to lower the chances of trying to handle another
		// event eventhough a reorg is pending (due to Go `select` pseudo-randomly picking a case
		// to execute if multiple of them are ready)
		select {
		case trigger := <-w.reorgCh:
			err = w.handleReorg(&trigger)
			continue
		default:
		}

		select {
		case <-w.startCh:
			if err := w.checkHeadRowConsumption(); err != nil {
				log.Error("failed to start head checkers", "err", err)
				return
			}

			_, err = w.tryCommitNewWork(time.Now(), w.chain.CurrentHeader().Hash(), nil)
		case trigger := <-w.reorgCh:
			err = w.handleReorg(&trigger)
		case chainHead := <-w.chainHeadCh:
			if w.isCanonical(chainHead.Block.Header()) {
				_, err = w.tryCommitNewWork(time.Now(), chainHead.Block.Hash(), nil)
			}
		case <-w.current.deadlineCh():
			w.current.deadlineReached = true
			if len(w.current.txs) > 0 {
				_, err = w.commit(false)
			}
		case ev := <-w.txsCh:
			// Apply transactions to the pending state
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current mining block. These transactions will
			// be automatically eliminated.
			if w.current != nil {
				shouldCommit, _ := w.processTxnSlice(ev.Txs)
				if shouldCommit || w.current.deadlineReached {
					_, err = w.commit(false)
				}
			}
			// Apply transactions to the pending state
			//
			// Note all transactions received may not be continuous with transactions
			// already included in the current mining block. These transactions will
			// be automatically eliminated.
			//  if w.currentPipeline != nil {
			//  	txs := make(map[common.Address][]*txpool.LazyTransaction)
			//  	signer := types.MakeSigner(w.chainConfig, w.currentPipeline.Header.Number, w.currentPipeline.Header.Time)
			//  	for _, tx := range ev.Txs {
			//  		acc, _ := types.Sender(signer, tx)
			//  		txs[acc] = append(txs[acc], &txpool.LazyTransaction{
			//  			Pool:      w.eth.TxPool(), // We don't know where this came from, yolo resolve from everywhere
			//  			Hash:      tx.Hash(),
			//  			Tx:        nil, // Do *not* set this! We need to resolve it later to pull blobs in
			//  			Time:      tx.Time(),
			//  			GasFeeCap: tx.GasFeeCap(),
			//  			GasTipCap: tx.GasTipCap(),
			//  			Gas:       tx.Gas(),
			//  			BlobGas:   tx.BlobGas(),
			//  		})
			//  	}
			//  	txset := newTransactionsByPriceAndNonce(signer, txs, w.currentPipeline.Header.BaseFee)
			//  	if result := w.currentPipeline.TryPushTxns(txset, w.onTxFailingInPipeline); result != nil {
			//  		w.handlePipelineResult(result)
			//  	}
			//  }
			w.newTxs.Add(int32(len(ev.Txs)))

		// System stopped
		case <-w.exitCh:
			return
		case <-w.txsSub.Err():
			return
		case <-w.chainHeadSub.Err():
			return
		}
	}
}

// updateSnapshot updates pending snapshot block and state.
// Note this function assumes the current variable is thread safe.
func (w *worker) updateSnapshot() {
	w.snapshotMu.Lock()
	defer w.snapshotMu.Unlock()

	w.snapshotBlock = types.NewBlock(
		w.current.header,
		w.current.txs,
		nil,
		w.current.receipts,
		trie.NewStackTrie(nil),
	)
	w.snapshotReceipts = copyReceipts(w.current.receipts)
	w.snapshotState = w.current.state.Copy()
}

func (w *worker) collectPendingL1Messages(startIndex uint64) []types.L1MessageTx {
	maxCount := w.chainConfig.Scroll.L1Config.NumL1MessagesPerBlock
	return rawdb.ReadL1MessagesFrom(w.eth.ChainDb(), startIndex, maxCount)
}

// newWork
func (w *worker) newWork(now time.Time, parentHash common.Hash, reorgReason error) error {
	parent := w.chain.GetBlockByHash(parentHash)
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		GasLimit:   core.CalcGasLimit(parent.GasLimit(), w.config.GasCeil),
		Extra:      w.extra,
		Time:       uint64(now.Unix()),
	}

	parentState, err := w.chain.StateAt(parent.Root())
	if err != nil {
		return fmt.Errorf("failed to fetch parent state: %w", err)
	}

	// Set baseFee if we are on an EIP-1559 chain
	if w.chainConfig.IsCurie(header.Number) {
		parentL1BaseFee := fees.GetL1BaseFee(parentState)
		header.BaseFee = misc.CalcBaseFee(w.chainConfig, parent.Header(), parentL1BaseFee)
	}
	// Only set the coinbase if our consensus engine is running (avoid spurious block rewards)
	if w.isRunning() {
		if w.coinbase == (common.Address{}) {
			return errors.New("refusing to mine without etherbase")
		}
		header.Coinbase = w.coinbase
	}

	prepareStart := time.Now()
	if err := w.engine.Prepare(w.chain, header); err != nil {
		return fmt.Errorf("failed to prepare header for mining: %w", err)
	}
	prepareTimer.UpdateSince(prepareStart)

	var nextL1MsgIndex uint64
	if dbVal := rawdb.ReadFirstQueueIndexNotInL2Block(w.eth.ChainDb(), header.ParentHash); dbVal != nil {
		nextL1MsgIndex = *dbVal
	}

	vmConfig := *w.chain.GetVMConfig()
	cccLogger := ccc.NewLogger()
	vmConfig.Debug = true
	vmConfig.Tracer = cccLogger

	deadline := time.Unix(int64(header.Time), 0)
	if w.chainConfig.Clique != nil && w.chainConfig.Clique.RelaxedPeriod {
		// clique with relaxed period uses time.Now() as the header.Time, calculate the deadline
		deadline = time.Unix(int64(header.Time+w.chainConfig.Clique.Period), 0)
	}

	w.current = &work{
		deadlineTimer:  time.NewTimer(time.Until(deadline)),
		cccLogger:      cccLogger,
		vmConfig:       vmConfig,
		header:         header,
		state:          parentState,
		txs:            types.Transactions{},
		receipts:       types.Receipts{},
		coalescedLogs:  []*types.Log{},
		gasPool:        new(core.GasPool).AddGas(header.GasLimit),
		nextL1MsgIndex: nextL1MsgIndex,
		reorgReason:    reorgReason,
	}
	return nil
}

// retryableCommitError wraps an error that happened during commit phase and indicates that worker can retry to build a new block
type retryableCommitError struct {
	inner error
}

func (e retryableCommitError) Error() string {
	return e.inner.Error()
}

func (e retryableCommitError) Unwrap() error {
	return e.inner
}

// commit runs any post-transaction state modifications, assembles the final block
// and commits new work if consensus engine is running.
func (w *worker) commit(res *pipeline.Result) error {
	sealDelay := time.Duration(0)
	defer func(t0 time.Time) {
		l2CommitTimer.Update(time.Since(t0) - sealDelay)
	}(time.Now())

	if res.CCCErr != nil {
		commitReasonCCCCounter.Inc(1)
	} else {
		commitReasonDeadlineCounter.Inc(1)
	}
	commitGasCounter.Inc(int64(res.FinalBlock.Header.GasUsed))

	block, err := w.engine.FinalizeAndAssemble(w.chain, res.FinalBlock.Header, res.FinalBlock.State,
		res.FinalBlock.Txs, nil, res.FinalBlock.Receipts, nil)
	if err != nil {
		return err
	}

	sealHash := w.engine.SealHash(block.Header())
	log.Info("Committing new mining work", "number", block.Number(), "sealhash", sealHash,
		"txs", res.FinalBlock.Txs.Len(),
		"gas", block.GasUsed(), "fees", totalFees(block, res.FinalBlock.Receipts),
		"elapsed", common.PrettyDuration(time.Since(w.currentPipelineStart)))

	resultCh, stopCh := make(chan *types.Block), make(chan struct{})
	if err := w.engine.Seal(w.chain, block, resultCh, stopCh); err != nil {
		return err
	}
	// Clique.Seal() will only wait for a second before giving up on us. So make sure there is nothing computational heavy
	// or a call that blocks between the call to Seal and the line below. Seal might introduce some delay, so we keep track of
	// that artificially added delay and subtract it from overall runtime of commit().
	sealStart := time.Now()
	block = <-resultCh
	sealDelay = time.Since(sealStart)
	if block == nil {
		return errors.New("missed seal response from consensus engine")
	}

	// verify the generated block with local consensus engine to make sure everything is as expected
	if err = w.engine.VerifyHeader(w.chain, block.Header()); err != nil {
		return retryableCommitError{inner: err}
	}

	blockHash := block.Hash()
	var logs []*types.Log
	for i, receipt := range res.FinalBlock.Receipts {
		// add block location fields
		receipt.BlockHash = blockHash
		receipt.BlockNumber = block.Number()
		receipt.TransactionIndex = uint(i)

		for _, log := range receipt.Logs {
			log.BlockHash = blockHash
		}

		logs = append(logs, receipt.Logs...)
	}

	for _, log := range res.FinalBlock.CoalescedLogs {
		log.BlockHash = blockHash
	}

	// It's possible that we've stored L1 queue index for this block previously,
	// in this case do not overwrite it.
	if index := rawdb.ReadFirstQueueIndexNotInL2Block(w.eth.ChainDb(), blockHash); index == nil {
		// Store first L1 queue index not processed by this block.
		// Note: This accounts for both included and skipped messages. This
		// way, if a block only skips messages, we won't reprocess the same
		// messages from the next block.
		log.Trace(
			"Worker WriteFirstQueueIndexNotInL2Block",
			"number", block.Number(),
			"hash", blockHash.String(),
			"nextL1MsgIndex", res.FinalBlock.NextL1MsgIndex,
		)
		rawdb.WriteFirstQueueIndexNotInL2Block(w.eth.ChainDb(), blockHash, res.FinalBlock.NextL1MsgIndex)
	} else {
		log.Trace(
			"Worker WriteFirstQueueIndexNotInL2Block: not overwriting existing index",
			"number", block.Number(),
			"hash", blockHash.String(),
			"index", *index,
			"nextL1MsgIndex", res.FinalBlock.NextL1MsgIndex,
		)
	}
	// Store circuit row consumption.
	log.Trace(
		"Worker write block row consumption",
		"id", w.circuitCapacityChecker.ID,
		"number", block.Number(),
		"hash", blockHash.String(),
		"accRows", res.Rows,
	)

	// A new block event will trigger a reorg in the txpool, pause reorgs to defer this until we fetch txns for next block.
	// We may end up trying to process txns that we already included in the previous block, but they will all fail the nonce check
	w.eth.TxPool().PauseReorgs()

	rawdb.WriteBlockRowConsumption(w.eth.ChainDb(), blockHash, res.Rows)
	// Commit block and state to database.
	_, err = w.chain.WriteBlockAndSetHead(block, res.FinalBlock.Receipts, logs, res.FinalBlock.State, true)
	if err != nil {
		log.Error("Failed writing block to chain", "err", err)
		return err
	}

	log.Info("Successfully sealed new block", "number", block.Number(), "sealhash", sealHash, "hash", blockHash)

	// Broadcast the block and announce chain insertion event
	w.mux.Post(core.NewMinedBlockEvent{Block: block})

	return nil
}

// copyReceipts makes a deep copy of the given receipts.
func copyReceipts(receipts []*types.Receipt) []*types.Receipt {
	result := make([]*types.Receipt, len(receipts))
	for i, l := range receipts {
		cpy := *l
		result[i] = &cpy
	}
	return result
}

func (w *worker) onTxFailingInPipeline(txIndex int, tx *types.Transaction, err error) bool {
	if !w.isRunning() {
		return false
	}

	writeTrace := func() {
		var trace *types.BlockTrace
		var errWithTrace *pipeline.ErrorWithTrace
		if w.config.StoreSkippedTxTraces && errors.As(err, &errWithTrace) {
			trace = errWithTrace.Trace
		}
		rawdb.WriteSkippedTransaction(w.eth.ChainDb(), tx, trace, err.Error(),
			w.currentPipeline.Header.Number.Uint64(), nil)
	}

	switch {
	case errors.Is(err, core.ErrGasLimitReached) && tx.IsL1MessageTx():
		// If this block already contains some L1 messages try again in the next block.
		if txIndex > 0 {
			break
		}
		// A single L1 message leads to out-of-gas. Skip it.
		queueIndex := tx.AsL1MessageTx().QueueIndex
		log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block",
			w.currentPipeline.Header.Number, "reason", "gas limit exceeded")
		writeTrace()
		l1TxGasLimitExceededCounter.Inc(1)

	case errors.Is(err, core.ErrInsufficientFunds):
		log.Trace("Skipping tx with insufficient funds", "tx", tx.Hash().String())
		w.eth.TxPool().RemoveTx(tx.Hash(), true, true)

	case errors.Is(err, pipeline.ErrUnexpectedL1MessageIndex):
		log.Warn(
			"Unexpected L1 message queue index in worker",
			"got", tx.AsL1MessageTx().QueueIndex,
		)
	case errors.Is(err, core.ErrGasLimitReached), errors.Is(err, core.ErrNonceTooLow), errors.Is(err, core.ErrNonceTooHigh), errors.Is(err, core.ErrTxTypeNotSupported):
		break
	default:
		// Strange error
		log.Debug("Transaction failed, account skipped", "hash", tx.Hash().String(), "err", err)
		if tx.IsL1MessageTx() {
			queueIndex := tx.AsL1MessageTx().QueueIndex
			log.Info("Skipping L1 message", "queueIndex", queueIndex, "tx", tx.Hash().String(), "block",
				w.currentPipeline.Header.Number, "reason", "strange error", "err", err)
			writeTrace()
			l1TxStrangeErrCounter.Inc(1)
		}
	}
	return false
}

// totalFees computes total consumed miner fees in ETH. Block transactions and receipts have to have the same order.
func totalFees(block *types.Block, receipts []*types.Receipt) *big.Float {
	feesWei := new(big.Int)
	for i, tx := range block.Transactions() {
		minerFee, _ := tx.EffectiveGasTip(block.BaseFee())
		feesWei.Add(feesWei, new(big.Int).Mul(new(big.Int).SetUint64(receipts[i].GasUsed), minerFee))
	}
	return new(big.Float).Quo(new(big.Float).SetInt(feesWei), new(big.Float).SetInt(big.NewInt(params.Ether)))
}
