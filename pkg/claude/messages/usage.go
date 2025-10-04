package messages

// UsageStats represents aggregated API usage statistics.
//
// Contains token counts for input and output, including cache-related
// tokens. Used to track resource consumption across API calls.
//
// Example:
//
//	usage := UsageStats{
//	    InputTokens: 1000,
//	    OutputTokens: 500,
//	    CacheReadInputTokens: 200,
//	    CacheCreationInputTokens: 50,
//	}
type UsageStats struct {
	// InputTokens is the total number of input tokens.
	InputTokens int `json:"input_tokens"`

	// OutputTokens is the total number of output tokens.
	OutputTokens int `json:"output_tokens"`

	// CacheReadInputTokens is the number of input tokens read from cache.
	CacheReadInputTokens int `json:"cache_read_input_tokens"`

	// CacheCreationInputTokens is the number of tokens for cache entries.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

// ModelUsage represents usage statistics for a specific model.
//
// Contains detailed usage metrics including token counts, cost,
// web search requests, and context window size. Used to track
// per-model resource consumption and costs.
//
// Example:
//
//	usage := ModelUsage{
//	    InputTokens: 1000,
//	    OutputTokens: 500,
//	    CostUSD: 0.025,
//	    ContextWindow: 200000,
//	}
type ModelUsage struct {
	// InputTokens is the number of input tokens for this model.
	InputTokens int `json:"inputTokens"`

	// OutputTokens is the number of output tokens for this model.
	OutputTokens int `json:"outputTokens"`

	// CacheReadInputTokens is the number of input tokens read from cache.
	CacheReadInputTokens int `json:"cacheReadInputTokens"`

	// CacheCreationInputTokens is the number of tokens for cache entries.
	CacheCreationInputTokens int `json:"cacheCreationInputTokens"`

	// WebSearchRequests is the number of web search API calls made.
	WebSearchRequests int `json:"webSearchRequests"`

	// CostUSD is the estimated cost in US dollars for this model's usage.
	CostUSD float64 `json:"costUSD"`

	// ContextWindow is the maximum context window size for this model.
	ContextWindow int `json:"contextWindow"`
}
