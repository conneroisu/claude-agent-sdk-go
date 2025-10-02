// Package querying provides message streaming functionality.
package querying

import (
	"context"
	"fmt"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

// streamChannels groups all channels used for streaming.
type streamChannels struct {
	routerMsg chan map[string]any
	routerErr chan error
	msg       chan messages.Message
	err       chan error
}

// streamMessages handles the message streaming loop.
func (s *Service) streamMessages(
	ctx context.Context,
	ch streamChannels,
) {
	for {
		if !s.processNextMessage(ctx, ch) {
			return
		}
	}
}

// processNextMessage processes the next message from available channels.
func (s *Service) processNextMessage(
	ctx context.Context,
	ch streamChannels,
) bool {
	select {
	case <-ctx.Done():
		return false
	case msg, ok := <-ch.routerMsg:
		if !ok {
			return false
		}

		return s.handleRouterMessage(msg, ch)
	case err := <-ch.routerErr:
		return s.handleRouterError(err, ch)
	}
}

// handleRouterMessage handles a message from the router channel.
func (s *Service) handleRouterMessage(
	msg map[string]any,
	ch streamChannels,
) bool {
	parsedMsg, err := s.parser.Parse(msg)
	if err != nil {
		ch.err <- fmt.Errorf("parse message: %w", err)

		return false
	}
	ch.msg <- parsedMsg

	return true
}

// handleRouterError handles an error from the router channel.
func (*Service) handleRouterError(err error, ch streamChannels) bool {
	if err != nil {
		ch.err <- err

		return false
	}

	return true
}
