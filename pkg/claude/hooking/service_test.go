package hooking_test

import (
	"context"
	"testing"

	"github.com/conneroisu/claude/pkg/claude/hooking"
)

func TestService_Execute(t *testing.T) {
	callbackInvoked := false
	callback := func(input map[string]any, toolUseID *string, ctx hooking.HookContext) (map[string]any, error) {
		callbackInvoked = true
		return map[string]any{"modified": true}, nil
	}

	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		hooking.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []hooking.HookCallback{callback},
			},
		},
	}

	service := hooking.NewService(hooks)

	input := map[string]any{
		"tool_name": "Bash",
		"tool_input": map[string]any{
			"command": "ls",
		},
	}

	result, err := service.Execute(
		context.Background(),
		hooking.HookEventPreToolUse,
		input,
		nil,
	)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !callbackInvoked {
		t.Error("Execute() callback was not invoked")
	}

	if modified, ok := result["modified"].(bool); !ok || !modified {
		t.Error("Execute() did not return expected modified result")
	}
}

func TestService_Execute_NoMatchingHook(t *testing.T) {
	callback := func(input map[string]any, toolUseID *string, ctx hooking.HookContext) (map[string]any, error) {
		return map[string]any{"modified": true}, nil
	}

	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		hooking.HookEventPreToolUse: {
			{
				Matcher: "Write",
				Hooks:   []hooking.HookCallback{callback},
			},
		},
	}

	service := hooking.NewService(hooks)

	input := map[string]any{
		"tool_name": "Bash",
		"tool_input": map[string]any{
			"command": "ls",
		},
	}

	result, err := service.Execute(
		context.Background(),
		hooking.HookEventPreToolUse,
		input,
		nil,
	)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Should return empty result when no hooks match
	if len(result) > 0 {
		t.Error("Execute() should return empty result when no hooks match")
	}
}

func TestService_Execute_MultipleHooks(t *testing.T) {
	var callOrder []string

	hook1 := func(input map[string]any, toolUseID *string, ctx hooking.HookContext) (map[string]any, error) {
		callOrder = append(callOrder, "hook1")
		return map[string]any{"hook1": true}, nil
	}

	hook2 := func(input map[string]any, toolUseID *string, ctx hooking.HookContext) (map[string]any, error) {
		callOrder = append(callOrder, "hook2")
		return map[string]any{"hook2": true}, nil
	}

	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		hooking.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []hooking.HookCallback{hook1, hook2},
			},
		},
	}

	service := hooking.NewService(hooks)

	input := map[string]any{
		"tool_name": "Bash",
	}

	result, err := service.Execute(
		context.Background(),
		hooking.HookEventPreToolUse,
		input,
		nil,
	)

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(callOrder) != 2 {
		t.Errorf("Execute() called %d hooks, want 2", len(callOrder))
	}

	if callOrder[0] != "hook1" || callOrder[1] != "hook2" {
		t.Errorf("Execute() call order = %v, want [hook1, hook2]", callOrder)
	}

	// Both hook outputs should be in result
	if _, ok := result["hook1"]; !ok {
		t.Error("Execute() result missing hook1 output")
	}
	if _, ok := result["hook2"]; !ok {
		t.Error("Execute() result missing hook2 output")
	}
}

func TestService_GetHooks(t *testing.T) {
	hooks := map[hooking.HookEvent][]hooking.HookMatcher{
		hooking.HookEventPreToolUse: {
			{
				Matcher: "Bash",
				Hooks:   []hooking.HookCallback{func(map[string]any, *string, hooking.HookContext) (map[string]any, error) { return nil, nil }},
			},
		},
	}

	service := hooking.NewService(hooks)
	retrieved := service.GetHooks()

	if len(retrieved) != 1 {
		t.Errorf("GetHooks() returned %d events, want 1", len(retrieved))
	}

	if _, ok := retrieved[hooking.HookEventPreToolUse]; !ok {
		t.Error("GetHooks() missing PreToolUse event")
	}
}

func stringPtr(s string) *string {
	return &s
}
