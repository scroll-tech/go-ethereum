package simulated

import (
	"context"
	"fmt"
	"math/big"

	"github.com/cockroachdb/errors"
	"github.com/holiman/uint256"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv3"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind/backends"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/simulated/contracts"
)

type L2 struct {
	keyManager *KeyManager
	backend    *backends.SimulatedBackend

	scrollChainAddress common.Address
	scrollChain        *contracts.ScrollChainMockFinalize
	l1MessageQueue     *contracts.L1MessageQueue
	l1Sender           func(transaction *types.Transaction) error

	batches            map[uint64]*Batch
	lastProducedBlock  *types.Block
	lastCommittedBatch *Batch
	lastFinalizedBatch *Batch
}

func NewL2(km *KeyManager, l1Sender func(transaction *types.Transaction) error, scrollChain *contracts.ScrollChainMockFinalize, l1MessageQueue *contracts.L1MessageQueue) (*L2, error) {
	gAlloc := core.GenesisAlloc{
		km.Address(defaultKeyAlias): {Balance: new(big.Int).SetUint64(1 * params.Ether)},
	}
	backend := backends.NewSimulatedBackend(gAlloc, 1000000000)

	fmt.Println("Started simulated L1 with following accounts:")
	for address, genesisAccount := range gAlloc {
		fmt.Printf("\tAddress: %s, %d\n", address, genesisAccount.Balance)
	}

	l2 := &L2{
		keyManager:     km,
		backend:        backend,
		scrollChain:    scrollChain,
		l1MessageQueue: l1MessageQueue,
		l1Sender:       l1Sender,
		batches:        make(map[uint64]*Batch),
	}

	return l2, nil
}

// TODO: create methods for other txs types as well
func (l2 *L2) SendDynamicFeeTransaction(fromAlias string, toAlias string, value *big.Int, data []byte, commit bool) (*types.Transaction, error) {
	fromAddress := l2.keyManager.Address(fromAlias)
	var toAddress *common.Address
	if toAlias != "" {
		toAddressNonNil := l2.keyManager.Address(toAlias)
		toAddress = &toAddressNonNil
	}

	nonce, err := l2.backend.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get nonce for %s, %s", fromAlias, fromAddress)
	}

	gasLimit := uint64(10000000)
	gasPrice, err := l2.backend.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get suggested gas price")
	}

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   l2.backend.Blockchain().Config().ChainID,
		Nonce:     nonce,
		GasTipCap: gasPrice,
		GasFeeCap: gasPrice,
		Gas:       gasLimit,
		To:        toAddress,
		Value:     value,
		Data:      data,
	})

	signer := types.LatestSigner(l2.backend.Blockchain().Config())
	signedTx, err := types.SignTx(tx, signer, l2.keyManager.Key(fromAlias))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sign tx for %s, %s", fromAlias, fromAddress)
	}

	err = l2.backend.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send tx for %s, %s", fromAlias, fromAddress)
	}

	fmt.Println("Sent transaction", "tx", signedTx.Hash().Hex(), "from", fromAddress, "to", toAddress, "value", value, "gasPrice", gasPrice, "gasLimit", gasLimit)

	if commit {
		l2.CommitBlock()
	}

	return signedTx, nil
}

type Batch struct {
	DABatch *codecv3.DABatch
	*encoding.Batch
	LastL2BlockNumber uint64
	Hash              common.Hash
}

func (b *Batch) StateRoot() common.Hash {
	lastChunk := len(b.Chunks) - 1
	lastBlock := len(b.Chunks[lastChunk].Blocks) - 1
	return b.Chunks[lastChunk].Blocks[lastBlock].Header.Root
}

func (l2 *L2) CommitBlock() *types.Block {
	hash := l2.backend.Commit()

	var err error
	l2.lastProducedBlock, err = l2.backend.BlockByHash(context.Background(), hash)

	// this should never happen as we just committed the block
	if err != nil {
		panic(err)
	}

	return l2.lastProducedBlock
}

