package chainmanager

import (
	"context"
	"sync"

	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/sirupsen/logrus"
)

type blockchainRegistry struct {
	logger      *logrus.Logger
	chains      map[uint64]types.Chain
	chainsMutex sync.RWMutex
	factory     interface {
		CreateChain(context.Context, *types.ChainConfig, *logrus.Logger) (types.Chain, error)
	}
	factoryMutex sync.RWMutex
}

func NewChainRegistry(factory interface {
	CreateChain(context.Context, *types.ChainConfig, *logrus.Logger) (types.Chain, error)
}, logger *logrus.Logger) types.ChainRegistry {
	return &blockchainRegistry{
		chains:  make(map[uint64]types.Chain),
		factory: factory,
		logger:  logger,
	}
}

func (r *blockchainRegistry) Add(ctx context.Context, config *types.ChainConfig) error {
	// Lock factory for reading to prevent changes during chain creation.
	r.factoryMutex.RLock()
	chain, err := r.factory.CreateChain(ctx, config, r.logger)
	r.factoryMutex.RUnlock()

	if err != nil {
		return err
	}

	// Lock chains map for writing
	r.chainsMutex.Lock()
	r.chains[config.ChainID] = chain
	r.chainsMutex.Unlock()

	return nil
}

func (r *blockchainRegistry) Get(chainID uint64) types.Chain {
	r.chainsMutex.RLock()
	chain := r.chains[chainID]
	r.chainsMutex.RUnlock()
	return chain
}

func (r *blockchainRegistry) Remove(chainID uint64) {
	r.chainsMutex.Lock()
	delete(r.chains, chainID)
	r.chainsMutex.Unlock()
}
