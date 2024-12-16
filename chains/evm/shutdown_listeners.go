package evm

// ShutdownListeners stops all active subscriptions and event handlers.
func (e *evm) ShutdownListeners() {
	e.eventHandlerMutex.Lock()
	defer e.eventHandlerMutex.Unlock()

	if e.eventHandler != nil {
		e.eventHandler.Stop()
		e.eventHandler = nil
	}

	e.monitor.Stop()
}
