package solana

import (
	"context"
	"errors"

	"github.com/ClipFinance/relay-lib/connectionmonitor"
	"github.com/gagliardetto/solana-go/rpc"
)

// solanaConnectionManager implements connectionmonitor.BlockchainClient interface
type solanaConnectionManager struct {
	chain *solana
}

func (m *solanaConnectionManager) CheckConnection(ctx context.Context) error {
	m.chain.clientMutex.RLock()
	client := m.chain.client
	m.chain.clientMutex.RUnlock()

	if client == nil {
		return errors.New("client not initialized")
	}

	// TODO: Implement actual connection check
	return nil
}

func (m *solanaConnectionManager) Reconnect(ctx context.Context) error {
	m.chain.clientMutex.Lock()
	defer m.chain.clientMutex.Unlock()

	if m.chain.client != nil {
		// TODO: Cleanup old client if needed
	}

	client := rpc.New(m.chain.config.RpcUrl)

	m.chain.client = client

	m.chain.eventHandlerMutex.Lock()
	if m.chain.eventHandler != nil {
		// TODO: Update client in event handler if needed
	}
	m.chain.eventHandlerMutex.Unlock()

	return nil
}

func (s *solana) initMonitor(ctx context.Context) error {
	s.monitorMutex.Lock()
	defer s.monitorMutex.Unlock()

	connectionManager := &solanaConnectionManager{chain: s}
	s.monitor = connectionmonitor.NewConnectionMonitor(connectionManager, s.logger, s.config.Name)
	return s.monitor.Start(ctx)
}
