package connectionmonitor

import (
	"context"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	// healthCheckInterval defines interval between connection health checks
	healthCheckInterval = 30 * time.Second
	// reconnectTimeout defines timeout for reconnection attempts
	reconnectTimeout = 5 * time.Second
	// maxReconnectAttempts defines maximum number of reconnection attempts
	maxReconnectAttempts = 3
)

// ConnectionMonitor represents connection state monitoring interface
type ConnectionMonitor interface {
	// Start starts connection monitoring
	Start(ctx context.Context) error
	// Stop stops connection monitoring
	Stop()
}

// BlockchainClient represents blockchain client interface
type BlockchainClient interface {
	// CheckConnection checks if connection is alive
	CheckConnection(ctx context.Context) error
	// Reconnect attempts to reconnect to blockchain node
	Reconnect(ctx context.Context) error
}

type connectionMonitor struct {
	client       BlockchainClient
	logger       *logrus.Logger
	chainName    string
	stopChan     chan struct{}
	isMonitoring bool
	monitorMutex sync.RWMutex
}

// NewConnectionMonitor creates a new connection monitor instance.
//
// Parameters:
// - client: the blockchain client to monitor.
// - logger: the logger for logging purposes.
// - chainName: the name of the blockchain chain.
//
// Returns:
// - ConnectionMonitor: the new connection monitor instance.
func NewConnectionMonitor(
	client BlockchainClient,
	logger *logrus.Logger,
	chainName string,
) ConnectionMonitor {
	return &connectionMonitor{
		client:       client,
		logger:       logger,
		chainName:    chainName,
		stopChan:     make(chan struct{}),
		isMonitoring: false,
	}
}

// Start starts connection monitoring.
//
// Parameters:
// - ctx: the context for managing the request.
//
// Returns:
// - error: an error if the connection monitor is already running.
func (m *connectionMonitor) Start(ctx context.Context) error {
	m.monitorMutex.Lock()
	if m.isMonitoring {
		m.monitorMutex.Unlock()
		return errors.Errorf("connection monitor is already running for chain %s", m.chainName)
	}
	m.isMonitoring = true
	m.monitorMutex.Unlock()

	go m.monitorConnection(ctx)
	return nil
}

// Stop stops connection monitoring.
func (m *connectionMonitor) Stop() {
	m.monitorMutex.Lock()
	defer m.monitorMutex.Unlock()

	if !m.isMonitoring {
		return
	}

	close(m.stopChan)
	m.isMonitoring = false
}

// monitorConnection monitors the connection state and attempts to reconnect if needed.
//
// Parameters:
// - ctx: the context for managing the request.
func (m *connectionMonitor) monitorConnection(ctx context.Context) {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.WithField("chain", m.chainName).Info("Connection monitoring stopped due to context cancellation")
			return

		case <-m.stopChan:
			m.logger.WithField("chain", m.chainName).Info("Connection monitoring stopped")
			return

		case <-ticker.C:
			if err := m.checkAndReconnect(ctx); err != nil {
				m.logger.WithFields(logrus.Fields{
					"chain": m.chainName,
					"error": err,
				}).Error("Failed to check or reconnect")
			}
		}
	}
}

// checkAndReconnect checks the connection state and attempts to reconnect if needed.
//
// Parameters:
// - ctx: the context for managing the request.
//
// Returns:
// - error: an error if the reconnection fails.
func (m *connectionMonitor) checkAndReconnect(ctx context.Context) error {
	// Check connection
	if err := m.client.CheckConnection(ctx); err != nil {
		m.logger.WithFields(logrus.Fields{
			"chain": m.chainName,
			"error": err,
		}).Warn("Connection check failed, attempting to reconnect")

		// Attempt to reconnect with retry logic
		for attempt := 1; attempt <= maxReconnectAttempts; attempt++ {
			if err := m.client.Reconnect(ctx); err != nil {
				m.logger.WithFields(logrus.Fields{
					"chain":   m.chainName,
					"attempt": attempt,
					"error":   err,
				}).Error("Reconnection attempt failed")

				if attempt == maxReconnectAttempts {
					return errors.Wrapf(err, "failed to reconnect to chain %s", m.chainName)
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(reconnectTimeout):
					continue
				}
			}

			m.logger.WithFields(logrus.Fields{
				"chain":   m.chainName,
				"attempt": attempt,
			}).Info("Client successfully reconnected")
			return nil
		}
	}

	m.logger.WithField("chain", m.chainName).Info("Ping successful")

	return nil
}
