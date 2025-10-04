package messages

// UsageStats represents API usage statistics.
// This tracks token consumption across different categories.
type UsageStats struct {
	// InputTokens is the number of input tokens consumed
	InputTokens int `json:"input_tokens"`

	// OutputTokens is the number of output tokens generated
	OutputTokens int `json:"output_tokens"`

	// CacheReadInputTokens is tokens read from cache
	CacheReadInputTokens int `json:"cache_read_input_tokens"`

	// CacheCreationInputTokens is tokens written to cache
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ModelUsage represents usage statistics for a specific model.
// This provides detailed usage breakdown including cost and limits.
type ModelUsage struct {
	// InputTokens is the number of input tokens consumed
	InputTokens int `json:"inputTokens"`

	// OutputTokens is the number of output tokens generated
	OutputTokens int `json:"outputTokens"`

	// CacheReadInputTokens is tokens read from cache
	CacheReadInputTokens int `json:"cacheReadInputTokens"`

	// CacheCreationInputTokens is tokens written to cache
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`

	// WebSearchRequests is the number of web searches performed
	WebSearchRequests int `json:"webSearchRequests"`

	// CostUSD is the cost in US dollars for this model
	CostUSD float64 `json:"costUSD"`

	// ContextWindow is the maximum context size for this model
	ContextWindow int `json:"contextWindow"`
}
