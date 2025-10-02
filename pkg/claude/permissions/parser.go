// Package permissions provides functionality for parsing and handling
// permission updates from Claude API responses, including permission rules,
// behaviors, modes, and directory restrictions.
package permissions

import "github.com/conneroisu/claude/pkg/claude/options"

// parseSuggestions converts raw suggestions into PermissionUpdate slice.
func parseSuggestions(
	suggestions []any,
) []PermissionUpdate {
	var parsedSuggestions []PermissionUpdate
	for _, sug := range suggestions {
		if sugMap, ok := sug.(map[string]any); ok {
			parsedSuggestions = append(
				parsedSuggestions,
				parsePermissionUpdate(sugMap),
			)
		}
	}

	return parsedSuggestions
}

// parsePermissionUpdate parses a single permission update from raw data.
func parsePermissionUpdate(data map[string]any) PermissionUpdate {
	update := PermissionUpdate{}

	parseUpdateType(&update, data)
	parseRules(&update, data)
	parseBehavior(&update, data)
	parseMode(&update, data)
	parseDirectories(&update, data)
	parseDestination(&update, data)

	return update
}

// parseUpdateType extracts the type field from raw data.
func parseUpdateType(update *PermissionUpdate, data map[string]any) {
	if updateType, ok := data["type"].(string); ok {
		update.Type = updateType
	}
}

// parseRules extracts and parses permission rules from raw data.
func parseRules(update *PermissionUpdate, data map[string]any) {
	rulesData, ok := data["rules"].([]any)
	if !ok {
		return
	}

	for _, ruleData := range rulesData {
		ruleMap, ok := ruleData.(map[string]any)
		if !ok {
			continue
		}

		rule := parsePermissionRule(ruleMap)
		update.Rules = append(update.Rules, rule)
	}
}

// parsePermissionRule parses a single permission rule.
func parsePermissionRule(ruleMap map[string]any) PermissionRuleValue {
	toolName, _ := ruleMap["toolName"].(string)
	var ruleContent *string
	if rc, ok := ruleMap["ruleContent"].(string); ok {
		ruleContent = &rc
	}

	return PermissionRuleValue{
		ToolName:    toolName,
		RuleContent: ruleContent,
	}
}

// parseBehavior extracts the behavior field from raw data.
func parseBehavior(update *PermissionUpdate, data map[string]any) {
	if behaviorStr, ok := data["behavior"].(string); ok {
		behavior := PermissionBehavior(behaviorStr)
		update.Behavior = &behavior
	}
}

// parseMode extracts the mode field from raw data.
func parseMode(update *PermissionUpdate, data map[string]any) {
	if modeStr, ok := data["mode"].(string); ok {
		mode := options.PermissionMode(modeStr)
		update.Mode = &mode
	}
}

// parseDirectories extracts the directories field from raw data.
func parseDirectories(update *PermissionUpdate, data map[string]any) {
	dirsData, ok := data["directories"].([]any)
	if !ok {
		return
	}

	for _, dirData := range dirsData {
		dir, ok := dirData.(string)
		if !ok {
			continue
		}
		update.Directories = append(update.Directories, dir)
	}
}

// parseDestination extracts the destination field from raw data.
func parseDestination(update *PermissionUpdate, data map[string]any) {
	if destStr, ok := data["destination"].(string); ok {
		dest := PermissionUpdateDestination(destStr)
		update.Destination = &dest
	}
}
