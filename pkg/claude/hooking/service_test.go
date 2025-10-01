package hooking

import (
	"context"
	"errors"
	"testing"
)

func TestNewService(t *testing.T) {
	t.Run("creates service with hooks", func(t *testing.T) {
		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "test",
					Hooks:   []HookCallback{},
				},
			},
		}

		svc := NewService(hooks)
		if svc == nil {
			t.Fatal("Expected service to be created")
		}
		if svc.hooks == nil {
			t.Error("Expected hooks to be initialized")
		}
	})

	t.Run("creates service with nil hooks", func(t *testing.T) {
		svc := NewService(nil)
		if svc == nil {
			t.Fatal("Expected service to be created")
		}
	})
}

func TestServiceGetHooks(t *testing.T) {
	t.Run("returns hooks", func(t *testing.T) {
		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{Matcher: "test", Hooks: []HookCallback{}},
			},
		}

		svc := NewService(hooks)
		result := svc.GetHooks()

		if result == nil {
			t.Fatal("Expected hooks to be returned")
		}
		if len(result) != 1 {
			t.Errorf("Expected 1 hook event, got %d", len(result))
		}
	})

	t.Run("returns nil for nil service", func(t *testing.T) {
		var svc *Service
		result := svc.GetHooks()

		if result != nil {
			t.Error("Expected nil hooks for nil service")
		}
	})
}

func TestServiceExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("executes hook successfully", func(t *testing.T) {
		called := false
		hook := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			called = true

			return map[string]any{"result": "success"}, nil
		}

		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "test",
					Hooks:   []HookCallback{hook},
				},
			},
		}

		svc := NewService(hooks)
		result, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !called {
			t.Error("Expected hook to be called")
		}
		if result == nil {
			t.Fatal("Expected result to be returned")
		}
		if val, ok := result["result"]; !ok || val != "success" {
			t.Error("Expected result to contain 'result': 'success'")
		}
	})

	t.Run("returns nil for nil service", func(t *testing.T) {
		var svc *Service
		result, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != nil {
			t.Error("Expected nil result for nil service")
		}
	})

	t.Run("returns nil for non-existent event", func(t *testing.T) {
		svc := NewService(map[HookEvent][]HookMatcher{})
		result, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != nil {
			t.Error("Expected nil result for non-existent event")
		}
	})

	t.Run("returns error when hook fails", func(t *testing.T) {
		hookErr := errors.New("hook failed")
		hook := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return nil, hookErr
		}

		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "test",
					Hooks:   []HookCallback{hook},
				},
			},
		}

		svc := NewService(hooks)
		_, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err == nil {
			t.Error("Expected error, got nil")
		}
		if !errors.Is(err, hookErr) {
			t.Errorf("Expected hook error to be wrapped, got %v", err)
		}
	})

	t.Run("stops on block decision", func(t *testing.T) {
		firstCalled := false
		secondCalled := false

		firstHook := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			firstCalled = true

			return map[string]any{"decision": "block", "reason": "blocked"}, nil
		}

		secondHook := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			secondCalled = true
			return map[string]any{"result": "should not execute"}, nil
		}

		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "test",
					Hooks:   []HookCallback{firstHook, secondHook},
				},
			},
		}

		svc := NewService(hooks)
		result, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !firstCalled {
			t.Error("Expected first hook to be called")
		}
		if secondCalled {
			t.Error("Expected second hook NOT to be called after block")
		}
		if decision, ok := result["decision"]; !ok || decision != "block" {
			t.Error("Expected block decision in result")
		}
	})

	t.Run("aggregates results from multiple hooks", func(t *testing.T) {
		hook1 := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return map[string]any{"key1": "value1"}, nil
		}

		hook2 := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return map[string]any{"key2": "value2"}, nil
		}

		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "test",
					Hooks:   []HookCallback{hook1, hook2},
				},
			},
		}

		svc := NewService(hooks)
		result, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if val, ok := result["key1"]; !ok || val != "value1" {
			t.Error("Expected key1='value1' in aggregated result")
		}
		if val, ok := result["key2"]; !ok || val != "value2" {
			t.Error("Expected key2='value2' in aggregated result")
		}
	})

	t.Run("later hooks override earlier results", func(t *testing.T) {
		hook1 := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return map[string]any{"key": "first"}, nil
		}

		hook2 := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return map[string]any{"key": "second"}, nil
		}

		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "test",
					Hooks:   []HookCallback{hook1, hook2},
				},
			},
		}

		svc := NewService(hooks)
		result, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if val, ok := result["key"]; !ok || val != "second" {
			t.Errorf("Expected key='second' (overridden), got %v", val)
		}
	})

	t.Run("handles nil result from hook", func(t *testing.T) {
		hook := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return nil, nil
		}

		hooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{
					Matcher: "test",
					Hooks:   []HookCallback{hook},
				},
			},
		}

		svc := NewService(hooks)
		result, err := svc.Execute(ctx, HookEventPreToolUse, map[string]any{}, nil)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Error("Expected empty result map for nil hook result")
		}
	})
}

func TestServiceRegister(t *testing.T) {
	t.Run("registers new hook", func(t *testing.T) {
		svc := NewService(nil)

		hook := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return map[string]any{}, nil
		}

		matcher := HookMatcher{
			Matcher: "test",
			Hooks:   []HookCallback{hook},
		}

		svc.Register(HookEventPreToolUse, matcher)

		hooks := svc.GetHooks()
		if len(hooks[HookEventPreToolUse]) != 1 {
			t.Error("Expected hook to be registered")
		}
	})

	t.Run("appends to existing hooks", func(t *testing.T) {
		hook1 := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return map[string]any{}, nil
		}

		initialHooks := map[HookEvent][]HookMatcher{
			HookEventPreToolUse: {
				{Matcher: "first", Hooks: []HookCallback{hook1}},
			},
		}

		svc := NewService(initialHooks)

		hook2 := func(input map[string]any, toolUseID *string, ctx HookContext) (map[string]any, error) {
			return map[string]any{}, nil
		}

		matcher := HookMatcher{
			Matcher: "second",
			Hooks:   []HookCallback{hook2},
		}

		svc.Register(HookEventPreToolUse, matcher)

		hooks := svc.GetHooks()
		if len(hooks[HookEventPreToolUse]) != 2 {
			t.Errorf("Expected 2 matchers, got %d", len(hooks[HookEventPreToolUse]))
		}
	})
}

func TestHookEvents(t *testing.T) {
	events := []HookEvent{
		HookEventPreToolUse,
		HookEventPostToolUse,
		HookEventUserPromptSubmit,
		HookEventStop,
		HookEventSubagentStop,
		HookEventPreCompact,
	}

	for _, event := range events {
		t.Run(string(event), func(t *testing.T) {
			if event == "" {
				t.Error("Hook event should not be empty")
			}
		})
	}
}
