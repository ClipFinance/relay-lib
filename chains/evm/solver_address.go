package evm

// SolverAddress returns the solver address for the chain.
//
// Returns:
// - string: the solver address for fetching balances.
func (e *evm) SolverAddress() string {
	e.solverAddressMutex.RLock()
	defer e.solverAddressMutex.RUnlock()
	return e.solverAddress.Hex()
}

