package claude

import "github.com/conneroisu/claude/pkg/claude/messages"

// createErrorChannels creates pre-populated error channels.
// Used when initialization fails before query execution.
func createErrorChannels(err error) (<-chan messages.Message, <-chan error) {
	msgCh := make(chan messages.Message)
	errCh := make(chan error, 1)
	errCh <- err
	close(msgCh)
	close(errCh)

	return msgCh, errCh
}
