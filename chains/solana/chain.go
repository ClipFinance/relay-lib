package solana

import (
	"context"
	"sync"

	"github.com/ClipFinance/relay-lib/chainmanager"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/ClipFinance/relay-lib/connectionmonitor"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	sol "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// solana represents the base Solana chain implementation
type solana struct {
	config *types.ChainConfig
	logger *logrus.Logger

	// Protected fields with their own mutexes
	clientMutex sync.RWMutex
	client      *rpc.Client

	signerMutex sync.RWMutex
	signer      sol.PrivateKey

	eventHandlerMutex sync.RWMutex
	eventHandler      interface{} // Replace with actual Solana event handler type

	monitorMutex sync.RWMutex
	monitor      connectionmonitor.ConnectionMonitor

	// Additional Solana-specific fields
	solverAddressMutex sync.RWMutex
	solverAddress      string
}

// NewSolanaChain creates a new Solana chain implementation
func NewSolanaChain(ctx context.Context, config *types.ChainConfig, logger *logrus.Logger) (types.Chain, error) {
	// Create RPC client
	client := rpc.New(config.RpcUrl)

	chain := &solana{
		config: config,
		logger: logger,
		client: client,
	}

	if config.SolverAddress != "" {
		chain.solverAddress = config.SolverAddress
	}

	if err := chain.initMonitor(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to init connection connection monitor")
	}

	builder := chainmanager.NewChainBuilder(config)
	builder.WithGasEstimator(chain)

	if config.PrivateKey != "" {
		signer, err := sol.PrivateKeyFromBase58(config.PrivateKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create signer")
		}

		chain.signerMutex.Lock()
		chain.signer = signer
		chain.signerMutex.Unlock()

		chain.solverAddressMutex.Lock()
		chain.solverAddress = getSolanaAddress(signer) // Replace with actual method
		chain.solverAddressMutex.Unlock()

		builder.WithTransactionSender(chain)
	}

	builder.WithTransactionWatcher(chain)
	builder.WithEventHandler(chain)
	builder.WithBalanceProvider(chain)

	return builder.Build(), nil
}

// Close should be called when chain is no longer needed
func (s *solana) Close() {
	s.monitorMutex.Lock()
	if s.monitor != nil {
		s.monitor.Stop()
	}
	s.monitorMutex.Unlock()

	s.clientMutex.Lock()
	if s.client != nil {
		// Add cleanup for Solana client if needed
		s.client = nil
	}
	s.clientMutex.Unlock()

	s.eventHandlerMutex.Lock()
	if s.eventHandler != nil {
		// Add cleanup for event handler if needed
		s.eventHandler = nil
	}
	s.eventHandlerMutex.Unlock()
}

func getSolanaAddress(signer sol.PrivateKey) string {
	return signer.PublicKey().String()
}
