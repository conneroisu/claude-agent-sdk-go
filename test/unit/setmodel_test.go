package unit

import (
	"encoding/json"
	"testing"
)

// TestSetModelRequestSerializationOldBehavior tests the OLD behavior where
// the model pointer is added directly, causing nil to serialize as null.
func TestSetModelRequestSerializationOldBehavior(t *testing.T) {
	t.Run("Old behavior with nil model - serializes to null (BAD)", func(t *testing.T) {
		// OLD implementation: directly assign pointer
		request := map[string]any{
			"subtype": "setModel",
			"model":   (*string)(nil), // This will serialize to null
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		jsonStr := string(jsonData)
		t.Logf("OLD behavior JSON: %s", jsonStr)

		// This shows the problem: the JSON contains "model":null
		var parsed map[string]any
		if err := json.Unmarshal(jsonData, &parsed); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v", err)
		}

		// The model key exists but has nil value
		modelValue, hasModel := parsed["model"]
		if !hasModel {
			t.Error("Expected 'model' field to be present in old behavior")
		}
		if modelValue != nil {
			t.Errorf("Expected model value to be nil, got %v", modelValue)
		}
		t.Logf("OLD behavior result: 'model' key exists=%v, value=%v", hasModel, modelValue)
	})
}

// TestSetModelRequestSerializationNewBehavior tests the NEW behavior where
// we conditionally add the model field only when non-nil.
func TestSetModelRequestSerializationNewBehavior(t *testing.T) {
	tests := []struct {
		name          string
		model         *string
		shouldContain bool
		expectedValue string
	}{
		{
			name:          "With nil model - should omit field",
			model:         nil,
			shouldContain: false,
		},
		{
			name:          "With non-nil model - should include dereferenced value",
			model:         stringPtr("claude-3-5-sonnet-20241022"),
			shouldContain: true,
			expectedValue: "claude-3-5-sonnet-20241022",
		},
		{
			name:          "With empty string model - should include empty string",
			model:         stringPtr(""),
			shouldContain: true,
			expectedValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NEW implementation: only add model key if model is not nil
			request := map[string]any{
				"subtype": "setModel",
			}

			if tt.model != nil {
				request["model"] = *tt.model
			}

			jsonData, err := json.Marshal(request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			jsonStr := string(jsonData)
			t.Logf("NEW behavior JSON: %s", jsonStr)

			var parsed map[string]any
			if err := json.Unmarshal(jsonData, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			modelValue, hasModel := parsed["model"]

			if tt.shouldContain {
				if !hasModel {
					t.Errorf("Expected 'model' field to be present, but it was missing")
				}
				if modelValue != tt.expectedValue {
					t.Errorf("Expected model value to be %q, got %v", tt.expectedValue, modelValue)
				}
			} else {
				if hasModel {
					t.Errorf("Expected 'model' field to be omitted, but it was present with value: %v", modelValue)
				}
			}
		})
	}
}
