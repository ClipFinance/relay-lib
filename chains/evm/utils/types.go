package utils

const (
	// ZeroAddress represents the zero address.
	ZeroAddress = "0x0000000000000000000000000000000000000000"
)

// EvmMetadata represents the metadata for an EVM event.
type EvmMetadata struct {
	EventType string
	LogIndex  uint
	Data      []byte
}