func (l2 *L2) commitGenesisBatch() (*Batch, error) {
	genesis := l2.backend.Blockchain().CurrentHeader()
	if genesis.Number.Uint64() != 0 {
		return nil, errors.New("genesis batch can only be committed at block 0")
	}

	chunk := &encoding.Chunk{
		Blocks: []*encoding.Block{{
			Header:         genesis,
			Transactions:   nil,
			WithdrawRoot:   common.Hash{},
			RowConsumption: &types.RowConsumption{},
		}},
	}

	batch := &encoding.Batch{
		Index:                      0,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            common.Hash{},
		Chunks:                     []*encoding.Chunk{chunk},
	}
	daBatch, err := codecv0.NewDABatch(batch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DA batch")
	}

	_, err = l2.scrollChain.ImportGenesisBatch(l2.defaultTransactor(), daBatch.Encode(), genesis.Root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to submit genesis batch transaction")
	}

	l2.lastCommittedBatch = &Batch{
		Batch:             batch,
		LastL2BlockNumber: genesis.Number.Uint64(),
		Hash:              daBatch.Hash(),
	}

	return l2.lastCommittedBatch, nil
}

func (l2 *L2) CommitBatch() (*Batch, error) {
	// put all blocks from the last committed to the last produced block into the new chunk
	var newChunk encoding.Chunk
	firstBlock := l2.lastCommittedBatch.LastL2BlockNumber + 1
	lastBlock := l2.lastProducedBlock.NumberU64()

	for i := firstBlock; i <= lastBlock; i++ {
		block, err := l2.backend.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get block %d", i)
		}

		newChunk.Blocks = append(newChunk.Blocks, &encoding.Block{
			Header:       block.Header(),
			Transactions: txsToTxsData(block.Transactions()),
			//WithdrawRoot:   common.BytesToHash(withdrawRoot),
			//RowConsumption: block.RowConsumption,
		})
	}

	// we only have one chunk in the batch
	newBatch := &encoding.Batch{
		Index: l2.lastCommittedBatch.Index + 1,
		//	TotalL1MessagePoppedBefore: dbChunks[0].TotalL1MessagesPoppedBefore,
		//	ParentBatchHash:            common.HexToHash(l2.lastCommittedBatch.Hash),
		Chunks: []*encoding.Chunk{&newChunk},
	}

	// now we can construct the DA batch, DA chunks and commit the batch
	daBatch, err := codecv3.NewDABatch(newBatch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DA batch")
	}

	encodedChunks := make([][]byte, len(newBatch.Chunks))
	for _, chunk := range newBatch.Chunks {
		// TODO: totalL1MessagesPoppedBefore
		daChunk, err := codecv3.NewDAChunk(chunk, 0)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create DA chunk")
		}
		encodedChunks = append(encodedChunks, daChunk.Encode())
	}

	blobDataProof, err := daBatch.BlobDataProofForPointEvaluation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create blob data proof")
	}

	skippedL1MessageBitmap, _, err := encoding.ConstructSkippedBitmap(newBatch.Index, newBatch.Chunks, newBatch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct skipped bitmap")
	}

	abi, err := contracts.ScrollChainMockFinalizeMetaData.GetAbi()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get abi")
	}
	calldata, err := abi.Pack("commitBatchWithBlobProof", daBatch.Version, l2.lastCommittedBatch.Hash[:], encodedChunks, skippedL1MessageBitmap, blobDataProof)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack calldata")
	}

	l2.lastCommittedBatch = &Batch{
		DABatch:           daBatch,
		Batch:             newBatch,
		LastL2BlockNumber: lastBlock,
		Hash:              daBatch.Hash(),
	}

	tx, err := l2.createCommitBatchTransaction(calldata, l2.lastCommittedBatch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create commit batch transaction for batch %s", l2.lastCommittedBatch.Hash)
	}

	err = l2.l1Sender(tx)
	if err != nil {
		fmt.Println("Failed to send commit batch transaction for batch", l2.lastCommittedBatch.Hash)
		return nil, errors.Wrapf(err, "failed to send commit batch transaction for batch %s", l2.lastCommittedBatch.Hash)
	}
	fmt.Printf("Committed batch: %d - %d chunks - blocks %d to %d\n", newBatch.Index, len(newBatch.Chunks), firstBlock, lastBlock)

	return l2.lastCommittedBatch, nil
}

