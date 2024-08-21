package simulated

import (
	"context"
	"fmt"
	"math/big"

	"github.com/cockroachdb/errors"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind/backends"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/simulated/contracts"
)

type L1 struct {
	keyManager *KeyManager
	backend    *backends.SimulatedBackend

	scrollChain      *contracts.ScrollChainMockFinalize
	l2GasPriceOracle *contracts.L2GasPriceOracle
	l1MessageQueue   *contracts.L1MessageQueue
}

func NewL1(km *KeyManager) (*L1, error) {
	gAlloc := core.GenesisAlloc{
		km.Address(defaultKeyAlias): {Balance: new(big.Int).SetUint64(1 * params.Ether)},
	}
	backend := backends.NewSimulatedBackend(gAlloc, 1000000000)

	fmt.Println("Started simulated L1 with following accounts:")
	for address, genesisAccount := range gAlloc {
		fmt.Printf("\tAddress: %s, %d\n", address, genesisAccount.Balance)
	}

	l1 := &L1{
		keyManager: km,
		backend:    backend,
	}

	err := l1.setupContracts()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to setup L1 contracts")
	}

	return l1, nil
}

func (l1 *L1) setupContracts() error {
	// _messageQueue can't be the empty address here, however, we need to deploy ScrollChain first to be able to specify
	// the correct address when deploying L1MessageQueue
	scrollChainAddress, _, scrollChain, err := contracts.DeployScrollChainMockFinalize(l1.defaultTransactor(), l1.backend, 5, common.Address{1}, l1.keyManager.Address("verifier"))
	if err != nil {
		return errors.Wrap(err, "failed to deploy ScrollChain")
	}
	l1.scrollChain = scrollChain
	fmt.Println("Deployed ScrollChain:", scrollChainAddress)

	l2GasPriceOracleAddress, _, l2GasPriceOracle, err := contracts.DeployL2GasPriceOracle(l1.defaultTransactor(), l1.backend)
	if err != nil {
		return errors.Wrap(err, "failed to deploy L2GasPriceOracle")
	}
	l1.l2GasPriceOracle = l2GasPriceOracle
	fmt.Println("Deployed L2GasPriceOracle:", l2GasPriceOracleAddress)

	// we don't deploy enforcedTxGateway
	l1MessageQueueAddress, _, l1MessageQueue, err := contracts.DeployL1MessageQueue(l1.defaultTransactor(), l1.backend, l1.keyManager.Address(defaultKeyAlias), scrollChainAddress, common.Address{1})
	if err != nil {
		return errors.Wrap(err, "failed to deploy L1MessageQueue")
	}
	l1.l1MessageQueue = l1MessageQueue
	fmt.Println("Deployed L1MessageQueue:", l1MessageQueueAddress)

	// first 3 parameters are deprecated and not used
	_, err = l1MessageQueue.Initialize(l1.defaultTransactor(), common.Address{}, common.Address{}, common.Address{}, l2GasPriceOracleAddress, new(big.Int).SetUint64(100000000))
	if err != nil {
		return errors.Wrap(err, "failed to initialize L1MessageQueue")
	}

	_, err = l1.scrollChain.Initialize(l1.defaultTransactor(), l1MessageQueueAddress, l1.keyManager.Address("verifier"), new(big.Int).SetUint64(1000))
	if err != nil {
		return errors.Wrap(err, "failed to initialize ScrollChain")
	}

	l1.CommitBlock()

	return nil
}

func (l1 *L1) SendL1ToL2Message(toAlias string, data []byte, commit bool) (*types.Transaction, error) {
	tx, err := l1.l1MessageQueue.AppendCrossDomainMessage(l1.defaultTransactor(), l1.keyManager.Address(toAlias), big.NewInt(10000000), data)
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

	opts := &bind.FilterOpts{
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

func (l1 *L1) CommitBlock() *types.Block {
	hash := l1.backend.Commit()

	var err error
	block, err := l1.backend.BlockByHash(context.Background(), hash)

	// this should never happen as we just committed the block
	if err != nil {
		panic(err)
	}

	return block
}

func (l1 *L1) ScrollChain() *contracts.ScrollChainMockFinalize {
	return l1.scrollChain
}

func (l1 *L1) L1MessageQueue() *contracts.L1MessageQueue {
	return l1.l1MessageQueue
}

func (l1 *L1) transactor(alias string) *bind.TransactOpts {
	return l1.keyManager.Transactor(alias, l1.backend.Blockchain().Config().ChainID)
}

func (l1 *L1) defaultTransactor() *bind.TransactOpts {
	return l1.transactor(defaultKeyAlias)
}

func (l1 *L1) SendTransaction(tx *types.Transaction) error {
	err := l1.backend.SendTransaction(context.Background(), tx)
	if err != nil {
		return errors.Wrap(err, "failed to send tx")
	}

	return nil
}
