package utils

// LamportsToSol converts lamports (uint64) to SOL (float64)
func LamportsToSol(lamports uint64) float64 {
	return float64(lamports) / 1e9
}

// SolToLamports converts SOL (float64) to lamports (uint64)
func SolToLamports(sol float64) uint64 {
	return uint64(sol * 1e9)
}
