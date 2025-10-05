package streaming

// Close terminates the streaming session and releases all resources.
// It is safe to call multiple times.
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.connected {
		return nil
	}

	s.connected = false

	return s.transport.Close()
}
