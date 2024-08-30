package testsuite

import (
	"context"
	"fmt"
	"math/big"

	"github.com/cockroachdb/errors"
	"github.com/ethereum/go-ethereum"
	abiETH "github.com/ethereum/go-ethereum/accounts/abi"
	bindETH "github.com/ethereum/go-ethereum/accounts/abi/bind"
	commonETH "github.com/ethereum/go-ethereum/common"
	typesETH "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/ethereum/go-ethereum/node"
	paramsETH "github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/testsuite/contracts"
)

type L1 struct {
	keyManager *KeyManager
	backend    *simulated.Backend
	client     simulated.Client
	config     *paramsETH.ChainConfig

	scrollChain        *contracts.ScrollChainMockFinalize
	scrollChainAddress commonETH.Address
	scrollChainABI     *abiETH.ABI

	l2GasPriceOracle        *contracts.L2GasPriceOracle
	l2GasPriceOracleAddress commonETH.Address

	l1MessageQueue        *contracts.L1MessageQueue
	l1MessageQueueAddress commonETH.Address
}

func NewL1(km *KeyManager) (*L1, error) {
	l1 := &L1{
		keyManager: km,
	}

	balance := 100 * params.Ether
	gAlloc := typesETH.GenesisAlloc{
		km.L1Address(defaultKeyAlias): {Balance: new(big.Int).SetUint64(uint64(balance))},
	}

	l1.backend = simulated.NewBackend(gAlloc, func(nodeConf *node.Config, ethConf *ethconfig.Config) {
		l1.config = ethConf.Genesis.Config
	})
	l1.client = l1.backend.Client()

	fmt.Println("Started simulated L1 with following accounts:")
	for address, genesisAccount := range gAlloc {
		fmt.Printf("\tAddress: %s, %d\n", address, genesisAccount.Balance)
	}

	// Commit the genesis block.
	l1.backend.Commit()

	err := l1.setupContracts()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to setup L1 contracts")
	}

	return l1, nil
}

func (l1 *L1) setupContracts() error {
	// _messageQueue can't be the empty address here, however, we need to deploy ScrollChain first to be able to specify
	// the correct address when deploying L1MessageQueue
	scrollChainAddress, _, scrollChain, err := contracts.DeployScrollChainMockFinalize(l1.defaultTransactor(), l1.client, 5, commonETH.Address{1}, l1.keyManager.L1Address("verifier"))
	if err != nil {
		return errors.Wrap(err, "failed to deploy ScrollChain")
	}
	l1.scrollChain = scrollChain
	l1.scrollChainAddress = scrollChainAddress

	scrollChainABI, err := contracts.ScrollChainMockFinalizeMetaData.GetAbi()
	if err != nil {
		return errors.Wrap(err, "failed to get abi")
	}
	l1.scrollChainABI = scrollChainABI
	fmt.Println("Deployed ScrollChain:", scrollChainAddress)

	l2GasPriceOracleAddress, _, l2GasPriceOracle, err := contracts.DeployL2GasPriceOracle(l1.defaultTransactor(), l1.client)
	if err != nil {
		return errors.Wrap(err, "failed to deploy L2GasPriceOracle")
	}
	l1.l2GasPriceOracle = l2GasPriceOracle
	l1.l2GasPriceOracleAddress = l2GasPriceOracleAddress
	fmt.Println("Deployed L2GasPriceOracle:", l2GasPriceOracleAddress)

	l1.CommitBlock()

	// we don't deploy enforcedTxGateway
	l1MessageQueueAddress, tx, l1MessageQueue, err := contracts.DeployL1MessageQueue(l1.defaultTransactor(), l1.client, l1.keyManager.L1Address(defaultKeyAlias), scrollChainAddress, commonETH.Address{1})
	if err != nil {
		return errors.Wrap(err, "failed to deploy L1MessageQueue")
	}
	l1.l1MessageQueue = l1MessageQueue
	l1.l1MessageQueueAddress = l1MessageQueueAddress
	fmt.Println("Deployed L1MessageQueue:", l1MessageQueueAddress)

	l1.CommitBlock()

	// first 3 parameters are deprecated and not used
	tx, err = l1MessageQueue.Initialize(l1.defaultTransactor(), commonETH.Address{}, commonETH.Address{}, commonETH.Address{}, l2GasPriceOracleAddress, new(big.Int).SetUint64(10000000000))
	if err != nil {
		return errors.Wrap(err, "failed to initialize L1MessageQueue")
	}
	if err = l1.simulateTxCall(tx); err != nil {
		return errors.Wrapf(err, "failed to simulate tx call")
	}
	l1.CommitBlock()

	tx, err = l1.scrollChain.Initialize(l1.defaultTransactor(), l1MessageQueueAddress, l1.keyManager.L1Address("verifier"), new(big.Int).SetUint64(1000))
	if err != nil {
		return errors.Wrap(err, "failed to initialize ScrollChain")
	}
	if err = l1.simulateTxCall(tx); err != nil {
		return errors.Wrapf(err, "failed to simulate tx call")
	}
	l1.CommitBlock()

	if err = l1.addSequencer(); err != nil {
		return errors.Wrap(err, "failed to add sequencer")
	}
	l1.CommitBlock()

	return nil
}

func (l1 *L1) LatestSigner() typesETH.Signer {
	return typesETH.LatestSigner(l1.config)
}

