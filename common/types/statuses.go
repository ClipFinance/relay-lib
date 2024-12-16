package types

type IntentStatus string

const (
	StatusCreated IntentStatus = "CREATED"
	// StatusPending is the status of an intent when it is pending.
	StatusPending IntentStatus = "PENDING"
	// StatusDone is the status of an intent when it is done.
	StatusDone IntentStatus = "DONE"
	// StatusFailed is the status of an intent when it is failed.
	StatusFailed IntentStatus = "FAILED"
)

type SubStatus string

const (
	// WaitSourceConfirmations indicates that the bridge is waiting for additional confirmations.
	WaitSourceConfirmations SubStatus = "WAIT_SOURCE_CONFIRMATIONS"

	// WaitDestinationTransaction indicates that the off-chain logic is in progress, waiting for the destination transaction to be mined.
	WaitDestinationTransaction SubStatus = "WAIT_DESTINATION_TRANSACTION"

	// BridgeNotAvailable indicates that the bridge API or subgraph is temporarily unavailable.
	BridgeNotAvailable SubStatus = "BRIDGE_NOT_AVAILABLE"

	// ChainNotAvailable indicates that the RPC for the source or destination chain is temporarily unavailable.
	ChainNotAvailable SubStatus = "CHAIN_NOT_AVAILABLE"

	// RefundInProgress indicates that the refund has been requested and is being processed.
	RefundInProgress SubStatus = "REFUND_IN_PROGRESS"

	// UnknownError indicates that the status of the transfer cannot be determined.
	UnknownError SubStatus = "UNKNOWN_ERROR"

	// Completed indicates that the transfer was successful.
	Completed SubStatus = "COMPLETED"

	// Partial indicates that the transfer was partially successful with alternative tokens provided.
	// This occurs with bridges like Across, Hop, Stargate, or Amarok in case of low liquidity.
	Partial SubStatus = "PARTIAL"

	// Refunded indicates that the transfer was not successful and tokens were refunded.
	Refunded SubStatus = "REFUNDED"

	// NotProcessableRefundNeeded indicates that the transfer cannot be completed and requires a refund.
	NotProcessableRefundNeeded SubStatus = "NOT_PROCESSABLE_REFUND_NEEDED"

	// OutOfGas indicates that the transaction ran out of gas during the execution.
	OutOfGas SubStatus = "OUT_OF_GAS"

	// SlippageExceeded indicates that the return amount is below the slippage limit.
	SlippageExceeded SubStatus = "SLIPPAGE_EXCEEDED"

	// InsufficientAllowance indicates that the transfer amount exceeds the token allowance.
	InsufficientAllowance SubStatus = "INSUFFICIENT_ALLOWANCE"

	// InsufficientBalance indicates that the transfer amount exceeds the available balance.
	InsufficientBalance SubStatus = "INSUFFICIENT_BALANCE"

	// Expired indicates that the transaction expired before processing.
	Expired SubStatus = "EXPIRED"

	// RefundFailed indicates that the refund process failed.
	RefundFailed SubStatus = "REFUND_FAILED"

	// RefundInsufficientBalance indicates that the refund process failed due to insufficient balance.
	RefundInsufficientBalance SubStatus = "REFUND_INSUFFICIENT_BALANCE"

	// RefundChainNotAvailable indicates that the refund process failed due to the chain being unavailable.
	RefundChainNotAvailable SubStatus = "REFUND_CHAIN_NOT_AVAILABLE"
)
