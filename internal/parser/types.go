package parser

// RawEvent represents a single line from a Claude Code JSONL log file.
type RawEvent struct {
	Type      string     `json:"type"`
	SessionID string     `json:"sessionId"`
	Slug      string     `json:"slug"`
	RequestID string     `json:"requestId"`
	Timestamp string     `json:"timestamp"`
	UUID      string     `json:"uuid"`
	Message   RawMessage `json:"message"`
}

type RawMessage struct {
	Role    string `json:"role"`
	Model   string `json:"model"`
	Usage   Usage  `json:"usage"`
	Content any    `json:"content"` // can be string or []ContentBlock
}

type Usage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// SessionInfo holds metadata extracted from the file path.
type SessionInfo struct {
	SessionID  string
	Project    string
	IsSubagent bool
}

// ParsedUsage is the aggregated token usage for a single request (after dedup).
type ParsedUsage struct {
	Model            string
	Slug             string
	SessionID        string
	Timestamp        string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
}
