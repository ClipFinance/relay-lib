package chains

import (
	"github.com/ClipFinance/relay-lib/chains/evm"
	"github.com/ClipFinance/relay-lib/chains/solana"
	commontypes "github.com/ClipFinance/relay-lib/common/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
)

// ChainConstructor represents a function that constructs a new chain instance.
//
// Parameters:
// - config: the configuration for the chain.
// - logger: the logger for logging purposes.
//
// Returns:
// - commontypes.Chain: the constructed chain instance.
// - error: an error if the chain construction fails.
type ChainConstructor func(config *commontypes.ChainConfig, logger *logrus.Logger) (commontypes.Chain, error)

// ChainFactory defines the interface for chain creation.
type ChainFactory interface {
	// RegisterConstructor registers a new chain constructor for a given chain type.
	//
	// Parameters:
	// - chainType: the type of the chain to register.
	// - constructor: the constructor function for the chain type.
	RegisterConstructor(chainType string, constructor ChainConstructor)

	// CreateChain creates a new chain instance based on the configuration.
	//
	// Parameters:
	// - config: the configuration for the chain.
	// - logger: the logger for logging purposes.
	//
	// Returns:
	// - commontypes.Chain: the created chain instance.
	// - error: an error if the chain creation fails.
	CreateChain(config *commontypes.ChainConfig, logger *logrus.Logger) (commontypes.Chain, error)
}

type chainFactory struct {
	// constructors stores the mapping of chain types to their constructors.
	constructors map[string]ChainConstructor
	// constructorsMutex protects access to the constructors map.
	constructorsMutex sync.RWMutex
}

// NewChainFactory creates a new instance of the chain factory.
//
// Returns:
// - ChainFactory: the new chain factory instance.
func NewChainFactory() ChainFactory {
	factory := &chainFactory{
		constructors: make(map[string]ChainConstructor),
	}

	// Initialize with default constructors.
	factory.registerConstructors()

	return factory
}

// RegisterConstructor registers a new chain constructor.
//
// Parameters:
// - chainType: the type of the chain to register.
// - constructor: the constructor function for the chain type.
func (f *chainFactory) RegisterConstructor(chainType string, constructor ChainConstructor) {
	f.constructorsMutex.Lock()
	defer f.constructorsMutex.Unlock()

	f.constructors[chainType] = constructor
}

// CreateChain creates a new chain instance based on the configuration.
//
// Parameters:
// - config: the configuration for the chain.
// - logger: the logger for logging purposes.
//
// Returns:
// - commontypes.Chain: the created chain instance.
// - error: an error if the chain creation fails.
func (f *chainFactory) CreateChain(config *commontypes.ChainConfig, logger *logrus.Logger) (commontypes.Chain, error) {
	f.constructorsMutex.RLock()
	constructor, exists := f.constructors[config.ChainType]
	f.constructorsMutex.RUnlock()

	if !exists {
		return nil, errors.New("invalid chain type")

	}

	return constructor(config, logger)
}

// registerConstructors registers the blockchain constructors for the chain factory instance.
func (f *chainFactory) registerConstructors() {
	// Register EVM chain constructor with the factory.
	f.RegisterConstructor(EVM.String(), func(config *commontypes.ChainConfig, logger *logrus.Logger) (commontypes.Chain, error) {
		return evm.NewEvmChain(config, logger)
	})

	// Register Solana chain constructor with the factory.
	f.RegisterConstructor(SOLANA.String(), func(config *commontypes.ChainConfig, logger *logrus.Logger) (commontypes.Chain, error) {
		return solana.NewSolanaChain(config, logger)
	})
}
