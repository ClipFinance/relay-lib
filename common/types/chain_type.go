package types

// ChainType represents supported blockchain types
type ChainType string

const (
	// EVM represents Ethereum Virtual Machine based chains (e.g. Ethereum, Linea, Base, etc.)
	EVM ChainType = "EVM"
	// SOLANA represents Solana chain.
	SOLANA ChainType = "SOLANA"
	// UNKNOWN represents unknown or unsupported chain type in the system.
	UNKNOWN ChainType = "UNKNOWN"
)

// String converts ChainType to string representation
func (t ChainType) String() string {
	return string(t)
}

// ParseChainType converts string to ChainType representation.
func ParseChainType(s string) ChainType {
	switch s {
	case EVM.String():
		return EVM
	case SOLANA.String():
		return SOLANA
	default:
		return UNKNOWN
	}
}
