package types

import "strings"

// SubscriptionMode defines the RPC connection type
type SubscriptionMode int

const (
	WebSocketMode SubscriptionMode = iota
	HTTPPollingMode
)

// GetSubscriptionMode returns mode based on RPC URL
func GetSubscriptionMode(rpcURL string) SubscriptionMode {
	if strings.HasPrefix(rpcURL, "wss://") || strings.HasPrefix(rpcURL, "ws://") {
		return WebSocketMode
	}
	return HTTPPollingMode
}

func (m SubscriptionMode) String() string {
	switch m {
	case WebSocketMode:
		return "WebSocket"
	case HTTPPollingMode:
		return "HTTP"
	default:
		return "Unknown"
	}
}