func (l1 *L1) getReceipt(tx *typesETH.Transaction) *typesETH.Receipt {
	receipt, err := l1.client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		panic(errors.Wrapf(err, "failed to get receipt for tx %s", tx.Hash().String()))
	}

	if receipt.Status != typesETH.ReceiptStatusSuccessful {
		fmt.Println("tx failed with status", receipt.Status)
		fmt.Println(receipt.Logs)
		//panic(errors.Errorf("tx %s failed with status %d", tx.Hash().String(), receipt.Status))
	}

	return receipt
}

func (l1 *L1) simulateTxCall(tx *typesETH.Transaction) error {
	// simulate call to contract before sending tx
	msg := ethereum.CallMsg{
		From:          l1.keyManager.L1Address(defaultKeyAlias),
		To:            tx.To(),
		GasFeeCap:     tx.GasFeeCap(),
		GasTipCap:     tx.GasTipCap(),
		Value:         tx.Value(),
		Data:          tx.Data(),
		AccessList:    tx.AccessList(),
		BlobGasFeeCap: tx.BlobGasFeeCap(),
		BlobHashes:    tx.BlobHashes(),
	}

	_, err := l1.client.PendingCallContract(context.Background(), msg)
	if err != nil {
		// hack to access the error data (function selector) and print the corresponding error name to ease debugging of contract calls.
		// necessary as the rpc.jsonError is not exported and as such we can't check with errors.Is or errors.As
		if dataErr, ok := err.(rpc.DataError); ok {
			if errData, ok := dataErr.ErrorData().(string); ok {
				return errors.Wrapf(err, "failed to call contract for tx %s with contract error %s (%s)", tx.Hash().String(), l1.scrollChainErrorName(errData), errData)
			}
		}

		return errors.Wrapf(err, "failed to call contract for tx %s", tx.Hash().String())
	}

	return nil
}

// scrollChainErrorName is a helper function to get the name of the error from the ScrollChain contract
// by the function selector that is returned if the call to the contract fails
func (l1 *L1) scrollChainErrorName(targetFunctionSelector string) string {
	for _, e := range l1.ScrollChainABI().Errors {
		// the first 4 bytes of the error ID are the function selector: https://docs.soliditylang.org/en/v0.4.24/abi-spec.html#function-selector
		fs := hexutil.Encode(e.ID[:4])
		if targetFunctionSelector == fs {
			return e.Name
		}
	}

	return ""
}

func (l1 *L1) addSequencer() error {
	tx, err := l1.scrollChain.AddSequencer(l1.defaultTransactor(), l1.keyManager.L1Address(defaultKeyAlias))
	if err != nil {
		return errors.Wrap(err, "failed to add sequencer")
	}
	if err = l1.simulateTxCall(tx); err != nil {
		return errors.Wrapf(err, "failed to simulate tx call")
	}

	return nil
}

func (l1 *L1) SendL1ToL2Message(toAlias string, data []byte, commit bool) (*typesETH.Transaction, error) {
	tx, err := l1.l1MessageQueue.AppendCrossDomainMessage(l1.defaultTransactor(), l1.keyManager.L1Address(toAlias), big.NewInt(10000), data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to append cross domain message")
	}

	if commit {
		l1.CommitBlock()
	}

	return tx, nil
}

func (l1 *L1) FilterL1MessageQueueTransactions(start uint64, end uint64) ([]*contracts.L1MessageQueueQueueTransaction, error) {
	var endFilter *uint64
	if end > 0 {
		endFilter = &end
	}

	opts := &bindETH.FilterOpts{
		Start:   start,
		End:     endFilter,
		Context: context.Background(),
	}

	iter, err := l1.l1MessageQueue.FilterQueueTransaction(opts, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to filter queue transaction")
	}

	var events []*contracts.L1MessageQueueQueueTransaction
	for iter.Next() {
		events = append(events, iter.Event)
	}

	return events, nil
}

func (l1 *L1) CommitBlock() *typesETH.Block {
	hash := l1.backend.Commit()

	var err error
	block, err := l1.client.BlockByHash(context.Background(), hash)

	// this should never happen as we just committed the block
	if err != nil {
		panic(errors.Wrapf(err, "failed to get block by hash %s", hash.String()))
	}

	return block
}

func (l1 *L1) ScrollChain() *contracts.ScrollChainMockFinalize {
	return l1.scrollChain
}

func (l1 *L1) ScrollChainAddress() commonETH.Address {
	return l1.scrollChainAddress
}

func (l1 *L1) ScrollChainABI() *abiETH.ABI {
	return l1.scrollChainABI
}

func (l1 *L1) L1MessageQueue() *contracts.L1MessageQueue {
	return l1.l1MessageQueue
}
func (l1 *L1) L1MessageQueueAddress() common.Address {
	return ethAddressToAddress(l1.l1MessageQueueAddress)
}
func (l1 *L1) transactor(alias string) *bindETH.TransactOpts {
	return l1.keyManager.L1Transactor(alias, l1.ChainID())
}

func (l1 *L1) defaultTransactor() *bindETH.TransactOpts {
	return l1.transactor(defaultKeyAlias)
}

func (l1 *L1) SendTransaction(tx *typesETH.Transaction) error {
	// Remove the sidecar from the tx before applying it to the backend
	// TODO: store sidecar for later retrieval
	if tx.Type() == types.BlobTxType {

	}

	err := l1.client.SendTransaction(context.Background(), tx)
	if err != nil {
		return errors.Wrap(err, "failed to send tx")
	}

	return nil
}

func (l1 *L1) ChainID() *big.Int {
	return l1.config.ChainID
}

func addressToETHAddress(addr common.Address) commonETH.Address {
	return (commonETH.Address)(addr)
}

func ethAddressToAddress(addr commonETH.Address) common.Address {
	return (common.Address)(addr)
}
