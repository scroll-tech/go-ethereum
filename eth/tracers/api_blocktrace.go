package tracers

import (
	"context"
	"errors"

	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/consensus/misc"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rpc"
)

var errNoScrollTracerWrapper = errors.New("no ScrollTracerWrapper")

type TraceBlock interface {
	GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error)
	GetTxBlockTraceOnTopOfBlock(ctx context.Context, tx *types.Transaction, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (*types.BlockTrace, error)
}

type scrollTracerWrapper interface {
	CreateTraceEnvAndGetBlockTrace(*params.ChainConfig, *vm.LogConfig, core.ChainContext, consensus.Engine, ethdb.Database, *state.StateDB, *types.Block, *types.Block, bool) (*types.BlockTrace, error)
}

// GetBlockTraceByNumberOrHash replays the block and returns the structured BlockTrace by hash or number.
func (api *API) GetBlockTraceByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (trace *types.BlockTrace, err error) {
	if api.scrollTracerWrapper == nil {
		return nil, errNoScrollTracerWrapper
	}

	var block *types.Block
	if number, ok := blockNrOrHash.Number(); ok {
		block, err = api.blockByNumber(ctx, number)
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}

	return api.createTraceEnvAndGetBlockTrace(ctx, config, block)
}

func (api *API) GetTxBlockTraceOnTopOfBlock(ctx context.Context, tx *types.Transaction, blockNrOrHash rpc.BlockNumberOrHash, config *TraceConfig) (*types.BlockTrace, error) {
	if api.scrollTracerWrapper == nil {
		return nil, errNoScrollTracerWrapper
	}

	// Try to retrieve the specified block
	var (
		err   error
		block *types.Block
	)
	if number, ok := blockNrOrHash.Number(); ok {
		block, err = api.blockByNumber(ctx, number)
	} else if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else {
		return nil, errors.New("invalid arguments; neither block number nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}

	block = types.NewBlockWithHeader(block.Header()).WithBody([]*types.Transaction{tx}, nil)

	return api.createTraceEnvAndGetBlockTrace(ctx, config, block)
}

// Make trace environment for current block, and then get the trace for the block.
func (api *API) createTraceEnvAndGetBlockTrace(ctx context.Context, config *TraceConfig, block *types.Block) (*types.BlockTrace, error) {
	if config == nil {
		config = &TraceConfig{
			LogConfig: &vm.LogConfig{
				DisableStorage:   true,
				DisableStack:     true,
				EnableMemory:     false,
				EnableReturnData: true,
			},
		}
	} else if config.Tracer != nil {
		config.Tracer = nil
		log.Warn("Tracer params is unsupported")
	}

	parent, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash())
	if err != nil {
		return nil, err
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	statedb, err := api.backend.StateAtBlock(ctx, parent, reexec, nil, true, true)
	if err != nil {
		return nil, err
	}

	chaindb := api.backend.ChainDb()
	// create a copy of api.backend.ChainConfig to modify
	chainConfig := new(params.ChainConfig)
	*chainConfig = *api.backend.ChainConfig()
	if config != nil && config.Overrides != nil {
		// In future we can add more fork related logics here
		// like upstream: https://github.com/ethereum/go-ethereum/pull/26655
		if curie := config.Overrides.CurieBlock; curie != nil {
			chainConfig.CurieBlock = curie
			if !api.backend.ChainConfig().IsCurie(block.Number()) && block.Number().Cmp(curie) > 0 {
				// set non zero values for these slots
				misc.ApplyCurieHardFork(statedb)
				statedb.IntermediateRoot(true)
			}
		}
		log.Info("chainConfig overrided by traceConfig.Overrides", "chainConfig", chainConfig, "config.Overrides", config.Overrides)
	}
	return api.scrollTracerWrapper.CreateTraceEnvAndGetBlockTrace(chainConfig, config.LogConfig, api.chainContext(ctx), api.backend.Engine(), chaindb, statedb, parent, block, true)
}
