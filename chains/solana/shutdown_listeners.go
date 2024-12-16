package solana

// ShutdownListeners stops all active subscriptions and event handlers.
func (s *solana) ShutdownListeners() {
	s.eventHandlerMutex.Lock()
	defer s.eventHandlerMutex.Unlock()

	if s.eventHandler != nil {
		// TODO: Implement the Stop method for the event handler.
		s.eventHandler = nil
	}
}
