package testsuite

import (
	"context"
	"fmt"
	"math/big"

	"github.com/cockroachdb/errors"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/holiman/uint256"
	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/da-codec/encoding/codecv0"
	"github.com/scroll-tech/da-codec/encoding/codecv3"

	typesETH "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/params"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind/backends"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
)

type L2 struct {
	keyManager *KeyManager
	backend    *backends.SimulatedBackend

	l1 *L1

	batches            map[uint64]*Batch
	lastProducedBlock  *types.Block
	lastCommittedBatch *Batch
	lastFinalizedBatch *Batch
}

func NewL2(km *KeyManager, l1 *L1) (*L2, error) {
	gAlloc := core.GenesisAlloc{
		km.Address(defaultKeyAlias): {Balance: new(big.Int).SetUint64(1 * params.Ether)},
	}
	backend := backends.NewSimulatedBackend(gAlloc, 1000000000)

	fmt.Println("Started simulated L1 with following accounts:")
	for address, genesisAccount := range gAlloc {
		fmt.Printf("\tAddress: %s, %d\n", address, genesisAccount.Balance)
	}

	l2 := &L2{
		keyManager: km,
		backend:    backend,
		l1:         l1,
		batches:    make(map[uint64]*Batch),
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

	_, err = l2.l1.ScrollChain().ImportGenesisBatch(l2.l1.defaultTransactor(), daBatch.Encode(), genesis.Root)
	if err != nil {
		return nil, errors.Wrap(err, "failed to submit genesis batch transaction")
	}

	l2.lastCommittedBatch = &Batch{
		BatchHeaderBytes:  daBatch.Encode(),
		Hash:              daBatch.Hash(),
		LastL2BlockNumber: genesis.Number.Uint64(),
		Batch:             batch,
	}

	return l2.lastCommittedBatch, nil
}

func (l2 *L2) CommitBatch() (*Batch, error) {
	parentBatch := l2.lastCommittedBatch

	// put all blocks from the last committed to the last produced block into the new chunk
	var newChunk encoding.Chunk
	firstBlock := parentBatch.LastL2BlockNumber + 1
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
		Index:                      parentBatch.Index + 1,
		TotalL1MessagePoppedBefore: 0,
		ParentBatchHash:            parentBatch.Hash,
		Chunks:                     []*encoding.Chunk{&newChunk},
	}

	// now we can construct the DA batch, DA chunks and commit the batch
	daBatch, err := codecv3.NewDABatch(newBatch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DA batch")
	}

	encodedChunks := make([][]byte, len(newBatch.Chunks))
	for i, chunk := range newBatch.Chunks {
		// TODO: totalL1MessagesPoppedBefore
		daChunk, err := codecv3.NewDAChunk(chunk, 0)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create DA chunk")
		}
		encodedChunks[i] = daChunk.Encode()
	}

	blobDataProof, err := daBatch.BlobDataProofForPointEvaluation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create blob data proof")
	}

	skippedL1MessageBitmap, _, err := encoding.ConstructSkippedBitmap(newBatch.Index, newBatch.Chunks, newBatch.TotalL1MessagePoppedBefore)
	if err != nil {
		return nil, errors.Wrap(err, "failed to construct skipped bitmap")
	}

	calldata, err := l2.l1.ScrollChainABI().Pack("commitBatchWithBlobProof", daBatch.Version, parentBatch.BatchHeaderBytes, encodedChunks, skippedL1MessageBitmap, blobDataProof)
	if err != nil {
		return nil, errors.Wrap(err, "failed to pack calldata")
	}

	tx, err := l2.createCommitBatchTransaction(calldata, (*kzg4844.Blob)(daBatch.Blob()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create commit batch transaction for batch %s", daBatch.Hash())
	}

	// simulate call to contract before sending tx
	if err = l2.l1.simulateTxCall(tx); err != nil {
		return nil, errors.Wrapf(err, "failed to call contract for batch %s", daBatch.Hash())
	}

	if err = l2.l1.SendTransaction(tx); err != nil {
		return nil, errors.Wrapf(err, "failed to send commit batch transaction for batch %s", daBatch.Hash())
	}

	fmt.Printf("Committed batch: %d with hash %s - %d chunks - blocks %d to %d in tx %s\n", newBatch.Index, daBatch.Hash(), len(newBatch.Chunks), firstBlock, lastBlock, tx.Hash())

	l2.lastCommittedBatch = &Batch{
		BatchHeaderBytes:  daBatch.Encode(),
		Hash:              daBatch.Hash(),
		LastL2BlockNumber: lastBlock,
		Batch:             newBatch,
	}

	return l2.lastCommittedBatch, nil
}

func (l2 *L2) createCommitBatchTransaction(calldata []byte, blob *kzg4844.Blob) (*typesETH.Transaction, error) {
	sidecar, err := makeSidecar(blob)
	if err != nil {
		return nil, errors.New("failed to create blob sidecar")
	}

	nonce, err := l2.l1.client.PendingNonceAt(context.Background(), l2.keyManager.L1Address(defaultKeyAlias))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nonce")
	}

	gasPrice, err := l2.l1.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get suggested gas price")
	}

	fmt.Println("Suggested gas price", gasPrice)
	// TODO: whenever sending txs, double check gas price and gas limit
	txData := &typesETH.BlobTx{
		ChainID:    uint256.MustFromBig(l2.l1.ChainID()),
		Nonce:      nonce,
		GasTipCap:  uint256.NewInt(gasPrice.Uint64()),
		GasFeeCap:  uint256.NewInt(gasPrice.Uint64()),
		Gas:        10000000,
		To:         l2.l1.ScrollChainAddress(),
		Data:       calldata,
		AccessList: nil,
		BlobFeeCap: uint256.NewInt(100000),
		BlobHashes: sidecar.BlobHashes(),
		Sidecar:    sidecar,
	}

	signedTx, err := typesETH.SignTx(typesETH.NewTx(txData), l2.l1.LatestSigner(), l2.keyManager.Key(defaultKeyAlias))
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

func makeSidecar(blob *kzg4844.Blob) (*typesETH.BlobTxSidecar, error) {
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

	return &typesETH.BlobTxSidecar{
		Blobs:       blobs,
		Commitments: commitments,
		Proofs:      proofs,
	}, nil
}
