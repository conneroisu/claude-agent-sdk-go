package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/conneroisu/claude-agent-sdk-go/pkg/clauderrs"
)

// GetServerInfo returns the initialization result stored during Initialize.
func (q *queryImpl) GetServerInfo() (map[string]any, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.initializationResult == nil {
		return nil, clauderrs.NewClientError(clauderrs.ErrCodeInvalidState, "query not initialized", nil).
			WithSessionID(q.sessionID)
	}

	return q.initializationResult, nil
}

// Initialize sends initialize control request and stores the response.
// This should be called if bidirectional control protocol is needed.
func (q *queryImpl) Initialize(ctx context.Context) (map[string]any, error) {
	// Build hooks configuration from opts.Hooks
	var hooksConfig map[string]JSONValue
	if len(q.opts.Hooks) > 0 {
		hooksConfig = make(map[string]JSONValue)

		for event, matchers := range q.opts.Hooks {
			if len(matchers) == 0 {
				continue
			}

			// Build array of hook matchers for this event
			matcherConfigs := make([]map[string]any, 0, len(matchers))
			for _, matcher := range matchers {
				// Register each callback and collect their IDs
				callbackIDs := make([]string, 0, len(matcher.Hooks))
				for _, callback := range matcher.Hooks {
					callbackID := fmt.Sprintf("hook_%d", q.nextCallbackID)
					q.nextCallbackID++
					q.hookCallbacks[callbackID] = callback
					callbackIDs = append(callbackIDs, callbackID)
				}

				// Build matcher config
				matcherConfig := map[string]any{
					"hookCallbackIds": callbackIDs,
				}
				if matcher.Matcher != nil {
					matcherConfig["matcher"] = *matcher.Matcher
				}
				matcherConfigs = append(matcherConfigs, matcherConfig)
			}

			// Marshal to JSONValue
			matcherBytes, err := json.Marshal(matcherConfigs)
			if err != nil {
				return nil, clauderrs.NewProtocolError(
					clauderrs.ErrCodeMessageParseFailed,
					fmt.Sprintf("failed to marshal hook matchers for event %s", event),
					err,
				).
					WithSessionID(q.sessionID).
					WithMessageType("initialize")
			}
			hooksConfig[string(event)] = matcherBytes
		}
	}

	resp, err := q.sendControlRequest(ctx, SDKControlInitializeRequest{
		Hooks: hooksConfig,
	})
	if err != nil {
		return nil, err
	}

	q.mu.Lock()
	q.initializationResult = resp
	q.mu.Unlock()

	return resp, nil
}
