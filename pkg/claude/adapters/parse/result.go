package parse

import (
	"errors"

	"github.com/conneroisu/claude/pkg/claude/messages"
)

func (*Adapter) parseResult(data map[string]any) (messages.Message, error) {
	resultData, ok := data["result"].(map[string]any)
	if !ok {
		return nil, errors.New("result message missing result field")
	}

	errorType, hasError := getStringField(resultData, "error_type", false)
	if hasError != nil || errorType == "" {
		// Success result
		sessionID, _ := getStringField(resultData, "session_id", false)

		return &messages.ResultMessageSuccess{
			SessionID: sessionID,
		}, nil
	}

	// Error result
	sessionID, _ := getStringField(resultData, "session_id", false)
	errorMsg, _ := getStringField(resultData, "error_message", false)

	return &messages.ResultMessageError{
		SessionID:    sessionID,
		ErrorType:    errorType,
		ErrorMessage: errorMsg,
	}, nil
}
