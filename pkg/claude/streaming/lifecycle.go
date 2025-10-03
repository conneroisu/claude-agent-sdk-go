package streaming

// Close disconnects from Claude.
func (s *Service) Close() error {
	if s.transport != nil {
		return s.transport.Close()
	}

	return nil
}
