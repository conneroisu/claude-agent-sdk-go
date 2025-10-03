package messages

// UsageStats represents aggregated API usage statistics.
type UsageStats struct {
	// InputTokens is count of input tokens used
	InputTokens int `json:"input_tokens"`

	// OutputTokens is count of output tokens generated
	OutputTokens int `json:"output_tokens"`

	// CacheReadInputTokens is count of cached input tokens read
	CacheReadInputTokens int `json:"cache_read_input_tokens"`

	// CacheCreationInputTokens is count of input tokens
	// used for cache creation
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ModelUsage represents usage statistics for a specific model.
type ModelUsage struct {
	// InputTokens is count of input tokens used
	InputTokens int `json:"inputTokens"`

	// OutputTokens is count of output tokens generated
	OutputTokens int `json:"outputTokens"`

	// CacheReadInputTokens is count of cached input tokens read
	CacheReadInputTokens int `json:"cacheReadInputTokens"`

	// CacheCreationInputTokens is count of input tokens
	// used for cache creation
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`

	// WebSearchRequests is count of web search API calls made
	WebSearchRequests int `json:"webSearchRequests"`

	// CostUSD is the cost in US dollars for this model's usage
	CostUSD float64 `json:"costUSD"`

	// ContextWindow is the maximum context size for this model
	ContextWindow int `json:"contextWindow"`
}