func (l2 *L2) createCommitBatchTransaction(calldata []byte, batch *Batch) (*types.Transaction, error) {
	sidecar, err := makeSidecar(batch.DABatch.Blob())
	if err != nil {
		return nil, errors.Wrap(err, "failed to create blob sidecar")
	}

	_, err = l2.backend.PendingNonceAt(context.Background(), l2.keyManager.Address(defaultKeyAlias))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nonce")
	}

	txData := &types.BlobTx{
		ChainID:    uint256.MustFromBig(l2.backend.Blockchain().Config().ChainID),
		Nonce:      8,
		GasTipCap:  uint256.NewInt(100000000000),
		GasFeeCap:  uint256.NewInt(100000000000),
		Gas:        27060,
		To:         l2.scrollChainAddress,
		Data:       calldata,
		AccessList: nil,
		BlobFeeCap: uint256.NewInt(100),
		BlobHashes: sidecar.BlobHashes(),
		Sidecar:    sidecar,
	}

	signer := types.LatestSigner(l2.backend.Blockchain().Config())
	signedTx, err := types.SignTx(types.NewTx(txData), signer, l2.keyManager.Key(defaultKeyAlias))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to sign tx")
	}

	return signedTx, nil
}

func (l2 *L2) RevertUntilBatch(batchIndex uint64) {

}

func (l2 *L2) FinalizeBatch() {

}

func (l2 *L2) transactor(alias string) *bind.TransactOpts {
	return l2.keyManager.Transactor(alias, l2.backend.Blockchain().Config().ChainID)
}

func (l2 *L2) defaultTransactor() *bind.TransactOpts {
	return l2.transactor(defaultKeyAlias)
}

func txsToTxsData(txs types.Transactions) []*types.TransactionData {
	txsData := make([]*types.TransactionData, txs.Len())
	for i, tx := range txs {
		v, r, s := tx.RawSignatureValues()

		nonce := tx.Nonce()

		// We need QueueIndex in `NewBatchHeader`. However, `TransactionData`
		// does not have this field. Since `L1MessageTx` do not have a nonce,
		// we reuse this field for storing the queue index.
		if msg := tx.AsL1MessageTx(); msg != nil {
			nonce = msg.QueueIndex
		}

		txsData[i] = &types.TransactionData{
			Type:       tx.Type(),
			TxHash:     tx.Hash().String(),
			Nonce:      nonce,
			ChainId:    (*hexutil.Big)(tx.ChainId()),
			Gas:        tx.Gas(),
			GasPrice:   (*hexutil.Big)(tx.GasPrice()),
			GasTipCap:  (*hexutil.Big)(tx.GasTipCap()),
			GasFeeCap:  (*hexutil.Big)(tx.GasFeeCap()),
			To:         tx.To(),
			Value:      (*hexutil.Big)(tx.Value()),
			Data:       hexutil.Encode(tx.Data()),
			IsCreate:   tx.To() == nil,
			AccessList: tx.AccessList(),
			V:          (*hexutil.Big)(v),
			R:          (*hexutil.Big)(r),
			S:          (*hexutil.Big)(s),
		}
	}
	return txsData
}

func makeSidecar(blob *kzg4844.Blob) (*types.BlobTxSidecar, error) {
	if blob == nil {
		return nil, errors.New("blob cannot be nil")
	}

	blobs := []kzg4844.Blob{*blob}
	var commitments []kzg4844.Commitment
	var proofs []kzg4844.Proof

	for i := range blobs {
		c, err := kzg4844.BlobToCommitment(&blobs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to get blob commitment, err: %w", err)
		}

		p, err := kzg4844.ComputeBlobProof(&blobs[i], c)
		if err != nil {
			return nil, fmt.Errorf("failed to compute blob proof, err: %w", err)
		}

		commitments = append(commitments, c)
		proofs = append(proofs, p)
	}

	return &types.BlobTxSidecar{
		Blobs:       blobs,
		Commitments: commitments,
		Proofs:      proofs,
	}, nil
}
