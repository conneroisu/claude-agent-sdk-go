//nolint:revive // comments-density: code is self-documenting
package querying

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// streamMessages reads and routes messages from the protocol
//nolint:revive // argument-limit: all parameters required
// adapter.
func (s *Service) streamMessages(
	ctx context.Context,
	routerMsgCh <-chan map[string]any,
	routerErrCh <-chan error,
	msgCh chan<- messages.Message,
	_ chan<- error,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case msg, ok := <-routerMsgCh:
			if !ok {
				return nil
			}

			if err := s.handleMessage(
				msg,
				msgCh,
			); err != nil {
				return err
			}

		case err := <-routerErrCh:
			if err != nil {
				return err
			}
		}
	}
}

// handleMessage parses and sends a message.
func (s *Service) handleMessage(
	raw map[string]any,
	msgCh chan<- messages.Message,
) error {
	parsedMsg, err := s.parser.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse message: %w", err)
	}

	msgCh <- parsedMsg

	return nil
}
