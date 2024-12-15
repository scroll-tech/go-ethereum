package system_contract

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/sync_service"
)

const (
	defaultSyncInterval = 10
)

// SystemContract
type SystemContract struct {
	config *params.SystemContractConfig // Consensus engine configuration parameters
	client sync_service.EthClient

	signerAddressL1 common.Address // Address of the signer stored in L1 System Contract

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer and proposals fields

	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a SystemContract consensus engine with the initial
// signers set to the ones provided by the user.
func New(ctx context.Context, config *params.SystemContractConfig, client sync_service.EthClient) *SystemContract {
	blockNumber := big.NewInt(-1) // todo: get block number from L1BlocksContract (l1 block hash relay) or other source (depending on exact design)
	ctx, cancel := context.WithCancel(ctx)
	address, err := client.StorageAt(ctx, config.SystemContractAddress, config.SystemContractSlot, blockNumber)
	if err != nil {
		log.Error("failed to get signer address from L1 System Contract", "err", err)
	}
	systemContract := &SystemContract{
		config:          config,
		client:          client,
		signerAddressL1: common.BytesToAddress(address),

		ctx:    ctx,
		cancel: cancel,
	}
	systemContract.Start()
	return systemContract
}

// Authorize injects a private key into the consensus engine to mint new blocks
// with.
func (s *SystemContract) Authorize(signer common.Address, signFn SignerFn) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.signer = signer
	s.signFn = signFn
}

func (s *SystemContract) Start() {
	log.Info("starting SystemContract")
	go func() {
		syncTicker := time.NewTicker(defaultSyncInterval)
		defer syncTicker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
			}
			select {
			case <-s.ctx.Done():
				return
			case <-syncTicker.C:
				blockNumber := big.NewInt(-1) // todo: get block number from L1BlocksContract (l1 block hash relay) or other source (depending on exact design)

				address, err := s.client.StorageAt(s.ctx, s.config.SystemContractAddress, s.config.SystemContractSlot, blockNumber)
				if err != nil {
					log.Error("failed to get signer address from L1 System Contract", "err", err)
				}
				s.lock.Lock()
				s.signerAddressL1 = common.BytesToAddress(address)
				s.lock.Unlock()
			}
		}
	}()
}

// Close implements consensus.Engine.
func (s *SystemContract) Close() error {
	s.cancel()
	return nil
}