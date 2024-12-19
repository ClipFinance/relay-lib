package evm

import (
	"context"
	"sync"
	"time"

	"github.com/ClipFinance/relay-lib/chainmanager"
	"github.com/ClipFinance/relay-lib/chains/evm/handler"
	"github.com/ClipFinance/relay-lib/chains/evm/signer"
	"github.com/ClipFinance/relay-lib/common/types"
	"github.com/ClipFinance/relay-lib/connectionmonitor"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// TxTypeLegacy represents the legacy transaction type.
	TxTypeLegacy = 0
	// TxTypeEIP1559 represents the EIP-1559 transaction type.
	TxTypeEIP1559 = 2
	// waitTimeout is the timeout duration for waiting operations.
	waitTimeout = 30 * time.Second
)

// evm represents the base EVM chain implementation.
type evm struct {
	config             *types.ChainConfig // Chain configuration.
	logger             *logrus.Logger     // Logger for logging events.
	solverAddress      common.Address     // Solver address.
	solverAddressMutex sync.RWMutex       // Mutex for solver address.

	// Protected fields with their own mutexes.
	clientMutex sync.RWMutex      // Mutex for client.
	client      *ethclient.Client // Ethereum client.

	signerMutex sync.RWMutex  // Mutex for signer.
	signer      signer.Signer // Signer for signing transactions.

	eventHandlerMutex sync.RWMutex          // Mutex for event handler.
	eventHandler      *handler.EventHandler // Event handler for handling chain events.

	monitorMutex sync.RWMutex                        // Mutex for connection monitor.
	monitor      connectionmonitor.ConnectionMonitor // Connection monitor.
}

// NewEvmChain creates a new EVM chain implementation.
//
// Parameters:
// - ctx: the context for managing the request.
// - config: the chain configuration.
// - logger: the logger for logging events.
//
// Returns:
// - types.Chain: a new EVM chain instance.
// - error: an error if any issue occurs during creation.
func NewEvmChain(ctx context.Context, config *types.ChainConfig, logger *logrus.Logger) (types.Chain, error) {
	client, err := ethclient.Dial(config.RpcUrl)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}

	chain := &evm{
		config: config,
		logger: logger,
		client: client,
	}

	if config.SolverAddress != "" {
		chain.solverAddress = common.HexToAddress(config.SolverAddress)
	}

	if err := chain.initMonitor(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to init connection monitor")
	}

	builder := chainmanager.NewChainBuilder(config)
	builder.WithGasEstimator(chain)

	if config.PrivateKey != "" {
		privKey, err := crypto.HexToECDSA(config.PrivateKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse private key")
		}

		signer, err := signer.NewSigner(privKey)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create signer")
		}

		chain.signerMutex.Lock()
		chain.signer = signer
		chain.signerMutex.Unlock()

		chain.solverAddress = signer.Address()
		builder.WithTransactionSender(chain)
	}

	builder.WithTransactionWatcher(chain)
	builder.WithEventHandler(chain)
	builder.WithBalanceProvider(chain)

	return builder.Build(), nil
}

// Close should be called when the chain is no longer needed.
// It stops the connection monitor, closes the client, and stops the event handler.
func (e *evm) Close() {
	e.monitorMutex.Lock()
	if e.monitor != nil {
		e.monitor.Stop()
	}
	e.monitorMutex.Unlock()

	e.clientMutex.Lock()
	if e.client != nil {
		e.client.Close()
		e.client = nil
	}
	e.clientMutex.Unlock()

	e.eventHandlerMutex.Lock()
	if e.eventHandler != nil {
		e.eventHandler.Stop()
		e.eventHandler = nil
	}
	e.eventHandlerMutex.Unlock()
}

// GetClient returns the Ethereum client.
//
// Returns:
// - *ethclient.Client: the Ethereum client.
func (e *evm) GetClient() *ethclient.Client {
	e.clientMutex.RLock()
	defer e.clientMutex.RUnlock()
	return e.client
}
