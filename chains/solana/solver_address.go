package solana

// SolverAddress returns the solver address for the chain.
//
// Returns:
// - string: the solver address for fetching balances.
func (s *solana) SolverAddress() string {
	s.solverAddressMutex.RLock()
	defer s.solverAddressMutex.RUnlock()
	return s.solverAddress
}
